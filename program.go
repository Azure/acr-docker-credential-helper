package main

import (
	dockerCommand "github.com/docker/cli/cli/command"
	dockerCredentials "github.com/docker/cli/cli/config/credentials"
	helperCredentials "github.com/docker/docker-credential-helpers/credentials"
	dockerTypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/pkg/term"
)

type storeWrapper struct {
	store *dockerCredentials.Store
}

const tokenUsername = "<token>"

func (w *storeWrapper) Add(cred *helperCredentials.Credentials) error {
	store := *w.store
	config := dockerTypes.AuthConfig{
		ServerAddress: cred.ServerURL,
		Username:      cred.Username,
		Password:      cred.Secret,
	}
	if cred.Username == tokenUsername {
		config.IdentityToken = cred.Secret
	}
	return store.Store(config)
}

func (w *storeWrapper) Delete(serverURL string) error {
	store := *w.store
	return store.Erase(serverURL)
}

func (w *storeWrapper) Get(serverURL string) (string, string, error) {
	user, cred, err := w.getFromStore(serverURL)
	if len(user) == 0 {
		return GetUsernamePassword(serverURL, "")
	} else if user == tokenUsername {
		return GetUsernamePassword(serverURL, cred)
	}
	return user, cred, err
}

func (w *storeWrapper) getFromStore(serverURL string) (string, string, error) {
	store := *w.store
	cred, err := store.Get(serverURL)
	if err != nil {
		return "", "", err
	}

	var secret string
	// Note that in github.com/docker/cli/cli/config/credentials/native_store.go,
	// cred.Username is ignored if the response form the cred helper has username <token>
	// we need to put it back here
	if len(cred.Username) == 0 {
		cred.Username = tokenUsername
	}

	if cred.Username == tokenUsername {
		secret = cred.IdentityToken
	} else {
		secret = cred.Password
	}

	return cred.Username, secret, nil
}

func (w *storeWrapper) List() (map[string]string, error) {
	store := *w.store
	storeResults, err := store.GetAll()
	if err != nil {
		return map[string]string{}, err
	}
	results := make(map[string]string)
	for k, v := range storeResults {
		results[k] = v.Username
	}
	return results, nil
}

func getCredentialsStore() *dockerCredentials.Store {
	_, _, stderr := term.StdStreams()
	config := dockerCommand.LoadDefaultConfigFile(stderr)
	if helperSuffix != "" {
		store := dockerCredentials.NewNativeStore(config, helperSuffix)
		return &store
	}
	store := dockerCredentials.NewFileStore(config)
	return &store
}

func main() {
	helperCredentials.Serve(&storeWrapper{
		store: getCredentialsStore(),
	})
}
