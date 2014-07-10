// Copyright 2014 The jenkins-client-launcher Authors. All rights reserved.
// Use of this source code is governed by a MIT license that can be found in the LICENSE file.

package modes

import (
	"time"
	"github.com/jkellerer/jenkins-client-launcher/launcher/util"
	"fmt"
	"io/ioutil"
	"regexp"
	"os/exec"
	"io"
	"os"
	"bufio"
)

/** Jenkins Client CLI help:
-auth user:pass                 : If your Jenkins is security-enabled, specify
                                   a valid user name and password.
 -connectTo HOST:PORT            : make a TCP connection to the given host and
                                   port, then start communication.
 -cp (-classpath) PATH           : add the given classpath elements to the
                                   system classloader.
 -jar-cache DIR                  : Cache directory that stores jar files sent
                                   from the master
 -jnlpCredentials USER:PASSWORD  : HTTP BASIC AUTH header to pass in for making
                                   HTTP requests.
 -jnlpUrl URL                    : instead of talking to the master via
                                   stdin/stdout, emulate a JNLP client by
                                   making a TCP connection to the master.
                                   Connection parameters are obtained by
                                   parsing the JNLP file.
 -noReconnect                    : Doesn't try to reconnect when a communication
                                   fail, and exit instead
 -proxyCredentials USER:PASSWORD : HTTP BASIC AUTH header to pass in for making
                                   HTTP authenticated proxy requests.
 -secret HEX_SECRET              : Slave connection secret to use instead of
                                   -jnlpCredentials.
 -slaveLog FILE                  : create local slave error log
 -tcp FILE                       : instead of talking to the master via
                                   stdin/stdout, listens to a random local
                                   port, write that port number to the given
                                   file, then wait for the master to connect to
                                   that port.
 -text                           : encode communication with the master with
                                   base64. Useful for running slave over 8-bit
                                   unsafe protocol like telnet`
**/

type ClientMode struct {
	status *util.AtomicInt32
}

func NewClientMode() *ClientMode {
	r := new(ClientMode)
	r.status = new(util.AtomicInt32)
	return r
}

func (self *ClientMode) Name() (string) {
	return "client"
}

func (self *ClientMode) Status() *util.AtomicInt32 {
	return self.status
}

func (self *ClientMode) IsConfigAcceptable(config *util.Config) (bool) {
	if !config.HasCIConnection() {
		util.GOut(self.Name(), "No Jenkins URI defined. Cannot connect to the CI server.")
		return false
	}

	if config.SecretKey == "" {
		if config.SecretKey = self.getSecretFromJenkins(config); config.SecretKey == "" {
			util.GOut(self.Name(), "No secret key set for node %v and the attempt to fetch it from Jenkins failed.", config.ClientName)
			return false
		}
	}

	return true
}

func (self *ClientMode) Start(config *util.Config) (error) {
	if !self.isStopped() {
		panic(fmt.Sprintf("Cannot start mode whose state is != ModeNone && != ModeStopped, was %v", self.status))
	}

	self.status.Set(ModeStarting)
	go self.execute(config)

	return nil
}

func (self *ClientMode) Stop() {
	if !self.isStopped() {
		self.status.Set(ModeStopping)
	}
}

func (self *ClientMode) isStopped() bool {
	return self.status.Get() == ModeNone || self.status.Get() == ModeStopped
}

func (self *ClientMode) execute(config *util.Config) {
	commandline := []string{}
	commandline = append(commandline, util.JavaArgs...)
	commandline = append(commandline, config.JavaArgs...)
	commandline = append(commandline,
		"-jar", util.ClientJar,
		"-jnlpUrl", fmt.Sprintf("%v/computer/%v/slave-agent.jnlp", config.CIHostURI, config.ClientName))

	if config.SecretKey != "" {
		commandline = append(commandline, "-secret", config.SecretKey)
	}

	if config.CIAcceptAnyCert {
		commandline = append(commandline, "-noCertificateCheck")
	}

	if config.HandleReconnectsInLauncher {
		commandline = append(commandline, "-noReconnect")
	}

	if config.CIUsername != "" && config.CIPassword != "" && config.PassCIAuth {
		commandline = append(commandline, "-auth", fmt.Sprintf("%s:%s", config.CIUsername, config.CIPassword))
		commandline = append(commandline, "-jnlpCredentials", fmt.Sprintf("%s:%s", config.CIUsername, config.CIPassword))
	}

	stoppingClient, clientStopped := make(chan bool), make(chan bool)

	go func() {
		command := exec.Command(util.Java, commandline...)
		if pOut, err := command.StdoutPipe(); err == nil {
			go self.redirectConsoleOutput(config, pOut, os.Stdout)
		} else {
			panic("Failed connecting stdout with console")
		}

		if pErr, err := command.StderrPipe(); err == nil {
			go self.redirectConsoleOutput(config, pErr, os.Stderr)
		} else {
			panic("Failed connecting stderr with console")
		}

		util.GOut("client", "Starting: %s", append([]string{util.Java}, commandline...))

		if err := command.Start(); err != nil {
			util.GOut("client", "Jenkins client failed to start with %v", err)
		} else {
			util.GOut("client", "Jenkins client was started.")

			go func() {
				<-stoppingClient
				command.Process.Kill()
				time.Sleep(time.Second * 1)
			}()

			if err := command.Wait(); err != nil {
				util.GOut("client", "Jenkins client quit with %v", err)
			} else {
				util.GOut("client", "Jenkins client was stopped.")
			}

			self.status.Set(ModeStopped)
			clientStopped<-true
		}
	}()

	// Entering main loop
	self.status.Set(ModeStarted)

	for self.status.Get() == ModeStarted {
		time.Sleep(time.Millisecond * 100)
	}

	stoppingClient <- true
	<-clientStopped

	self.status.Set(ModeStopped)
}

func (self *ClientMode) redirectConsoleOutput(config *util.Config, input io.ReadCloser, output io.Writer) {
	defer input.Close()
	reader := bufio.NewReader(input)

	for {
		line, isPrefix, err := reader.ReadLine()

		if len(line) > 0 {
			// Send to output
			output.Write(line)

			if !isPrefix {
				output.Write([]byte("\n"))
			}

			if config.ClientMonitorConsole && config.ConsoleMonitor.IsRestartTriggered(string(line)) {
				util.GOut("client", "RESTART TOKEN found. Client state may be invalid, forcing a restart.")
				go func() {
					time.Sleep(time.Second * 1)
					self.Stop()
				}()
			}
		}

		if err != nil {
			return
		}
	}
}

func (self *ClientMode) getSecretFromJenkins(config *util.Config) string {
	response, err := config.CIGet(fmt.Sprintf("computer/%s/", config.ClientName))
	if err == nil {
		defer response.Body.Close()

		if response.StatusCode == 200 {
			var content []byte
			if content, err = ioutil.ReadAll(response.Body); err == nil {
				return self.extractSecret(content)
			}
		} else {
			util.GOut("client", "Failed fetching secret key from Jenkins. Cause: %v", response.Status)
		}
	}

	if err != nil {
		util.GOut("client", "Failed fetching secret key from Jenkins. Cause: %v", err)
	}

	return ""
}

func (self *ClientMode) extractSecret(content []byte) string {
	if pattern, err := regexp.Compile(`(?i)<pre>.*-secret ([A-F0-9]+)[^A-F0-9]*</pre>`); err == nil {
		if matches := pattern.FindSubmatch(content); matches != nil && len(matches) == 2 {
			return string(matches[1])
		}
	}
	return ""
}

// Registering the client mode.
var _ = RegisterMode(NewClientMode())
