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
	"math"
	"time"
)

// The interval when the the SSH tunnel tries to reach Jenkins to see if the SSH tunnel is still alive.
var nodeSshTunnelAliveMonitoringInterval = time.Second * 30

// Implements the establishing of an SSH tunnel between the node and the jenkins server.
type SSHTunnelEstablisher struct {
	closables []io.Closer
	ciHostURL *url.URL

	aliveTicker *time.Ticker
	aliveTickEvaluator *time.Ticker
	expectedAliveTick *util.AtomicInt32
	lastAliveTick *util.AtomicInt32
	tunnelConnected *util.AtomicBoolean
}

// Creates a new tunnel establisher.
func NewSSHTunnelEstablisher(registerInMode bool) *SSHTunnelEstablisher {
	self := new(SSHTunnelEstablisher)
	self.closables = []io.Closer{}
	self.ciHostURL = nil

	self.aliveTicker, self.aliveTickEvaluator = time.NewTicker(nodeSshTunnelAliveMonitoringInterval), time.NewTicker(nodeSshTunnelAliveMonitoringInterval)
	self.expectedAliveTick, self.lastAliveTick = util.NewAtomicInt32(), util.NewAtomicInt32()
	self.tunnelConnected = util.NewAtomicBoolean()

	if registerInMode {
		modes.RegisterModeListener(func(mode modes.ExecutableMode, nextStatus int32, config *util.Config) {
			if !config.CITunnelSSHEnabled || config.CITunnelSSHAddress == "" || mode.Name() != "client" || !config.HasCIConnection() {
				return
			}

			if nextStatus == modes.ModeStarting {
				var err error;
				if self.ciHostURL, err = url.Parse(config.CIHostURI); err != nil {
					util.GOut("ssh-tunnel", "ERROR: Failed parsing Jenkins URI. Cannot tunnel connections to Jenkins. Cause: %v", err)
					return
				}

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
	self.startAliveStateMonitoring(config)
}

// Monitors that the tunnel is alive by periodically querying the node status off Jenkins.
// Timeout, hanging connections or connection errors lead to a restart of the current execution mode (which implicitly closes SSH tunnel as well).
func (self *SSHTunnelEstablisher) startAliveStateMonitoring(config *util.Config) {
	// Periodically check the node status and increment lastAliveTick on success
	go func() {
		for _ = range self.aliveTicker.C {
			if !self.tunnelConnected.Get() { continue }

			if _, err := GetJenkinsNodeStatus(config); err == nil {
				self.lastAliveTick.Set(self.expectedAliveTick.Get());
			}
		}
	}()

	// Periodically check that lastAliveTick was incremented.
	go func() {
		for _ = range self.aliveTickEvaluator.C {
			if !self.tunnelConnected.Get() { continue }

			if math.Abs(float64(self.expectedAliveTick.Get() - self.lastAliveTick.Get())) > 1 {
				util.GOut("ssh-tunnel", "WARN: The SSH tunnel appears to be dead or Jenkins is gone. Forcing restart of client and SSH tunnel.")
				modes.GetConfiguredMode(config).Stop()
			} else {
				self.expectedAliveTick.AddAndGet(1)
			}
		}
	}()
}

func (self *SSHTunnelEstablisher) resetAliveStateMonitoring(config *util.Config) {
	self.lastAliveTick.Set(0)
	self.expectedAliveTick.Set(0)
}

// Closes a previously opened SSL connection.
func (self *SSHTunnelEstablisher) tearDownSSHTunnel(config *util.Config) {
	defer self.tunnelConnected.Set(false)

	if self.ciHostURL == nil {
		return
	}

	self.resetAliveStateMonitoring(config)

	config.CIHostURI = self.ciHostURL.String()

	if self.closables != nil && len(self.closables) > 0 {
		for i := len(self.closables) - 1; i >= 0; i-- {
			self.closables[i].Close()
		}
		self.closables = self.closables[0:0]
	}
}

// Opens a new SSH connection, local server ports (JNLP, HTTP) and forwards it to the corresponding ports on Jenkins.
func (self *SSHTunnelEstablisher) setupSSHTunnel(config *util.Config) {
	if self.ciHostURL == nil {
		return
	}

	defer self.tunnelConnected.Set(true)

	if !config.PassCIAuth && config.SecretKey != "" {
		util.GOut("ssh-tunnel", "WARN: Secret key is not supported in combination with SSH tunnel. Implicitly setting %v to %v", "client>passAuth", "true");
		config.PassCIAuth = true
	}

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

	// Fetching target ports
	jnlpTargetAddress, err := self.formatJNLPHostAndPort(config)
	if err != nil {
		util.GOut("ssh-tunnel", "ERROR: Failed fetching JNLP port from '%v'. Cause: %v.", config.CIHostURI, err)
		return
	}

	httpTargetAddress := self.formatHttpHostAndPort();

	// Creating a local server listeners to use for port forwarding.
	httpListener, err1 := self.newLocalServerListener()
	jnlpListener, err2 := self.newLocalServerListener()

	if err1 != nil || err2 != nil {
		self.tearDownSSHTunnel(config)
		return
	}

	// Forward local connections to the HTTP(S)/JNLP ports.
	go self.forwardLocalConnectionsTo(config, sshClient, httpListener, httpTargetAddress)
	go self.forwardLocalConnectionsTo(config, sshClient, jnlpListener, jnlpTargetAddress)

	// Apply the tunnel configuration
	localCiURL, _ := url.Parse(self.ciHostURL.String())
	localCiURL.Host = httpListener.Addr().String()
	config.CIHostURI = localCiURL.String()
	util.JnlpArgs["-url"] = localCiURL.String()
	util.JnlpArgs["-tunnel"] = jnlpListener.Addr().String()
}

// Opens a new local server socket.
func (self *SSHTunnelEstablisher) newLocalServerListener() (serverListener net.Listener, err error) {
	serverListener, err = net.Listen("tcp", "localhost:0")
	if err == nil {
		self.closables = append(self.closables, serverListener)
		util.GOut("ssh-tunnel", "Opened local listener on '%v'.", serverListener.Addr())
	} else {
		util.GOut("ssh-tunnel", "ERROR: Failed opening local listener. Cause: %v", err)
	}
	return
}

// Forwards the local server listener to the specified target address (format host:port) using the SSH connection as tunnel.
// What this method does is the same as "ssh -L $ANY-PORT:jenkins-host:$TARGET-PORT" jenkins-host.
func (self *SSHTunnelEstablisher) forwardLocalConnectionsTo(config *util.Config, ssh *ssh.Client, listener net.Listener, targetAddress string) {
	transfer := func(source io.ReadCloser, target io.Writer) {
		defer source.Close()
		_, _ = io.Copy(target, source)
	}

	establishBIDITransport := func(source net.Conn, target net.Conn) {
		go transfer(source, target)
		go transfer(target, source)
	}

	sshAddress := ssh.Conn.RemoteAddr().String()
	localAddress := listener.Addr().String()

	util.GOut("ssh-tunnel", "Forwarding local connections on '%v' to '%v' via '%v'.", localAddress, targetAddress, sshAddress)

	for {
		if sourceConnection, err := listener.Accept(); err == nil {
			if targetConnection, err := ssh.Dial("tcp", targetAddress); err == nil {
				establishBIDITransport(sourceConnection, targetConnection)
			} else {
				util.GOut("ssh-tunnel", "ERROR: Failed forwarding incoming local connection on '%v' to '%v' via '%v'.", localAddress, targetAddress, sshAddress)
			}
		} else {
			util.GOut("ssh-tunnel", "Stop forwarding local connections on '%v' to '%v'.", localAddress, targetAddress)
			return
		}
	}
}

// Returns "hort:port" of the JNLP server listener on Jenkins.
func (self *SSHTunnelEstablisher) formatJNLPHostAndPort(config *util.Config) (hostAndPort string, err error) {
	jnlpHost := self.ciHostURL.Host
	if self.ciHostContainsPort() {
		jnlpHost = jnlpHost[0:strings.LastIndex(jnlpHost, ":")]
	}

	if jnlpPort, err := self.getJNLPListenerPort(config); err == nil {
		hostAndPort = fmt.Sprintf("%s:%s", jnlpHost, jnlpPort)
	} else {
		return "", err
	}

	return
}

// Returns "hort:port" of the HTTP/S server listener on Jenkins.
func (self *SSHTunnelEstablisher) formatHttpHostAndPort() (hostAndPort string) {
	if !self.ciHostContainsPort() {
		port := 80
		if strings.EqualFold(self.ciHostURL.Scheme, "https") {
			port = 443
		}
		hostAndPort = fmt.Sprintf("%s:%v", self.ciHostURL.Host, port)
	} else {
		hostAndPort = self.ciHostURL.Host
	}
	return
}

func (self *SSHTunnelEstablisher) ciHostContainsPort() (containsPort bool) {
	containsPort, _ = regexp.MatchString("^.+:[0-9]+$", self.ciHostURL.Host)
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

// Registering the tunnel establisher.
var _ = RegisterPreparer(NewSSHTunnelEstablisher(true))
