// Copyright 2014 The jenkins-client-launcher Authors. All rights reserved.
// Use of this source code is governed by a MIT license that can be found in the LICENSE file.

package environment

import (
	"github.com/jkellerer/jenkins-client-launcher/launcher/modes"
	"github.com/jkellerer/jenkins-client-launcher/launcher/util"
	"code.google.com/p/go.crypto/ssh"
	"crypto/md5"
	"encoding/hex"
	"regexp"
	"net"
	"fmt"
	"io"
	"net/url"
	"net/http"
	"strings"
	"io/ioutil"
	"encoding/xml"
	"bytes"
)

// Implements the establishing of an SSH tunnel between the node and the jenkins server.
type SSHTunnelEstablisher struct {
	closables []io.Closer
}

// Creates a new tunnel establisher.
func NewSSHTunnelEstablisher(registerInMode bool) *SSHTunnelEstablisher {
	self := new(SSHTunnelEstablisher)
	self.closables = []io.Closer{}

	if registerInMode {
		modes.RegisterModeListener(func(mode modes.ExecutableMode, nextStatus int32, config *util.Config) {
			if !config.CITunnelSSHEnabled || config.CITunnelSSHAddress == "" || mode.Name() != "client" || !config.HasCIConnection() {
				return
			}

			if nextStatus == modes.ModeStarting {
				self.setupSSHTunnel(config)
			} else if nextStatus == modes.ModeStopped {
				self.tearDownSSHTunnel(config)
			}
		})
	}

	return self
}

func (self *SSHTunnelEstablisher) Name() string {
	return "SSH Tunnel Establisher"
}

func (self *SSHTunnelEstablisher) IsConfigAcceptable(config *util.Config) (bool) {
	if config.CITunnelSSHEnabled && config.CITunnelSSHAddress == "" {
		util.GOut("ssh-tunnel", "WARN: SSH tunnel is enabled but SSH server address is empty.");
		return false
	}
	if config.CITunnelSSHAddress != "" && !config.HasCIConnection() {
		util.GOut("ssh-tunnel", "WARN: No Jenkins URI defined. SSH tunnel settings are not enough to connect to Jenkins.");
		return false
	}
	return true
}

func (self *SSHTunnelEstablisher) Prepare(config *util.Config) {
	// Nothing to do in prepare.
}

// Closes a previously opened SSL connection.
func (self *SSHTunnelEstablisher) tearDownSSHTunnel(config *util.Config) {
	if self.closables != nil && len(self.closables) > 0 {
		for i := len(self.closables) - 1; i >= 0; i-- {
			self.closables[i].Close()
		}
		self.closables = self.closables[0:0]

		// Reset tunnel again.
		self.applyTunnelAddress(config, "")
	}
}

// Opens a new SSH connection, opens a local server port and forwards it to the JNLP port on Jenkins.
func (self *SSHTunnelEstablisher) setupSSHTunnel(config *util.Config) {
	// Ensure no other SSL connections are still open.
	self.tearDownSSHTunnel(config)

	// Configuring the SSH client
	clientConfig := &ssh.ClientConfig{
		User: config.CITunnelSSHUsername,
		Auth: []ssh.AuthMethod{
			ssh.Password(config.CITunnelSSHPassword),
		},
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			expected, actual := config.CITunnelSSHFingerprint, self.formatHostFingerprint(key)
			if actual != expected && expected != "-" {
				if expected == "" {
					return fmt.Errorf("The host fingerprint of '%v' is '%v'. Please add this to the configuration in order to connect.", hostname, actual)
				} else {
					return fmt.Errorf("The host fingerprint of '%v' is '%v' while '%v' was expected. Connection aborted.", hostname, actual, expected)
				}
			}
			return nil
		},
	}

	// Connecting to the SSH host
	if config.CITunnelSSHPort == 0 { config.CITunnelSSHPort = 22 }
	sshAddress := fmt.Sprintf("%v:%v", config.CITunnelSSHAddress, config.CITunnelSSHPort)

	sshClient, err := ssh.Dial("tcp", sshAddress, clientConfig)
	if err == nil {
		self.closables = append(self.closables, sshClient)
		util.GOut("ssh-tunnel", "Successfully connected with '%v'.", sshAddress)
	} else {
		util.GOut("ssh-tunnel", "ERROR: Failed connecting with %v. Cause: %v", sshAddress, err)
		return
	}

	// Creating a local server listener port to use for forwarding.
	serverListener, err := net.Listen("tcp", "localhost:0")
	if err == nil {
		self.closables = append(self.closables, serverListener)
		util.GOut("ssh-tunnel", "Opened local listener on '%v'.", serverListener.Addr())

		if err = self.applyTunnelAddress(config, serverListener.Addr().String()); err != nil {
			util.GOut("ssh-tunnel", "ERROR: Failed configuring local listener as tunnel inside Jenkins. Cause: %v", err)
			return
		}
	} else {
		util.GOut("ssh-tunnel", "ERROR: Failed opening local listener. Cause: %v", err)
		return
	}

	// Forward local connections to the JNLP port.
	go self.forwardLocalConnectionsToJNLP(config, serverListener, sshClient)
}

// Forwards the local server listener to the JNLP host:port using the SSH connection as tunnel.
// What this method does is the same as "ssh -L $ANY-PORT:jenkins-host:$JNLP-PORT" jenkins-host.
func (self *SSHTunnelEstablisher) forwardLocalConnectionsToJNLP(config *util.Config, serverListener net.Listener, sshClient *ssh.Client) {
	jnlpAddress, err := self.formatJNLPHostAndPort(config)
	if err != nil {
		util.GOut("ssh-tunnel", "ERROR: Failed fetching JNLP port from '%v'. Cause: %v.", config.CIHostURI, err)
		return
	}

	establishBIDITransport := func(source net.Conn, target net.Conn) {
		transfer := func(source io.ReadCloser, target io.Writer) {
			defer source.Close()
			_, _ = io.Copy(target, source)
		}

		go transfer(source, target)
		go transfer(target, source)
	}

	for {
		if sourceConnection, err := serverListener.Accept(); err == nil {
			if targetConnection, err := sshClient.Dial("tcp", jnlpAddress); err == nil {
				util.GOut("ssh-tunnel", "Forwarding local connection to '%v' via '%v'.", jnlpAddress, sshClient.Conn.RemoteAddr().String())
				establishBIDITransport(sourceConnection, targetConnection)
			} else {
				util.GOut("ssh-tunnel", "ERROR: Failed forwarding incoming local connection to '%v' via '%v'.", jnlpAddress, sshClient.Conn.RemoteAddr().String())
			}
		} else {
			util.GOut("ssh-tunnel", "ERROR: Failed accepting next incoming local connection, assuming connection was closed.")
			return
		}
	}
}

// Returns "hort:port" of the JNLP server listener.
func (self *SSHTunnelEstablisher) formatJNLPHostAndPort(config *util.Config) (jnlpAddress string, err error) {
	jnlpHost := "localhost"
	jenkinsUrl, err := url.Parse(config.CIHostURI)
	if err != nil {
		util.GOut("ssh-tunnel", "ERROR: Failed extracting host out of url '%v'. Cause: %v.", config.CIHostURI, err)
	} else {
		jnlpHost = jenkinsUrl.Host
		if containsPort, _ := regexp.MatchString("^.+:[0-9]+$", jnlpHost); containsPort {
			jnlpHost = jnlpHost[0:strings.LastIndex(jnlpHost, ":")]
		}
	}

	if jnlpPort, err := self.getJNLPListenerPort(config); err == nil {
		jnlpAddress = fmt.Sprintf("%s:%s", jnlpHost, jnlpPort)
	} else {
		return "", err
	}

	return
}

// Gets the port that is used by the JNLP client to communicate with Jenkins.
// Returns the port number as string and an error if fetching the port failed for any reason.
func (self *SSHTunnelEstablisher) getJNLPListenerPort(config *util.Config) (port string, err error) {
	var response *http.Response
	port = ""

	if response, err = config.CIGet("/tcpSlaveAgentListener/"); err == nil {
		response.Body.Close()
		if response.StatusCode == 200 {
			if port = response.Header.Get("X-Jenkins-JNLP-Port"); port == "" {
				port = response.Header.Get("X-Hudson-JNLP-Port")
			}
		}

		if port == "" {
			err = fmt.Errorf("Jenkins did not provide the JNLP-Port, the reply was %v.", response.Status)
		}
	}

	return
}

var twoDigitHexMatcher, _ = regexp.Compile("([0-9a-z]{2})")

// Returns the fingerprint of the SSH server's public key as string (using the same algorithm as the ssh client).
func (self *SSHTunnelEstablisher) formatHostFingerprint(key ssh.PublicKey) string {
	value := key.Marshal()
	valueMD5 := md5.Sum(value)
	fingerprint := hex.EncodeToString([]byte(valueMD5[:]))
	return twoDigitHexMatcher.ReplaceAllString(fingerprint, "$1:")[0:len(valueMD5)*3-1]
}

var tunnelReplacer, _ = regexp.Compile("(?i)(<tunnel>)(.*?)(</tunnel>)")
var tunnelReplacement = "${1}%s${3}"
var singleLauncherReplacer, _ = regexp.Compile("(?i)(<launcher[^>]+?)(/>)")
var singleLauncherReplacement = "${1}>\n    <tunnel>%s</tunnel>\n  </launcher>"
var launcherReplacer, _ = regexp.Compile("(?i)(</launcher>)")
var launcherReplacement = "  <tunnel>%s</tunnel>\n  ${1}"

// Updates or adds the local tunnel address:port to the specified Jenkins node config XML.
func (self *SSHTunnelEstablisher) updateOrAddTunnelAddress(configXML []byte, hostAndPort string) (updatedXML []byte) {
	buffer := bytes.NewBuffer(make([]byte, 0, len(hostAndPort)+10))
	if err := xml.EscapeText(buffer, []byte(hostAndPort)); err == nil {
		hostAndPort = string(buffer.Bytes())
	}

	updatedXML = tunnelReplacer.ReplaceAll(configXML, []byte(fmt.Sprintf(tunnelReplacement, hostAndPort)))
	if bytes.Equal(updatedXML, configXML) {
		updatedXML = singleLauncherReplacer.ReplaceAll(configXML, []byte(fmt.Sprintf(singleLauncherReplacement, hostAndPort)))
		if bytes.Equal(updatedXML, configXML) {
			updatedXML = launcherReplacer.ReplaceAll(configXML, []byte(fmt.Sprintf(launcherReplacement, hostAndPort)))
		}
	}

	return
}

// Applies the local tunnel address:port to the settings of the node inside Jenkins.
func (self *SSHTunnelEstablisher) applyTunnelAddress(config *util.Config, hostAndPort string) error {
	nodeConfigPath := fmt.Sprintf("/computer/%s/config.xml", config.ClientName)

	if response, err := config.CIGet(nodeConfigPath); err == nil {
		defer response.Body.Close()
		if configXML, err := ioutil.ReadAll(response.Body); err == nil {
			// If we have changes, we apply them now.
			if updatedXML := self.updateOrAddTunnelAddress(configXML, hostAndPort); !bytes.Equal(updatedXML, configXML) {
				success := false
				failedMessage := ""

				if request, err := config.CIRequest("POST", nodeConfigPath, bytes.NewReader(updatedXML)); err == nil {
					request.Header.Add("Content-Type", "application/xml")
					if response, err := config.CIClient().Do(request); err == nil {
						response.Body.Close()
						success = response.StatusCode == 200
						failedMessage = fmt.Sprintf("%v", response.Status)
					} else {
						failedMessage = fmt.Sprintf("%v", err)
					}
				} else {
					failedMessage = fmt.Sprintf("%v", err)
				}

				if success {
					util.GOut("ssh-tunnel", "Updated node '%s' to tunnel connections via '%s'", config.ClientName, hostAndPort)
				} else {
					return fmt.Errorf(failedMessage)
				}
			}
			return nil
		} else {
			return err
		}
	} else {
		return err
	}
}

// Registering the tunnel establisher.
var _ = RegisterPreparer(NewSSHTunnelEstablisher(true))
