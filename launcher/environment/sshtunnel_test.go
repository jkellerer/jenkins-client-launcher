// Copyright 2014 The jenkins-client-launcher Authors. All rights reserved.
// Use of this source code is governed by a MIT license that can be found in the LICENSE file.

package environment

import (
	"testing"
	"code.google.com/p/go.crypto/ssh"
)

var sshTunnel = NewSSHTunnelEstablisher(false)

func TestCanCreateHostFingerPrint(t *testing.T) {
	var key = []byte("ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQDEbKq5U57fhzQ3SBbs3NVmgY2ouYZfPhc6cXBNEFpRT3T100fnbkYw+EHi76nwsp+uGxk08kh4GG881DrgotptrJj2dJxXpWp/SFdVu5S9fFU6l6dCTC9IBYYCCV8PvXbBZ3oDZyyyJT7/vXSaUdbk3x9MeNlYrgItm2KY6MdHYEg8R994Sspn1sE4Ydey5DfG/WNWVrzFCI0sWI3yj4zuCcUXFz9sEG8fIYikD9rNuohiMenWjkj6oLTwZGVW2q4wRL0051XBkmfnPD/H6gqOML9MbZQ8D6/+az0yF9oD61SkifhBNBRRNaIab/Np7XD61siR8zNMG/vCKjFGICnp andrew@localhost")
	publicKey, _, _, _, _ := ssh.ParseAuthorizedKey(key)
	in, out := sshTunnel.formatHostFingerprint(publicKey), "65:30:38:96:35:56:4f:64:64:e8:e3:a4:7d:59:3e:19"
	if in != out {
		t.Errorf("sshTunnel.formatHostFingerprint(key) = %v, want %v", in, out)
	}
}

func TestCanUpdateTunnelAddress(t *testing.T) {
	in, out :=  string(sshTunnel.updateOrAddTunnelAddress([]byte(`
<?xml version="1.0" encoding="UTF-8"?>
<slave>
  <name>x123</name>
  <description>JSL auto generated node &apos;x123&apos;.</description>
  <remoteFS>C:\JenkinsSlaveLauncher</remoteFS>
  <numExecutors>1</numExecutors>
  <mode>EXCLUSIVE</mode>
  <retentionStrategy class="hudson.slaves.RetentionStrategy$Always"/>
  <launcher class="hudson.slaves.JNLPLauncher">
    <tunnel>127.0.0.1:54218</tunnel>
  </launcher>
  <label>JSL windows amd64</label>
  <nodeProperties/>
</slave>`), "[::1]:12345")),	`
<?xml version="1.0" encoding="UTF-8"?>
<slave>
  <name>x123</name>
  <description>JSL auto generated node &apos;x123&apos;.</description>
  <remoteFS>C:\JenkinsSlaveLauncher</remoteFS>
  <numExecutors>1</numExecutors>
  <mode>EXCLUSIVE</mode>
  <retentionStrategy class="hudson.slaves.RetentionStrategy$Always"/>
  <launcher class="hudson.slaves.JNLPLauncher">
    <tunnel>[::1]:12345</tunnel>
  </launcher>
  <label>JSL windows amd64</label>
  <nodeProperties/>
</slave>`

	if in != out {
		t.Errorf("sshTunnel.updateOrAddTunnelAddress(configXML, hostAndPort) = %v, want %v", in, out)
	}
}

func TestCanAddTunnelAddressVariant1(t *testing.T) {
	in, out := string(sshTunnel.updateOrAddTunnelAddress([]byte(`
<?xml version="1.0" encoding="UTF-8"?>
<slave>
  <name>x123</name>
  <description>JSL auto generated node &apos;x123&apos;.</description>
  <remoteFS>C:\JenkinsSlaveLauncher</remoteFS>
  <numExecutors>1</numExecutors>
  <mode>EXCLUSIVE</mode>
  <retentionStrategy class="hudson.slaves.RetentionStrategy$Always"/>
  <launcher class="hudson.slaves.JNLPLauncher"/>
  <label>JSL windows amd64</label>
  <nodeProperties/>
</slave>`), "[::1]:12345")),	`
<?xml version="1.0" encoding="UTF-8"?>
<slave>
  <name>x123</name>
  <description>JSL auto generated node &apos;x123&apos;.</description>
  <remoteFS>C:\JenkinsSlaveLauncher</remoteFS>
  <numExecutors>1</numExecutors>
  <mode>EXCLUSIVE</mode>
  <retentionStrategy class="hudson.slaves.RetentionStrategy$Always"/>
  <launcher class="hudson.slaves.JNLPLauncher">
    <tunnel>[::1]:12345</tunnel>
  </launcher>
  <label>JSL windows amd64</label>
  <nodeProperties/>
</slave>`

	if in != out {
		t.Errorf("sshTunnel.updateOrAddTunnelAddress(configXML, hostAndPort) = %v, want %v", in, out)
	}
}

func TestCanAddTunnelAddressVariant2(t *testing.T) {
	in, out := string(sshTunnel.updateOrAddTunnelAddress([]byte(`
<?xml version="1.0" encoding="UTF-8"?>
<slave>
  <name>x123</name>
  <description>JSL auto generated node &apos;x123&apos;.</description>
  <remoteFS>C:\JenkinsSlaveLauncher</remoteFS>
  <numExecutors>1</numExecutors>
  <mode>EXCLUSIVE</mode>
  <retentionStrategy class="hudson.slaves.RetentionStrategy$Always"/>
  <launcher class="hudson.slaves.JNLPLauncher">
    <someproperty/>
  </launcher>
  <label>JSL windows amd64</label>
  <nodeProperties/>
</slave>`), "[::1]:12345")),	`
<?xml version="1.0" encoding="UTF-8"?>
<slave>
  <name>x123</name>
  <description>JSL auto generated node &apos;x123&apos;.</description>
  <remoteFS>C:\JenkinsSlaveLauncher</remoteFS>
  <numExecutors>1</numExecutors>
  <mode>EXCLUSIVE</mode>
  <retentionStrategy class="hudson.slaves.RetentionStrategy$Always"/>
  <launcher class="hudson.slaves.JNLPLauncher">
    <someproperty/>
    <tunnel>[::1]:12345</tunnel>
  </launcher>
  <label>JSL windows amd64</label>
  <nodeProperties/>
</slave>`

	if in != out {
		t.Errorf("sshTunnel.updateOrAddTunnelAddress(configXML, hostAndPort) = %v, want %v", in, out)
	}
}
