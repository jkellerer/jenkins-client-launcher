// Copyright 2014 The jenkins-client-launcher Authors. All rights reserved.
// Use of this source code is governed by a MIT license that can be found in the LICENSE file.

package environment

import (
	"testing"
	util "github.com/jkellerer/jenkins-client-launcher/launcher/util"
)

var handler = NewAutostartHandler()

func TestCanRegisterAutoStart(t *testing.T) {
	unregisterAutostart(); defer unregisterAutostart()

	if handler.isRegistered() { t.Errorf("Initial state is not 'unregistered'.") }

	config := util.NewDefaultConfig()

	config.Autostart = true
	handler.Prepare(config)

	if !handler.isRegistered() { t.Errorf("Handler did not register for autostart when requested.") }

	config.Autostart = false
	handler.Prepare(config)

	if handler.isRegistered() { t.Errorf("Handler did not unregister from autostart when requested.") }
}

func unregisterAutostart() {
	if handler.isRegistered() {
		handler.unregister()
	}
}
