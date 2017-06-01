package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
)

var force bool

func main() {
	var configFile, helper, server string
	cmd := &cobra.Command{
		Use:   "Docker Login Config Editor",
		Short: "Configure docker to use different helper for login.",
		Long:  "Configure docker to use different helper for login.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(configFile) == 0 {
				configFile = path.Join(userHomeDir(), ".docker", "config.json")
			}
			if len(helper) == 0 {
				return fmt.Errorf("Please specify a helper name")
			}

			var err error
			var configObj *map[string]interface{}
			if configObj, err = loadConfigObject(configFile); err != nil {
				return err
			}

			err = editConfigObject(configObj, server, helper)
			if err != nil {
				return err
			}

			var bytes []byte
			if bytes, err = json.MarshalIndent(configObj, "", "\t"); err != nil {
				return fmt.Errorf("Error trying to marshal config object, err: %s", err)
			}

			var bakFileName = configFile + ".bak"
			if _, err = os.Stat(bakFileName); err == nil || !os.IsNotExist(err) {
				if err = promptForAbort("Please note that bak file will be overwritten, continue?"); err != nil {
					return err
				}
			}

			// NOTE: if any process created a bak file at this point by any chance, it would be overwritten
			if err = os.Rename(configFile, bakFileName); err != nil && !os.IsNotExist(err) {
				if err = promptForAbort("Unable to back up config file, continue?"); err != nil {
					return err
				}
			}

			fmt.Printf("Docker config %s will be edited\n", configFile)
			if err = ioutil.WriteFile(configFile, bytes, 0644); err != nil {
				return fmt.Errorf("Error trying to write file to location %s, err: %s", configFile, err)
			}

			return nil
		},
	}

	fmt.Println("Runing ACR docker config editor...")

	flags := cmd.Flags()
	flags.StringVar(&configFile, "config-file", "", "Location of the config file.")
	flags.StringVar(&helper, "helper", "", "Name of the login helper to be used.")
	flags.StringVar(&server, "server", "", "Docker registry url to use this helper.")
	flags.BoolVar(&force, "force", false, "Silently continue on warnings")

	cmd.MarkFlagRequired("helper")

	if err := cmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running subcommand: %s\n", err)
		os.Exit(-1)
	}
}

func loadConfigObject(configFile string) (configObj *map[string]interface{}, err error) {
	if _, err = os.Stat(configFile); err != nil {
		if os.IsNotExist(err) {
			configObj = &map[string]interface{}{}
		} else {
			return nil, fmt.Errorf("Error trying to access config file: %s, err: %s", configFile, err)
		}
	} else {
		var bytes []byte
		if bytes, err = ioutil.ReadFile(configFile); err != nil {
			return nil, fmt.Errorf("Error trying to read config file %s, err: %s", configFile, err)
		}
		if err = json.Unmarshal(bytes, &configObj); err != nil {
			return nil, fmt.Errorf("Error trying to unmarshal config file %s, err: %s", configFile, err)
		}
	}
	return configObj, err
}

func editConfigObject(configObj *map[string]interface{}, server string, helper string) error {
	if len(server) == 0 {
		// edit credsStore element
		(*configObj)["credsStore"] = helper
	} else {
		// edit credHelpers element
		helperMapObj, exists := (*configObj)["credHelpers"]
		helperMap := make(map[string]string)
		if exists {
			oldMap, ok := helperMapObj.(map[string]interface{})
			if !ok {
				return fmt.Errorf("Error parsing old credHelpers")
			}
			for k, v := range oldMap {
				value, ok := v.(string)
				if !ok {
					return fmt.Errorf("Error parsing old credHelpers value")
				}
				helperMap[k] = value
			}
		}
		(*configObj)["credHelpers"] = helperMap
		helperMap[server] = helper
	}

	return nil
}

func promptForAbort(msg string) error {
	if force {
		return nil
	}

	fmt.Printf("%s [Y/y]", msg)
	var ans string
	var err error
	if _, err = fmt.Scanf("%s", &ans); err != nil {
		return fmt.Errorf("Unable to get user input")
	}
	if !strings.EqualFold(ans, "y") {
		return fmt.Errorf("User aborted")
	}
	return nil
}

func userHomeDir() string {
	if runtime.GOOS == "windows" {
		home := os.Getenv("HOMEDRIVE") + os.Getenv("HOMEPATH")
		if home == "" {
			home = os.Getenv("USERPROFILE")
		}
		return home
	}
	return os.Getenv("HOME")
}
