// Copyright 2014 The jenkins-client-launcher Authors. All rights reserved.
// Use of this source code is governed by a MIT license that can be found in the LICENSE file.

package modes

import (
	"testing"
	"github.com/jkellerer/jenkins-client-launcher/launcher/util"
	"regexp"
	"fmt"
)

var jenkinsNodePageHTML = `
<li><p>Run from slave command line:</p>
<pre>java -jar <a href="/ci/jnlpJars/slave.jar">slave.jar</a> -jnlpUrl https://jenkins/ci/computer/it-s-w2k12-x64-en/slave-agent.jnlp -secret 6319b6e88be1a62708e903fa422540fa58c2228aeb72161bdaf3b</pre></li>
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

func TestCommandlineIsFilteredForPasswords(t *testing.T) {
	mode := new(ClientMode)
	in := mode.createFilteredCommands([]string{"-a", "aval", "xyz", "-Auth", "123", "-b", "bval"})
	out := []string{util.Java, "-a", "aval", "xyz", "-Auth", "***", "-b", "bval"}

	if fmt.Sprintf("%v", in) != fmt.Sprintf("%v", out) {
		t.Errorf("mode.createFilteredCommands([]string(..., '-Auth', '123', ...)) = %v, want %v", in, out)
	}
}

func TestCanCustomizeAgentConfigJNLP(t *testing.T) {
	mode := new(ClientMode)
	config := &util.Config{RunMode:"client"}
	util.JnlpArgs["-url"] = "http://my-jenkins-host/my-ci/"
	util.JnlpArgs["-tunnel"] = "127.0.0.1:12345"

	in, out := mode.applyCustomJnlpArgs(config, []byte(`<jnlp spec="1.0+" codebase="https://jenkins/ci/computer/jenkins-vb-test/">
  <information>
	<title>Slave Agent for jenkins-vb-test</title>
	<vendor>Jenkins project</vendor>
	<homepage href="https://jenkins-ci.org/"></homepage>
  </information>
  <security>
	<all-permissions></all-permissions>
  </security>
  <resources>
	<j2se version="1.5+"></j2se>
	<jar href="https://jenkins/ci/jnlpJars/remoting.jar"></jar>
	<property name="hudson.showWindowsServiceInstallLink" value="true"></property>
  </resources>
  <application-desc main-class="hudson.remoting.jnlp.Main">
	<argument>6319b6e88be1a62708e903fa422540fa58c2228aeb72161bdaf3b</argument>
	<argument>jenkins-vb-test</argument>
	<argument>-url</argument>
	<argument>https://ci.jenkins/ci/</argument>
	<argument>-url</argument>
	<argument>http://ci.jenkins/ci/</argument>
	<argument>-someother</argument>
	<argument>some-value</argument>
  </application-desc>
</jnlp>`)), []byte(`<jnlp spec="1.0+" codebase="https://jenkins/ci/computer/jenkins-vb-test/">
  <information>
	<title>Slave Agent for jenkins-vb-test</title>
	<vendor>Jenkins project</vendor>
	<homepage href="https://jenkins-ci.org/"></homepage>
  </information>
  <security>
	<all-permissions></all-permissions>
  </security>
  <resources>
	<j2se version="1.5+"></j2se>
	<jar href="https://jenkins/ci/jnlpJars/remoting.jar"></jar>
	<property name="hudson.showWindowsServiceInstallLink" value="true"></property>
  </resources>
  <application-desc main-class="hudson.remoting.jnlp.Main">
	<argument>6319b6e88be1a62708e903fa422540fa58c2228aeb72161bdaf3b</argument>
	<argument>jenkins-vb-test</argument>
	<argument>-someother</argument>
	<argument>some-value</argument>
	<argument>-url</argument>
	<argument>http://my-jenkins-host/my-ci/</argument>
	<argument>-tunnel</argument>
	<argument>127.0.0.1:12345</argument>
  </application-desc>
</jnlp>`)

	spaces, _ := regexp.Compile("\\s+")
	if string(spaces.ReplaceAll(in, []byte(""))) != string(spaces.ReplaceAll(out, []byte(""))) {
		t.Errorf("mode.processCustomizedAgentJNLP(config, ...) = %v, want %v", string(in), string(out))
	}
}
