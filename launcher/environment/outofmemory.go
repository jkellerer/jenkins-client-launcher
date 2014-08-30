// Copyright 2014 The jenkins-client-launcher Authors. All rights reserved.
// Use of this source code is governed by a MIT license that can be found in the LICENSE file.

package environment

import (
	"time"
	"sync"
	"fmt"
	"os"
	"path/filepath"
	"github.com/jkellerer/jenkins-client-launcher/launcher/modes"
	"github.com/jkellerer/jenkins-client-launcher/launcher/util"
)

// Defines an object which triggers a periodic restart of the Jenkins client when enabled.
type OutOfMemoryErrorRestarter struct {
	util.AnyConfigAcceptor
	once *sync.Once
	ticker *time.Ticker
	outOfMemoryErrorMarker string
}

func NewOutOfMemoryErrorRestarter() *OutOfMemoryErrorRestarter {
	p := new(OutOfMemoryErrorRestarter)
	p.once = new(sync.Once)
	return p
}

func (self *OutOfMemoryErrorRestarter) Name() string {
	return "OOM-Error Client Restarter"
}

func (self *OutOfMemoryErrorRestarter) Prepare(config *util.Config) {
	if !config.OutOfMemoryRestartEnabled {
		return
	}

	// Make sure this code runs only once.
	self.once.Do(func() {
		cwd, _ := os.Getwd()
		self.outOfMemoryErrorMarker = filepath.Join(cwd, ".oom-restart")

		util.JavaArgs = append(util.JavaArgs, fmt.Sprintf("-XX:OnOutOfMemoryError=%s", self.createOOMErrorTriggerCommand()))

		// Clearing OOM state when mode status is changing.
		modes.RegisterModeListener(func(mode modes.ExecutableMode, nextStatus int32, config *util.Config) {
			self.oomErrorTriggered()
		})

		self.ticker = time.NewTicker(time.Second*5)

		go func() {
			// Run in schedule
			for _ = range self.ticker.C {
				if self.oomErrorTriggered() {
					util.GOut("OOM", "WARN: A client restart is now triggered as consequence to an OutOfMemory error inside the JVM.")
					self.waitForIdleIfRequired(config)
					// Stopping the mode as this will automatically do a restart.
					modes.GetConfiguredMode(config).Stop()
				}
			}
		}()
	})
}

func (self *OutOfMemoryErrorRestarter) waitForIdleIfRequired(config *util.Config) {
	if config.OutOfMemoryRestartOnlyWhenIDLE {
		for !util.NodeIsIdle.Get() {
			util.GOut("OOM", "Waiting for node to become IDLE before triggering a restart.")
			time.Sleep(time.Minute * 5)
		}
	}
}

// Returns true if a OOM error triggered a restart and resets the error state to false.
// Executing this method multiple times when OOM error was triggered returns true first and false
// with every subsequent call until the error is triggered again.
func (self *OutOfMemoryErrorRestarter) oomErrorTriggered() bool {
	if fi, err := os.Stat(self.outOfMemoryErrorMarker); err == nil && !fi.IsDir() {
		os.Remove(self.outOfMemoryErrorMarker)
		return true
	}
	return false
}

// Registering the restarter.
var _ = RegisterPreparer(NewOutOfMemoryErrorRestarter())
