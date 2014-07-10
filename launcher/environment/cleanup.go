// Copyright 2014 The jenkins-client-launcher Authors. All rights reserved.
// Use of this source code is governed by a MIT license that can be found in the LICENSE file.

package environment

import (
	"os"
	"github.com/jkellerer/jenkins-client-launcher/launcher/util"
	"path/filepath"
	"time"
	"strings"
	"fmt"
	"encoding/xml"
)

// Min time to persist entries inside the folders selected for cleanup.
var minCleanupTTL = time.Hour * 6

// The min interval when folders selected for cleanup are monitored.
var minMonitoringInterval = time.Hour * 2

type JenkinsNodeConfig struct {
	XMLName     xml.Name `xml:"slave"`
	Name        string   `xml:"name"`
	RemoteFS    string   `xml:"remoteFS"`
}

// Returns the current configuration of this Jenkins node from the Jenkins server.
func GetJenkinsNodeConfig(config *util.Config) (*JenkinsNodeConfig, error) {
	if response, err := config.CIGet(fmt.Sprintf("/computer/%s/config.xml", config.ClientName)); err == nil {
		defer response.Body.Close()
		if response.StatusCode == 200 {
			config := &JenkinsNodeConfig{}
			err = xml.NewDecoder(response.Body).Decode(config)
			return config, err
		} else {
			return nil, fmt.Errorf(response.Status)
		}
	} else {
		return nil, err
	}
}

// Defines an object which continuously watches and cleans the temporary directory of
// files that haven't been modified for maxTTLInTempDirectories (default 24 hours).
type LocationCleaner struct {
	util.AnyConfigAcceptor

	tickers       []*time.Ticker
	workspacePath string
}

func NewLocationCleaner() *LocationCleaner {
	p := new(LocationCleaner)
	p.tickers = []*time.Ticker{}
	return p
}

func (self *LocationCleaner) Name() string {
	return "Directory Cleaner"
}

func (self *LocationCleaner) Prepare(config *util.Config) {
	for _, ticker := range self.tickers {
		ticker.Stop()
	}

	self.tickers = self.tickers[0:0]
	self.workspacePath = self.getWorkspacePath(config)

	for _, setting := range config.Maintenance.CleanupSettingsList {
		if !setting.Enabled {
			continue
		}
		self.initializeLocation(setting)
	}
}

func (self *LocationCleaner) getWorkspacePath(config *util.Config) string {
	baseDir := ""
	if nodeConfig, err := GetJenkinsNodeConfig(config); err == nil && nodeConfig.RemoteFS != "" {
		baseDir = filepath.FromSlash(nodeConfig.RemoteFS)
	} else {
		baseDir, _ = os.Getwd()
	}
	return filepath.Join(baseDir, "workspace")
}

func (self *LocationCleaner) initializeLocation(setting util.CleanupSettings) {
	maxTTL := time.Hour * time.Duration(setting.TTLHours)
	if maxTTL < minCleanupTTL { maxTTL = minCleanupTTL }

	monitoringInterval := time.Hour * time.Duration(setting.IntervalHours)
	if monitoringInterval < minMonitoringInterval { monitoringInterval = minMonitoringInterval }

	ticker := time.NewTicker(monitoringInterval)
	self.tickers = append(self.tickers, ticker)

	findLocations := func() []string {
		loc := os.Expand(setting.Location, func(name string) string {
				if strings.EqualFold(name, "workspace") {
					// TODO: Get real workspace path here
					cwd, _ := os.Getwd()
					return filepath.Join(cwd, "workspace")
				} else {
					return os.Getenv(name)
				}
			})

		if matches, err := filepath.Glob(loc); err == nil {
			return matches
		} else if fi, err := os.Stat(loc); err == nil && fi.IsDir() {
			return []string{loc}
		} else {
			return []string{}
		}
	}

	waitForIdle := func() {
		if setting.OnlyWhenIDLE && len(findLocations()) > 0 {
			self.waitForIdle()
		}
	}

	go func() {
		// Run first
		waitForIdle()
		self.cleanupLocations(findLocations(), setting.Exclusions, setting.Mode, maxTTL)

		// Run in schedule
		for _ = range ticker.C {
			waitForIdle()
			self.cleanupLocations(findLocations(), setting.Exclusions, setting.Mode, maxTTL)
		}
	}()
}

func (self *LocationCleaner) waitForIdle() {
	for !util.NodeIsIdle.Get() {
		util.GOut("temp", "Waiting for node to become IDLE before cleaning configured locations.")
		time.Sleep(time.Minute * 5)
	}
}

func (self *LocationCleaner) cleanupLocations(dirsToKeepClean, exclusions []string, mode string, maxTTL time.Duration) {
	for _, rootDir := range dirsToKeepClean {
		util.GOut("temp", "Cleaning expired files in %v", rootDir)

		dirToEmptyMap := map[string]bool{}
		expiredTimeOffset := time.Now().Add(-maxTTL)

		if mode == "TTLPerLocation" {
			exclusionCount := self.cleanupFiles(rootDir, expiredTimeOffset, true, exclusions, dirToEmptyMap)
			if exclusionCount > 0 {
				return
			}
		}

		// Handling outdated temporary files
		_ = self.cleanupFiles(rootDir, expiredTimeOffset, false, exclusions, dirToEmptyMap)

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

func (self *LocationCleaner) cleanupFiles(rootDir string, expiredTimeOffset time.Time, dryRun bool,
	exclusions []string, dirToEmptyMap map[string]bool) (exclusionCount int64) {

	filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
			if err == nil {
				dirToEmptyMap[filepath.Dir(path)] = false

				if info.IsDir() {
					dirToEmptyMap[filepath.Clean(path)] = true
				} else {
					fileIsToRemove := true

					if fileIsToRemove && info.ModTime().After(expiredTimeOffset) {
						fileIsToRemove = false
					}

					if fileIsToRemove && len(exclusions) > 0 {
						for _, pattern := range exclusions {
							if matchesExclusionPattern, _ := filepath.Match(pattern, path); matchesExclusionPattern {
								fileIsToRemove = false
								break
							}
						}
					}

					if fileIsToRemove {
						if !dryRun {
							if err := os.Remove(path); err == nil {
								util.GOut("temp", "Removed expired: %v", path)
							}
						}
					} else {
						exclusionCount++
					}
				}
			}
			return err
		})
	return
}

// Registering the cleaner.
var _ = RegisterPreparer(NewLocationCleaner())
