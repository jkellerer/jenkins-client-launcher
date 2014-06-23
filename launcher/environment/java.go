// Copyright 2014 The jenkins-client-launcher Authors. All rights reserved.
// Use of this source code is governed by a MIT license that can be found in the LICENSE file.

package environment

import (
	"launcher/util"
	"fmt"
	"os/exec"
	"regexp"
	version "github.com/mcuadros/go-version"
)

const (
	MinJavaVersion = "1.6.0"
)

// Defines an interface for implementations that can install java.
type JavaInstaller interface {
	InstallJava(config *util.Config) error
}

// Implements a downloader that ensures that a JDK is installed before either server of client mode is executed.
type JavaDownloader struct {
	util.AnyConfigAcceptor
}

func (self *JavaDownloader) Name() string {
	return "Java Downloader"
}

func (self *JavaDownloader) Prepare(config *util.Config) {
	if !self.javaIsInstalled() {
		var i interface{}; i = self

		if installer, implemented := i.(JavaInstaller); implemented {
			if err := installer.InstallJava(config); err != nil || !self.javaIsInstalled() {
				panic(fmt.Sprintf("Java installation failed, cannot continue. Cause: %v", err))
			}
		} else {
			panic("Java installation support not implemented. Install Java manually to overcome this message.")
		}
	}
}

// Checks if java is installed and if the java version is greater or equals the required version.
func (self *JavaDownloader) javaIsInstalled() bool {
	if java, err := exec.LookPath("java"); err == nil {
		// Set absolute path to java
		util.Java = java
		// Check the version
		if output, err := exec.Command(java, "-version").CombinedOutput(); err == nil {
			if pattern, err := regexp.Compile(`(?i)java version "([^"]+)"`); err == nil {
				if matches := pattern.FindSubmatch(output); matches != nil && len(matches) == 2 {
					javaVersion := string(matches[1])
					if version.Compare(javaVersion, MinJavaVersion, ">=") {
						util.GOut("java", "Found java version %v, no need to install a newer version.", javaVersion)
						return true
					}
					util.GOut("java", "Found java version %v. A newer version is required to run the Jenkins client.", javaVersion)
				}
			}
		}
	}

	return false
}

// Registering the downloader.
var _ = RegisterPreparer(new(JavaDownloader))
