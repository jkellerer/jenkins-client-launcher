// Copyright 2014 The jenkins-client-launcher Authors. All rights reserved.
// Use of this source code is governed by a MIT license that can be found in the LICENSE file.

package launcher

import (
	"flag"
	"os"
	"path/filepath"
	modes "github.com/jenkins-client-launcher/launcher/modes"
	util "github.com/jenkins-client-launcher/launcher/util"
	environment "github.com/jenkins-client-launcher/launcher/environment"
	"fmt"
	"regexp"
	"net/http"
	"io"
	"time"
)

const (
	AppName    = "Jenkins Client Launcher"
	AppVersion = "0.1"
	ConfigName = "launcher.config"
)

// Is the applications main run loop.
func Run() {
	fmt.Println("\n" + AppName + " " + AppVersion + "\n---------------------------")

	modeNames := make([]string, len(modes.AllModes))
	for i, m := range modes.AllModes { modeNames[i] = m.Name() }


	help := flag.Bool("help", false, "Show this help.")
	autoStart := flag.Bool("autostart", false, "Automatically start this app when the OS runs (implies '-persist=true').")
	runMode := flag.String("runMode", "client", "Specifies the mode in which the launcher operates if not already set " +
				"inside '"+ConfigName+"'. Supporte modes are "+fmt.Sprintf("%v", modeNames)+".")

	url := flag.String("url", "", "Specifies the URL to Jenkins server if not already set inside '"+ConfigName+"'.")
	name := flag.String("name", "", "Specifies the name of this node in Jenkins (defaults to [hostname] if not specified).")
	secretKey := flag.String("secret", "", "Specifies the secret key to use in client mode when starting the Jenkins client.")
	acceptAnyCert := flag.Bool("anyCert", false, "Disabled cert verification for TLS connections with Jenkins (this is not secure at all).")

	dir := flag.String("directory", "", "Changes the current working directory before performing any other operations.")
	saveChanges := flag.Bool("persist", false, "Stores any CLI config overrides inside '"+ConfigName+"'.")
	defaultConfig := flag.String("defaultConfig", "", "Loads the initial config from the specified path or URL. " +
				"Does nothing when '"+ConfigName+"' exists already.")
	overwrite := flag.Bool("overwrite", false, "Overwrites '"+ConfigName+"' with the content from initial config " +
				"(requires '-defaultConfig=...', implies '-persist=true').")

	flag.CommandLine.Init(AppName, flag.ContinueOnError)
	flag.CommandLine.SetOutput(os.Stdout)
	flag.Parse()

	wd, _ := os.Getwd()
	if len(*dir) > 0 && *dir != wd {
		util.Out("Changing working directory to %v", *dir)
		if err := os.Chdir(*dir); err != nil {
			wd, _ = os.Getwd()
			panic(fmt.Sprintf("Failed changing working directory, stopping here to prevent any damage. Cause: %v ; CWD is %v", err, wd))
		}
	}

	config := loadConfig(*defaultConfig, *overwrite)

	if len(*runMode) > 0 { config.RunMode = *runMode }
	if len(*url) > 0 { config.CIHostURI = *url }
	if len(*secretKey) > 0 { config.SecretKey = *secretKey }
	if len(*name) > 0 { config.ClientName = *name }
	if *acceptAnyCert { config.CIAcceptAnyCert = true }

	saveConfigIfRequired := func() {
		if config.NeedsSave || *saveChanges || *autoStart || *overwrite {
			config.Save(ConfigName)
		}
	}

	if *help {
		saveConfigIfRequired()
		fmt.Printf(`
This application attempts to provide a stable runtime environment for a Jenkins client.
Regardless of the run mode, clients are started in 'user-mode' inheriting user & environment of the caller.
Functionality is controlled via CLI options and '`+ConfigName+`' which is created in the current working directory when missing.

Usage %s [options]

Options:

`, filepath.Base(os.Args[0]))
		flag.PrintDefaults()
		return
	}

	environment.RunPreparers(config)

	abort := !modes.GetConfiguredMode(config).IsConfigAcceptable(config)
	saveConfigIfRequired()

	if abort {
		return
	}

	restartCount := int64(0)
	sleepTimePerRestart := time.Second * 30
	runTimeAfterResettingRestartCount := time.Hour * 2

	timeOfLastStart := time.Now()

	for modes.RunConfiguredMode(config) {
		fmt.Print("\n:::::::::::::::::::::::::::::::::\n")
		fmt.Print("::  Restarting Jenkins Client  ::")
		fmt.Print("\n:::::::::::::::::::::::::::::::::\n\n")

		if timeOfLastStart.Before(time.Now().Add(-runTimeAfterResettingRestartCount)) {
			restartCount = 0
		}

		if sleepTime := time.Duration(restartCount * int64(sleepTimePerRestart)); sleepTime > 0 {
			fmt.Printf("Sleeping %v seconds before restarting the client.\n\n", sleepTime.Seconds())
			time.Sleep(sleepTime)
		}

		restartCount++
		timeOfLastStart = time.Now()
	}
}

// Loads the config file from disk or from the specified defaultConfig location
// when either the local file is missing or overwriteWithInitial is true.
func loadConfig(defaultConfig string, overwriteWithInitial bool) *util.Config {
	config := util.NewConfig(ConfigName)

	if len(defaultConfig) > 0 && (config.NeedsSave || overwriteWithInitial) {
		var err error
		var configSource io.ReadCloser
		defer func() {
			if (configSource != nil) {
				configSource.Close()
			}
		}()

		if isHttpUrl, _ := regexp.MatchString("^(?i)http(s|)://.+", defaultConfig); isHttpUrl {
			util.Out("Downloading: %v", defaultConfig)
			var response *http.Response
			if response, err = http.Get(defaultConfig); err == nil {
				if response.StatusCode == 200 {
					configSource = response.Body
				} else {
					err = fmt.Errorf(response.Status)
				}
			}
		} else {
			util.Out("Copying: %v", defaultConfig)
			configSource, err = os.Open(defaultConfig)
		}

		if err != nil {
			panic(fmt.Sprintf("Failed loading %v;\nCause: %v; => exiting.", defaultConfig, err))
		}

		if configFile, err := os.Create(ConfigName); err == nil {
			defer configFile.Close()

			if _, err = io.Copy(configFile, configSource); err == nil {
				config = util.NewConfig(ConfigName)
			}
		} else {
			panic(fmt.Sprintf("Failed creating initial %v from %v;\ncause: %v; => exiting.", ConfigName, defaultConfig, err))
		}
	}

	return config;
}
