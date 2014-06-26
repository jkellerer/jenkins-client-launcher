// Copyright 2014 The jenkins-client-launcher Authors. All rights reserved.
// Use of this source code is governed by a MIT license that can be found in the LICENSE file.

package environment

import (
	"github.com/jkellerer/jenkins-client-launcher/launcher/util"
	"github.com/jkellerer/jenkins-client-launcher/launcher/modes"
	"time"
)

// Defines an object which triggers a periodic restart of the Jenkins client when enabled.
type PeriodicRestarter struct {
	util.AnyConfigAcceptor

	ticker *time.Ticker
}

func (self *PeriodicRestarter) Name() string {
	return "Periodic Client Restarter"
}

func (self *PeriodicRestarter) Prepare(config *util.Config) {
	if self.ticker != nil {
		self.ticker.Stop()
	}

	if !config.PeriodicClientRestartEnabled || config.PeriodicClientRestartIntervalHours <= 0 {
		return
	}

	self.ticker = time.NewTicker(time.Hour*time.Duration(config.PeriodicClientRestartIntervalHours))

	go func() {
		// Run in schedule
		for time := range self.ticker.C {
			util.GOut("periodic", "Triggering periodic restart.", time)
			self.waitForIdleIfRequired(config)
			// Stopping the mode as this will automatically do a restart.
			modes.GetConfiguredMode(config).Stop()
		}
	}()
}

func (self *PeriodicRestarter) waitForIdleIfRequired(config *util.Config) {
	if config.PeriodicClientRestartOnlyWhenIDLE {
		for !util.NodeIsIdle.Get() {
			util.GOut("periodic", "Waiting for node to become IDLE before triggering a restart.")
			time.Sleep(time.Minute * 5)
		}
	}
}

// Registering the restarter.
var _ = RegisterPreparer(new(PeriodicRestarter))
