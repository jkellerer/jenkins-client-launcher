// Copyright 2014 The jenkins-client-launcher Authors. All rights reserved.
// Use of this source code is governed by a MIT license that can be found in the LICENSE file.

package environment

import (
	util "github.com/jkellerer/jenkins-client-launcher/launcher/util"
	"fmt"
)

// Implements Java installation for Windows
func (self *JavaDownloader) InstallJava(config *util.Config) error {
	util.GOut("DOWNLOAD", "Getting %v", config.CIHostURI)
	util.GOut("INSTALL", "Installing %v", config.CIHostURI)

	// TODO: Implement like done here:
	// TODO: https://github.com/jenkinsci/jenkins/blob/main/core/src/main/java/hudson/tools/JDKInstaller.java

	return fmt.Errorf("Installing Java is not implemented yet. Install it manually.")
}




