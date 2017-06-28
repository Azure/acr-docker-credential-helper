package main

import (
	"os/exec"

	"fmt"
	"os"

	"path/filepath"

	"strings"

	"strconv"

	dockerCommand "github.com/docker/cli/cli/command"
	cliconfig "github.com/docker/cli/cli/config"
	"github.com/docker/cli/cli/config/configfile"
	dockerCredentials "github.com/docker/cli/cli/config/credentials"
	helperCredentials "github.com/docker/docker-credential-helpers/credentials"
	dockerTypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/pkg/term"
)

type storeWrapper struct {
	store *dockerCredentials.Store
}

const tokenUsername = "<token>"
const chunkPostfix = "-acr-credential-helper"
const maxChunksAllowed = 16

func (w *storeWrapper) Add(cred *helperCredentials.Credentials) error {
	store := *w.store
	chunks, err := toChunks(cred)
	if err != nil {
		return err
	}
	for _, chunk := range chunks {
		config := dockerTypes.AuthConfig{
			ServerAddress: chunk.ServerURL,
			Username:      chunk.Username,
		}
		if chunk.Username == tokenUsername {
			config.IdentityToken = chunk.Secret
		} else {
			config.Password = chunk.Secret
		}
		if err := store.Store(config); err != nil {
			return err
		}
	}
	return nil
}

// Note that this method is designed to swallow credentials not found error to workaround
// a docker cli bug where it could try to delete an entry form store multiple times when
// a non-default registry is used
func (w *storeWrapper) Delete(serverURL string) error {
	chunkCount, _, err := w.getAggregatedAuthConfigs(serverURL)
	if err != nil {
		return err
	}

	store := *w.store
	for i := chunkCount - 1; i >= 0; i-- {
		err := store.Erase(toChunkName(serverURL, i))
		if err != nil {
			return err
		}
	}

	return nil
}

func (w *storeWrapper) Get(serverURL string) (string, string, error) {
	user, cred, err := w.getFromStore(serverURL)
	if user == tokenUsername {
		// NOTE: currently docker calls Get from credstore even if the
		// user passes -u and -p. If we enable interactive login during
		// Get, this will result in docker prompt for interactive login
		// even if -u -p. Make sure that docker stop calling Get when -u
		// and -p before we can even think about enable interactive login
		user, cred, err = GetUsernamePassword(serverURL, cred)
	}
	return user, cred, err
}

func (w *storeWrapper) List() (map[string]string, error) {
	store := *w.store
	storeResults, err := store.GetAll()
	if err != nil {
		return map[string]string{}, err
	}
	results := make(map[string]string)
	for k, v := range storeResults {
		if !isChunkName(k) {
			results[k] = v.Username
		}
	}
	return results, nil
}

func (w *storeWrapper) getFromStore(serverURL string) (string, string, error) {
	numChunks, cred, err := w.getAggregatedAuthConfigs(serverURL)
	if err != nil {
		return "", "", err
	}

	if numChunks == 0 {
		return "", "", helperCredentials.NewErrCredentialsNotFound()
	}

	var secret string
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

var emptyConfig = dockerTypes.AuthConfig{}

// Note that filestore does not throw error when not found
// This should be considered a bug, we are working around it
func (w *storeWrapper) safeGet(key string) (dockerTypes.AuthConfig, error) {
	store := *w.store
	cred, err := store.Get(key)
	if err == nil && cred == emptyConfig {
		err = helperCredentials.NewErrCredentialsNotFound()
	}
	return cred, err
}

// Note: this method does not give an error when nothing is found
func (w *storeWrapper) getAggregatedAuthConfigs(serverURL string) (int, dockerTypes.AuthConfig, error) {
	var aggregate dockerTypes.AuthConfig
	chunkCount := 0
	for {
		chunk, getChunkErr := w.safeGet(toChunkName(serverURL, chunkCount))
		if getChunkErr == nil {
			if chunkCount == 0 {
				aggregate = chunk
			} else {
				if aggregate.Username != chunk.Username {
					return 0, dockerTypes.AuthConfig{}, fmt.Errorf("Chunk mismatch detected for %s", serverURL)
				}
				aggregate.IdentityToken = aggregate.IdentityToken + chunk.IdentityToken
				aggregate.Password = aggregate.Password + chunk.Password
			}
			chunkCount++
			if chunkCount > maxChunksAllowed {
				return 0, dockerTypes.AuthConfig{}, fmt.Errorf("Too many chunk detected for %s", serverURL)
			}
		} else if helperCredentials.IsErrCredentialsNotFound(getChunkErr) {
			// end of chunks
			break
		} else {
			return 0, dockerTypes.AuthConfig{}, fmt.Errorf("Error gathering credential chunks: %s", getChunkErr)
		}
	}
	return chunkCount, aggregate, nil
}

func getCredentialsStore() (*dockerCredentials.Store, error) {
	_, _, stderr := term.StdStreams()
	// NOTE: This tool would always use the default config file location currently
	config := dockerCommand.LoadDefaultConfigFile(stderr)
	if config == nil {
		return nil, fmt.Errorf("Problem loading docker config file at default location")
	}
	// NOTE: This tool would always use wincred for windows
	// secretservice for linux
	// osxkeychain for osx
	// if they are found. Otherwise it would revert to using native
	if configHelperFound() {
		store := dockerCredentials.NewNativeStore(config, helperSuffix)
		return &store, nil
	}

	return newSecondaryFileStore(config)
}

func newSecondaryFileStore(config *configfile.ConfigFile) (store *dockerCredentials.Store, err error) {
	configdir := filepath.Dir(config.Filename)
	secondarydir := filepath.Join(configdir, "acr")
	var fileInfo os.FileInfo
	var secondaryConfig *configfile.ConfigFile
	if fileInfo, err = os.Stat(secondarydir); err != nil {
		if os.IsNotExist(err) {
			err = os.Mkdir(secondarydir, 0777)
			if err != nil {
				return nil, fmt.Errorf("Failed to create secondary file store dir %s", secondarydir)
			}
			secondaryConfig = getEmptyConfig(secondarydir)
		} else {
			return nil, fmt.Errorf("Failed to stat dir %s", secondarydir)
		}
	} else {
		if !fileInfo.IsDir() {
			return nil, fmt.Errorf("Failed to create secondary file store dir %s, a file already exist in its location", secondarydir)
		}
		secondaryConfig, err = cliconfig.Load(secondarydir)
		if err != nil {
			_, statError := os.Stat(filepath.Join(secondarydir, "config.json"))
			if os.IsNotExist(statError) {
				secondaryConfig = getEmptyConfig(secondarydir)
			} else {
				return nil, fmt.Errorf("Failed to load existing config from %s, error: %s", secondarydir, err)
			}
		}
	}

	// THIS IS REALLY INEFFICIENT...
	var oldCreds map[string]dockerTypes.AuthConfig
	oldStore := dockerCredentials.NewFileStore(config)
	oldCreds, err = oldStore.GetAll()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error retrieving old credentials, skipping credentials sync. Error: %s", err)
	} else {
		for server, oldCred := range oldCreds {
			if _, found := secondaryConfig.AuthConfigs[server]; !found {
				secondaryConfig.AuthConfigs[server] = oldCred
				// note that the oldCreds would be wiped by docker as a side effect
			}
		}
	}
	secondaryFileStore := dockerCredentials.NewFileStore(secondaryConfig)
	return &secondaryFileStore, nil
}

func getEmptyConfig(dir string) *configfile.ConfigFile {
	return &configfile.ConfigFile{
		Filename:    filepath.Join(dir, "config.json"),
		AuthConfigs: map[string]dockerTypes.AuthConfig{},
	}
}

func configHelperFound() bool {
	helperName := "docker-credential-" + helperSuffix
	lookupCmd := exec.Command(exeFinder, helperName)
	err := lookupCmd.Run()
	return err == nil
}

func toChunks(cred *helperCredentials.Credentials) ([]helperCredentials.Credentials, error) {
	numChunks := getNumChunks(len(cred.Secret), helperMaxBlobLength)
	if numChunks > maxChunksAllowed {
		return []helperCredentials.Credentials{}, fmt.Errorf("Input credential is too big")
	}
	result := make([]helperCredentials.Credentials, numChunks)
	for i := 0; i < numChunks; i++ {
		result[i].Username = cred.Username
		lower := i * helperMaxBlobLength
		var upper int
		if i == numChunks-1 {
			upper = len(cred.Secret)
		} else {
			upper = lower + helperMaxBlobLength
		}
		result[i].Secret = cred.Secret[lower:upper]
		result[i].ServerURL = toChunkName(cred.ServerURL, i)
	}
	return result[:], nil
}

func toChunkName(server string, index int) string {
	if index == 0 {
		return server
	}
	return server + strconv.Itoa(index-1) + chunkPostfix
}

func getNumChunks(strLen int, chunkLen int) int {
	division := strLen / chunkLen
	if division == 0 || strLen%chunkLen != 0 {
		return division + 1
	}
	return division
}

func isChunkName(key string) bool {
	return strings.HasSuffix(key, chunkPostfix)
}

func main() {
	store, err := getCredentialsStore()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating credential store helper: %s\n", err)
		os.Exit(1)
	}
	helperCredentials.Serve(&storeWrapper{
		store: store,
	})
}
