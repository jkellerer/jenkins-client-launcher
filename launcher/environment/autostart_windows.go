// Copyright 2014 The jenkins-client-launcher Authors. All rights reserved.
// Use of this source code is governed by a MIT license that can be found in the LICENSE file.

package environment

import (
	"fmt"
	"os"
	"io/ioutil"
	"path/filepath"
	"os/exec"
	"sync"
	util "github.com/jenkins-client-launcher/launcher/util"
)

const (
	AutostartRegKey = `HKEY_CURRENT_USER\Software\Microsoft\Windows\CurrentVersion\Run\JenkinsClientLauncher`

	RegReadScript = `try {
		WScript.CreateObject("WScript.Shell").RegRead(WScript.Arguments(0))
		WScript.Quit(0)
	} catch (e) {
		WScript.Echo(e.message); WScript.Quit(1)
	}`

	RegWriteScript = `try {
		WScript.CreateObject("WScript.Shell").RegWrite(WScript.Arguments(0), WScript.StdIn.ReadLine(), "REG_SZ")
		WScript.Quit(0)
	} catch (e) {
		WScript.Echo(e.message); WScript.Quit(1)
	}`

	RegDeleteScript = `try {
		WScript.CreateObject("WScript.Shell").RegDelete(WScript.Arguments(0))
		WScript.Quit(0)
	} catch (e) {
		WScript.Echo(e.message); WScript.Quit(1)
	}`
)

// Implements "autostart" functionality for windows.
type AutostartHandler struct {
	util.AnyConfigAcceptor
	commandline string
	scriptMutex *sync.Mutex
}

// Creates a new autostart handler for windows.
func NewAutostartHandler() *AutostartHandler {
	self := new(AutostartHandler)
	self.scriptMutex = new(sync.Mutex)
	self.commandline = ""

	return self
}

func (self *AutostartHandler) Name() string {
	return "Autostart Handler"
}

// Performs registration & deregistration for autostart on windows.
func (self *AutostartHandler) Prepare(config *util.Config) {
	cwd, _ := os.Getwd()
	self.commandline = fmt.Sprintf("\"%s\" \"-directory=%s\"", os.Args[0], cwd)

	if config.Autostart {
		util.GOut("autostart", "Registering %v for autostart.", self.commandline)
		err := self.register()
		if (err != nil) {
			util.GOut("autostart", "FAILED to register for autostart. Cause: %v", err)
		}
	} else {
		if self.isRegistered() {
			util.GOut("autostart", "Unregistering %v from autostart.", self.commandline)
			err := self.unregister()
			if (err != nil) {
				util.GOut("autostart", "FAILED to unregister from autostart. Cause: %v", err)
			}
		}
	}
}

// Returns true when registered for autostart.
func (self *AutostartHandler) isRegistered() bool {
	err := self.withScriptFile(RegReadScript, func(scriptFile string) error {
			return self.execute("cscript", "//Nologo", scriptFile, AutostartRegKey)
		})

	return err == nil
}

// Registers for autostart.
func (self *AutostartHandler) register() error {
	return self.withScriptFile(RegWriteScript, func(scriptFile string) error {
			return self.executeI(self.commandline, "cscript", "//Nologo", scriptFile, AutostartRegKey)
		})
}

// Unregisters for autostart.
func (self *AutostartHandler) unregister() error {
	return self.withScriptFile(RegDeleteScript, func(scriptFile string) error {
			return self.execute("cscript", "//Nologo", scriptFile, AutostartRegKey)
		})
}

// Executes the specified callback with a temporary script file which was created with the given scriptContent.
func (self *AutostartHandler) withScriptFile(scriptContent string, callback func(scriptFile string) error) error {
	self.scriptMutex.Lock();
	defer self.scriptMutex.Unlock()

	tempFile, err := filepath.Abs("~autostart-script.js")
	if (err == nil) {
		defer os.Remove(tempFile)
		err = ioutil.WriteFile(tempFile, []byte(scriptContent), os.ModeTemporary)
		if err == nil {
			err = callback(tempFile)
		}
	}

	return err
}

// Executes the specified command.
// Returns an error when the execution was not successful (e.g. return code != 0).
func (self *AutostartHandler) execute(args ...string) error {
	return self.executeI("", args...)
}

// Executes the specified command passing stdin to stdin of the process.
// Returns an error when the execution was not successful (e.g. return code != 0).
func (self *AutostartHandler) executeI(stdin string, args ...string) error {
	cmd := exec.Command(os.Getenv("ComSpec"), append([]string{"/c"}, args...)...)

	if stdin != "" {
		if pin, err := cmd.StdinPipe(); err == nil {
			_, _ = pin.Write([]byte(stdin))
			_, _ = pin.Write([]byte("\n"))
			_ = pin.Close()
		} else {
			panic(fmt.Sprintf("Failed opening stdin pipe when executing %v", args))
		}
	}

	err := cmd.Run()
	//output, err := cmd.CombinedOutput()
	//fmt.Println(string(output))

	if err == nil && cmd.ProcessState.Success() {
		return nil
	}
	return err
}

// Registering the handler.
var _ = RegisterPreparer(NewAutostartHandler())
