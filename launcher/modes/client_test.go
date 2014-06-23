// Copyright 2014 The jenkins-client-launcher Authors. All rights reserved.
// Use of this source code is governed by a MIT license that can be found in the LICENSE file.

package modes

import (
	"testing"
	util "github.com/jkellerer/jenkins-client-launcher/launcher/util"
)

var jenkinsNodePageHTML = `
<li><p>Run from slave command line:</p>
<pre>java -jar <a href="/ci/jnlpJars/slave.jar">slave.jar</a> -jnlpUrl https://ci.cttl.trendmicro.de/ci/computer/it-s-w2k12-x64-en/slave-agent.jnlp -secret 6319b6e88be1a62708e903fa422540fa58c2228aeb72161bdaf3b</pre></li>
`

var expectedSecret = "6319b6e88be1a62708e903fa422540fa58c2228aeb72161bdaf3b"

func TestClientModeIsRegistered(t *testing.T) {
	if new(ClientMode).Name() != GetConfiguredMode(&util.Config{RunMode:"client"}).Name() {
		t.Error("ClientMode is not registered in the modes list.")
	}
}

func TestCanExtractSecretKey(t *testing.T) {
	mode := new(ClientMode)
	in, out := mode.extractSecret([]byte(jenkinsNodePageHTML)), expectedSecret
	if in != out {
		t.Errorf("mode.extractSecret([]byte(jenkinsNodePageHTML)) = %v, want %v", in, out)
	}
}

