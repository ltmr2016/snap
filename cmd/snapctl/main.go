/*
http://www.apache.org/licenses/LICENSE-2.0.txt


Copyright 2015 Intel Corporation

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"golang.org/x/crypto/ssh/terminal"

	"github.com/codegangsta/cli"
	"github.com/intelsdi-x/snap/mgmt/rest/client"
)

var (
	gitversion string
	pClient    *client.Client
	timeFormat = time.RFC1123
	err        error
)

func main() {
	app := cli.NewApp()
	app.Name = "snapctl"
	app.Version = gitversion
	app.Usage = "A powerful telemetry framework"
	app.Flags = []cli.Flag{flURL, flSecure, flAPIVer, flPassword, flConfig}
	app.Commands = append(commands, tribeCommands...)
	sort.Sort(ByCommand(app.Commands))
	app.Before = beforeAction
	app.Run(os.Args)
}

// Run before every command
func beforeAction(ctx *cli.Context) error {
	username, password := checkForAuth(ctx)
	pClient, err = client.New(ctx.String("url"), ctx.String("api-version"), ctx.Bool("insecure"))
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	pClient.Password = password
	pClient.Username = username
	if err = checkTribeCommand(ctx); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	return nil
}

// Checks if a tribe command was issued when tribe mode was not
// enabled on the specified snapd instance.
func checkTribeCommand(ctx *cli.Context) error {
	tribe := false
	for _, a := range os.Args {
		for _, command := range tribeCommands {
			if strings.Contains(a, command.Name) {
				tribe = true
				break
			}
		}
		if tribe {
			break
		}
	}
	if !tribe {
		return nil
	}
	resp := pClient.ListAgreements()
	if resp.Err != nil {
		if resp.Err.Error() == "Invalid credentials" {
			return resp.Err
		}
		return fmt.Errorf("Tribe mode must be enabled in snapd to use tribe command")
	}
	return nil
}

// Checks for authentication flags and returns a username/password
// from the specified settings
func checkForAuth(ctx *cli.Context) (username, password string) {
	if ctx.IsSet("password") {
		username = "snap" // for now since username is unused but needs to exist for basicAuth
		// Prompt for password
		fmt.Print("Password:")
		pass, err := terminal.ReadPassword(0)
		if err != nil {
			password = ""
		} else {
			password = string(pass)
		}
		// Go to next line after password prompt
		fmt.Println()
		return
	}
	//Get config file path in the order:
	if ctx.IsSet("config") {
		cfg := &config{}
		if err := cfg.loadConfig(ctx.String("config")); err != nil {
			fmt.Println(err)
		}
		if cfg.RestAPI.Password != nil {
			password = *cfg.RestAPI.Password
		} else {
			fmt.Println("Error config password field 'rest-auth-pwd' is empty")
		}
	}
	return
}
