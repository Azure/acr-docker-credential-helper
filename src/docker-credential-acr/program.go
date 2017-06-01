package main

import (
	"os/exec"

	"fmt"
	"os"

	"path/filepath"

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

func (w *storeWrapper) Add(cred *helperCredentials.Credentials) error {
	store := *w.store
	config := dockerTypes.AuthConfig{
		ServerAddress: cred.ServerURL,
		Username:      cred.Username,
	}
	if cred.Username == tokenUsername {
		config.IdentityToken = cred.Secret
	} else {
		config.Password = cred.Secret
	}
	return store.Store(config)
}

func (w *storeWrapper) Delete(serverURL string) error {
	store := *w.store
	return store.Erase(serverURL)
}

func (w *storeWrapper) Get(serverURL string) (string, string, error) {
	user, cred, err := w.getFromStore(serverURL)
	if user == tokenUsername {
		// no password/token is saved
		if cred == "" {
			// pass through
			return "", "", nil
			// NOTE: currently docker calls Get from credstore even if the
			// user passes -u and -p. If we enable interactive login during
			// Get, this will result in docker prompt for interactive login
			// even if -u -p. Make sure that docker stop calling Get when -u
			// and -p before we can even think about enable interactive login
		}
		user, cred, err = GetUsernamePassword(serverURL, cred)
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
