// Copyright 2014 The jenkins-client-launcher Authors. All rights reserved.
// Use of this source code is governed by a MIT license that can be found in the LICENSE file.

package modes

import (
	"code.google.com/p/go.crypto/ssh/terminal"
	"code.google.com/p/go.crypto/ssh"
	"fmt"
	"io/ioutil"
	"net"
	"time"
	"github.com/jkellerer/jenkins-client-launcher/launcher/util"
	"os"
)

type ServerMode struct {
	status *util.AtomicInt32
	config *util.Config
}

func NewServerMode() *ServerMode {
	r := new(ServerMode)
	r.status = util.NewAtomicInt32()
	return r
}

func (self *ServerMode) Name() (string) {
	return "ssh-server"
}

func (self *ServerMode) Status() (*util.AtomicInt32) {
	return self.status
}

func (self *ServerMode) IsConfigAcceptable(config *util.Config) (bool) {
	// TODO
	return true;
}

func (self *ServerMode) Start(config *util.Config) (error) {
	if self.status.Get() != ModeNone {
		panic("Cannot start mode whose state is != ModeNone")
	}

	self.config = config;
	self.status.Set(ModeStarting)

	var keySearchPaths = []string {
		"id_rsa",
		"id_dsa",
			os.Getenv("home") + "/.ssh/id_rsa",
			os.Getenv("home") + "/.ssh/id_dsa",
				os.Getenv("homedrive") + os.Getenv("homepath") + "/.ssh/id_rsa",
				os.Getenv("homedrive") + os.Getenv("homepath") + "/.ssh/id_dsa",
	}

	var key ssh.Signer = nil
	var err error = nil

	for _, keyPath := range keySearchPaths {
		key, err = self.readPrivateKey(keyPath)

		if (err == nil) {
			break
		}

		if _, pathError := err.(*os.PathError); !pathError {
			break
		}
	}

	if (err == nil) {
		go self.execute(key)
		return nil
	} else {
		return err
	}
}

func (self *ServerMode) Stop() {
	self.status.Set(ModeStopping)
}

func (self *ServerMode) createPrivateKey(privateKeyFile string) (*ssh.Signer, error) {
	panic("Function not implemented.")
}

func (self *ServerMode) readPrivateKey(privateKeyFile string) (ssh.Signer, error) {
	var private ssh.Signer = nil
	privateBytes, err := ioutil.ReadFile(privateKeyFile)

	if err == nil {
		private, err = ssh.ParsePrivateKey(privateBytes)
	}

	return private, err
}

func (self *ServerMode) execute(privateKey ssh.Signer) {
	// An SSH server is represented by a ServerConfig, which holds
	// certificate details and handles authentication of ServerConns.
	config := &ssh.ServerConfig{
		PasswordCallback: func(c ssh.ConnMetadata, pass []byte) (*ssh.Permissions, error) {
			// Slowing down password check to make BF attacks more difficult.
			time.Sleep(time.Second * 1)

			if c.User() == self.config.SSHUsername && string(pass) == self.config.SSHPassword {
				return nil, nil
			} else {
				return nil, fmt.Errorf("SSH: Password rejected for %q", c.User())
			}
		},
		AuthLogCallback: func(c ssh.ConnMetadata, method string, err error) {
			if (err == nil) {
				util.GOut("SSH", "Authentication succeeded '%v' using '%v'", c.User(), method)
			} else {
				util.GOut("SSH", "Failed attempt to authenticate '%v' using '%v' ; Caused by: %v", c.User(), method, err)
			}
		},
	}

	config.AddHostKey(privateKey)

	// Once a ServerConfig has been configured, connections can be
	// accepted.
	address := fmt.Sprintf("%v:%v", self.config.SSHListenAddress, self.config.SSHListenPort)

	util.GOut("SSH", "Starting to listen @ %v", address)

	listener, err := net.Listen("tcp", address)
	if err != nil {
		panic("SSH: Failed to listen @ " + address)
	} else {
		defer func() {
			listener.Close()
			self.status.Set(ModeStopped)
		}()
	}

	go func() {
		for {
			connection, err := listener.Accept()
			if err != nil {
				util.GOut("SSH", "Failed to accept next incoming SSH connection, assuming connection was closed.")
				return
			}

			// Handling only one connection at a time should be enough.
			self.handleSSHRequest(&connection, config)
		}
	}()

	// Entering main loop and remain there until the terminal is stopped and the deferred channel close is triggered.
	self.status.Set(ModeStarted)
	for self.status.Get() == ModeStarted {
		time.Sleep(time.Millisecond * 100)
	}
}

func (self *ServerMode) handleSSHRequest(connection *net.Conn, config *ssh.ServerConfig) {

	// TODO: Check if we need to close anything.

	// Before use, a handshake must be performed on the incoming net.Conn.
	_, channels, requests, err := ssh.NewServerConn(*connection, config)
	if err != nil {
		panic("failed to handshake")
	}

	// The incoming Request channel must be serviced.
	go ssh.DiscardRequests(requests)

	// Service the incoming Channel channel.
	for newChannel := range channels {
		// Channels have a type, depending on the application level protocol intended. In the case of a shell, the type is
		// "session" and ServerShell may be used to present a simple terminal interface.
		if newChannel.ChannelType() != "session" {
			newChannel.Reject(ssh.UnknownChannelType, "unknown channel type")
			continue
		}

		channel, requests, err := newChannel.Accept()
		if err != nil {
			panic("could not accept channel.")
		}

		// Sessions have out-of-band requests such as "shell", "pty-req" and "env".
		// Here we handle only the "shell" request.
		go func(in <-chan *ssh.Request) {
			for req := range in {
				ok := false
				switch req.Type {
				case "shell":
					ok = true
					if len(req.Payload) > 0 {
						// We don't accept any commands, only the default shell.
						ok = false
					}
				}
				req.Reply(ok, nil)
			}
		}(requests)

		term := terminal.NewTerminal(channel, "> ")

		go func() {
			defer channel.Close()
			for {
				line, err := term.ReadLine()
				if err != nil {
					break
				}
				// TODO: Likely we need to interpret the incoming commands in here.
				fmt.Println("INPUT-SSH:" + line)
			}
		}()
	}
}

// Registering the server mode.
var _ = RegisterMode(NewServerMode())
