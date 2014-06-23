// Copyright 2014 The jenkins-client-launcher Authors. All rights reserved.
// Use of this source code is governed by a MIT license that can be found in the LICENSE file.

package modes

import (
	"testing"
	"launcher/util"
)

func TestServerModeIsRegistered(t *testing.T) {
	if new(ServerMode).Name() != GetConfiguredMode(&util.Config{RunMode:"ssh-server"}).Name() {
		t.Error("ClientMode is not registered in the modes list.")
	}
}

