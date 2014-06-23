// Copyright 2014 The jenkins-client-launcher Authors. All rights reserved.
// Use of this source code is governed by a MIT license that can be found in the LICENSE file.

package environment

import (
	"os"
	util "github.com/jenkins-client-launcher/launcher/util"
	"path/filepath"
	"time"
)

// Max time to persist entries inside the temporary folders.
var maxTTLInTempDirectories = time.Hour * 24

// The interval when temporary directories are monitored.
var tempDirectoriesMonitoringInterval = time.Hour * 4

// Defines an object which continuously watches and cleans the temporary directory of
// files that haven't been modified for maxTTLInTempDirectories (default 24 hours).
type TempLocationCleaner struct {
	util.AnyConfigAcceptor

	ticker *time.Ticker
}

func (self *TempLocationCleaner) Name() string {
	return "Temp Directory Cleaner"
}

func (self *TempLocationCleaner) Prepare(config *util.Config) {
	if self.ticker != nil {
		self.ticker.Stop()
	}

	if !config.CleanTempLocation {
		return
	}

	if config.CleanTempLocationTTLHours > 0 {
		maxTTLInTempDirectories = time.Hour*time.Duration(config.CleanTempLocationTTLHours)
	}

	if config.CleanTempLocationIntervalHours > 0 {
		tempDirectoriesMonitoringInterval = time.Hour*time.Duration(config.CleanTempLocationIntervalHours)
	}

	self.ticker = time.NewTicker(tempDirectoriesMonitoringInterval) // Clean temp locations every 4 hours.

	go func() {
		dirsToKeepClean := []string{os.TempDir()}
		// Run first
		self.cleanTempLocations(config, dirsToKeepClean)
		// Run in schedule
		for time := range self.ticker.C {
			util.GOut("temp", "Looking after temp locations at %v", time)
			self.cleanTempLocations(config, dirsToKeepClean)
		}
	}()
}

func (self *TempLocationCleaner) cleanTempLocations(config *util.Config, dirsToKeepClean []string) {
	for _, rootDir := range dirsToKeepClean {
		util.GOut("temp", "Cleaning expired files in %v", rootDir)

		dirToEmptyMap := map[string]bool{}
		expiredTime := time.Now().Add(-maxTTLInTempDirectories)

		// Handling outdated temporary files
		filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
				if err == nil {
					dirToEmptyMap[filepath.Dir(path)] = false

					if info.IsDir() {
						dirToEmptyMap[filepath.Clean(path)] = true
					} else {
						// TODO: Implement config.CleanTempLocationExclusions
						if info.ModTime().Before(expiredTime) {
							if err := os.Remove(path); err == nil {
								util.GOut("temp", "Removed expired: %v", path)
							}
						}
					}
				}
				return err
			})

		// Handling all directories that are known to be empty
		for dirPath, emptyDir := range dirToEmptyMap {
			if emptyDir {
				if err := os.Remove(dirPath); err == nil {
					util.GOut("temp", "Removed empty directory: %v", dirPath)
				}
			}
		}
	}
}

// Registering the cleaner.
var _ = RegisterPreparer(new(TempLocationCleaner))
