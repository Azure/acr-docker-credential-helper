package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/Sirupsen/logrus"
	jwt "github.com/dgrijalva/jwt-go"
)

type authDirective struct {
	service string
	realm   string
}

type accessTokenPayload struct {
	TenantID string `json:"tid"`
}

type acrTokenPayload struct {
	Expiration int64  `json:"exp"`
	TenantID   string `json:"tenant"`
	Credential string `json:"credential"`
}

type acrAuthResponse struct {
	RefreshToken string `json:"refresh_token"`
}

// const NullUsername = "00000000-0000-0000-0000-000000000000"

// 5 minutes buffer time to allow timeshift between local machine and AAD
const timeShiftBuffer = 300

func (token *acrTokenPayload) isExpiredOrNear() bool {
	return time.Now().Unix() > token.Expiration-timeShiftBuffer
}

func getOAuthBaseURL() *url.URL {
	return &url.URL{
		Scheme: "https",
		Host:   constAADServer,
		Path:   "common/oauth2",
	}
}

// GetUsernamePassword get the AAD based ACR login credentials
func GetUsernamePassword(serverAddress string, identityToken string) (user string, cred string, err error) {
	if identityToken == "" {
		return "", "", fmt.Errorf("Unexpected empty token. Please call 'az acr login' to generate a valid token")
	}

	var acrToken *acrTokenPayload
	acrToken, err = parseAcrToken(identityToken)
	if err != nil {
		return "", "", fmt.Errorf("Bad identity token")
	}
	refreshToken := identityToken
	if acrToken.isExpiredOrNear() {
		var challenge *authDirective
		if challenge, err = receiveChallengeFromLoginServer(serverAddress); err != nil {
			// ignore all error when receiving the challenge
			logrus.Infof("[Azure Login Helper] server %s didn't respond with a valid challenge, reverting to default login...\nerror: %s\n", serverAddress, err)
			return "", "", nil
		}
		refreshToken, err = performTokenExchange(serverAddress, challenge, acrToken.TenantID, acrToken.Credential)
	}

	return tokenUsername, refreshToken, nil
}

func receiveChallengeFromLoginServer(serverAddress string) (*authDirective, error) {
	challengeURL := url.URL{
		Scheme: "https",
		Host:   serverAddress,
		Path:   "v2/",
	}
	var err error
	var challenge *http.Response
	if challenge, err = http.Get(challengeURL.String()); err != nil {
		return nil, fmt.Errorf("Error reaching registry endpoint %s, error: %s", challengeURL.String(), err)
	}
	defer challenge.Body.Close()

	if challenge.StatusCode != 401 {
		return nil, fmt.Errorf("Registry did not issue a valid AAD challenge, status: %d", challenge.StatusCode)
	}

	var authHeader []string
	var ok bool
	if authHeader, ok = challenge.Header["Www-Authenticate"]; !ok {
		return nil, fmt.Errorf("Challenge response does not contain header 'Www-Authenticate'")
	}

	if len(authHeader) != 1 {
		return nil, fmt.Errorf("Registry did not issue a valid AAD challenge, authenticate header [%s]",
			strings.Join(authHeader, ", "))
	}

	authSections := strings.SplitN(authHeader[0], " ", 2)
	authType := strings.ToLower(authSections[0])
	var authParams *map[string]string
	if authParams, err = parseAssignments(authSections[1]); err != nil {
		return nil, fmt.Errorf("Unable to understand the contents of Www-Authenticate header %s", authSections[1])
	}

	// verify headers
	if !strings.EqualFold("Bearer", authType) {
		return nil, fmt.Errorf("Www-Authenticate: expected realm: Bearer, actual: %s", authType)
	}
	if len((*authParams)["service"]) == 0 {
		return nil, fmt.Errorf("Www-Authenticate: missing header \"service\"")
	}
	if len((*authParams)["realm"]) == 0 {
		return nil, fmt.Errorf("Www-Authenticate: missing header \"realm\"")
	}

	return &authDirective{
		service: (*authParams)["service"],
		realm:   (*authParams)["realm"],
	}, nil
}

// func getAADTokensWithDeviceLogin() (tenantID string, refreshToken string, err error) {
// 	var adalToken *adal.Token
// 	if adalToken, err = adalDeviceLogin(); err != nil {
// 		return "", "", err
// 	}

// 	accessTokenEncoded := adalToken.AccessToken
// 	accessTokenSplit := strings.Split(accessTokenEncoded, ".")
// 	if len(accessTokenSplit) < 2 {
// 		return "", "", fmt.Errorf("invalid encoded id token: %s", accessTokenEncoded)
// 	}

// 	idPayloadEncoded := accessTokenSplit[1]
// 	var idJSON []byte
// 	if idJSON, err = jwt.DecodeSegment(idPayloadEncoded); err != nil {
// 		return "", "", fmt.Errorf("Error decoding accessToken: %s", err)
// 	}

// 	var accessToken accessTokenPayload
// 	if err := json.Unmarshal(idJSON, &accessToken); err != nil {
// 		return "", "", fmt.Errorf("Error unmarshalling id token: %s", err)
// 	}

// 	return accessToken.TenantID, adalToken.RefreshToken, nil
// }

func adalDeviceLogin() (*adal.Token, error) {
	oauthClient := &http.Client{}
	authEndpoint := getOAuthBaseURL()
	authEndpoint.Path = path.Join(authEndpoint.Path, "authorize")
	tokenEndpoint := getOAuthBaseURL()
	tokenEndpoint.Path = path.Join(tokenEndpoint.Path, "token")
	deviceCodeEndpoint := getOAuthBaseURL()
	deviceCodeEndpoint.Path = path.Join(deviceCodeEndpoint.Path, "devicecode")

	var err error
	var deviceCode *adal.DeviceCode
	if deviceCode, err = adal.InitiateDeviceAuth(
		oauthClient,
		adal.OAuthConfig{
			AuthorizeEndpoint:  *authEndpoint,
			TokenEndpoint:      *tokenEndpoint,
			DeviceCodeEndpoint: *deviceCodeEndpoint,
		},
		constAppID,
		"https://management.core.windows.net/"); err != nil {
		return nil, fmt.Errorf("Failed to start device auth flow: %s", err)
	}

	fmt.Fprintf(os.Stderr, "%s\n", *deviceCode.Message)
	var token *adal.Token
	if token, err = adal.WaitForUserCompletion(oauthClient, deviceCode); err != nil {
		return nil, fmt.Errorf("Failed to finish device auth flow: %s", err)
	}
	return token, nil
}

func parseAcrToken(identityToken string) (token *acrTokenPayload, err error) {
	tokenSegments := strings.Split(identityToken, ".")
	if len(tokenSegments) < 2 {
		return nil, fmt.Errorf("Invalid existing refresh token length: %d", len(tokenSegments))
	}
	payloadSegmentEncoded := tokenSegments[1]
	var payloadBytes []byte
	if payloadBytes, err = jwt.DecodeSegment(payloadSegmentEncoded); err != nil {
		return nil, fmt.Errorf("Error decoding payload segment from refresh token, error: %s", err)
	}
	var payload acrTokenPayload
	if err = json.Unmarshal(payloadBytes, &payload); err != nil {
		return nil, fmt.Errorf("Error unmarshalling acr payload, error: %s", err)
	}
	return &payload, nil
}

func performTokenExchange(
	serverAddress string,
	directive *authDirective,
	tenant string,
	refreshTokenEncoded string) (string, error) {
	var err error
	data := url.Values{
		"service":       []string{directive.service},
		"grant_type":    []string{"refresh_token"},
		"refresh_token": []string{refreshTokenEncoded},
		"tenant":        []string{tenant},
	}

	var realmURL *url.URL
	if realmURL, err = url.Parse(directive.realm); err != nil {
		return "", fmt.Errorf("Www-Authenticate: invalid realm %s", directive.realm)
	}
	authEndpoint := fmt.Sprintf("%s://%s/oauth2/exchange", realmURL.Scheme, realmURL.Host)

	client := &http.Client{}
	datac := data.Encode()
	r, _ := http.NewRequest("POST", authEndpoint, bytes.NewBufferString(datac))
	r.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	r.Header.Add("Content-Length", strconv.Itoa(len(datac)))

	var exchange *http.Response
	if exchange, err = client.Do(r); err != nil {
		return "", fmt.Errorf("Www-Authenticate: failed to reach auth url %s", authEndpoint)
	}

	defer exchange.Body.Close()
	if exchange.StatusCode != 200 {
		return "", fmt.Errorf("Www-Authenticate: auth url %s responded with status code %d", authEndpoint, exchange.StatusCode)
	}

	var content []byte
	if content, err = ioutil.ReadAll(exchange.Body); err != nil {
		return "", fmt.Errorf("Www-Authenticate: error reading response from %s", authEndpoint)
	}

	var authResp acrAuthResponse
	if err = json.Unmarshal(content, &authResp); err != nil {
		return "", fmt.Errorf("Www-Authenticate: unable to read response %s", content)
	}

	return authResp.RefreshToken, nil
}

// Try and parse a string of assignments in the form of:
// key1 = value1, key2 = "value 2", key3 = ""
// Note: this method and handle quotes but does not handle escaping of quotes
func parseAssignments(statements string) (*map[string]string, error) {
	var cursor int
	result := make(map[string]string)
	var errorMsg = fmt.Errorf("malformed header value: %s", statements)
	for {
		// parse key
		equalIndex := nextOccurrence(statements, cursor, "=")
		if equalIndex == -1 {
			return nil, errorMsg
		}
		key := strings.TrimSpace(statements[cursor:equalIndex])

		// parse value
		cursor = nextNoneSpace(statements, equalIndex+1)
		if cursor == -1 {
			return nil, errorMsg
		}
		// case: value is quoted
		if statements[cursor] == '"' {
			cursor = cursor + 1
			// like I said, not handling escapes, but this will skip any comma that's
			// within the quotes which is somewhat more likely
			closeQuoteIndex := nextOccurrence(statements, cursor, "\"")
			if closeQuoteIndex == -1 {
				return nil, errorMsg
			}
			value := statements[cursor:closeQuoteIndex]
			result[key] = value

			commaIndex := nextNoneSpace(statements, closeQuoteIndex+1)
			if commaIndex == -1 {
				// no more comma, done
				return &result, nil
			} else if statements[commaIndex] != ',' {
				// expect comma immidately after close quote
				return nil, errorMsg
			} else {
				cursor = commaIndex + 1
			}
		} else {
			commaIndex := nextOccurrence(statements, cursor, ",")
			endStatements := commaIndex == -1
			var untrimmed string
			if endStatements {
				untrimmed = statements[cursor:commaIndex]
			} else {
				untrimmed = statements[cursor:]
			}
			value := strings.TrimSpace(untrimmed)

			if len(value) == 0 {
				// disallow empty value without quote
				return nil, errorMsg
			}

			result[key] = value

			if endStatements {
				return &result, nil
			}
			cursor = commaIndex + 1
		}
	}
}

func nextOccurrence(str string, start int, sep string) int {
	if start >= len(str) {
		return -1
	}
	offset := strings.Index(str[start:], sep)
	if offset == -1 {
		return -1
	}
	return offset + start
}

func nextNoneSpace(str string, start int) int {
	if start >= len(str) {
		return -1
	}
	offset := strings.IndexFunc(str[start:], func(c rune) bool { return !unicode.IsSpace(c) })
	if offset == -1 {
		return -1
	}
	return offset + start
}
