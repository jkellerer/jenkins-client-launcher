// Copyright 2014 The jenkins-client-launcher Authors. All rights reserved.
// Use of this source code is governed by a MIT license that can be found in the LICENSE file.

package environment

import (
	"github.com/jkellerer/jenkins-client-launcher/launcher/util"
	"time"
	"strings"
	"net/url"
	"fmt"
)

const (
	FullGCURL      = "/computer/%s/scriptText"
	FullGCScript   = "System.gc()"
	FullGCPostBody = "script=%s"
)

// Defines an object which triggers a periodic restart of the Jenkins client when enabled.
type FullGCInvoker struct {
	ticker *time.Ticker
}

func (self *FullGCInvoker) Name() string {
	return "Full GC Invoker"
}

func (self *FullGCInvoker) IsConfigAcceptable(config *util.Config) (bool) {
	if config.ForceFullGC && !config.HasCIConnection() {
		util.GOut("gc", "No Jenkins URI defined. System.GC() cannot be called inside the Jenkins client.");
		return false;
	}
	return true;
}

func (self *FullGCInvoker) Prepare(config *util.Config) {
	if self.ticker != nil {
		self.ticker.Stop()
	}

	if !config.ForceFullGC || config.ForceFullGCIntervalMinutes <= 0 {
		return
	}

	self.ticker = time.NewTicker(time.Minute*time.Duration(config.ForceFullGCIntervalMinutes))

	if config.ForceFullGCOnlyWhenIDLE {
		util.GOut("gc", "Periodic forced full GC is enabled when the node is IDLE.")
	} else {
		util.GOut("gc", "Periodic forced full GC is enabled.")
	}

	go func() {
		// Run in schedule
		for _ = range self.ticker.C {
			self.waitForIdleIfRequired(config)
			self.invokeSystemGC(config)
		}
	}()
}

func (self *FullGCInvoker) waitForIdleIfRequired(config *util.Config) {
	if config.ForceFullGCOnlyWhenIDLE {
		for !util.NodeIsIdle.Get() {
			time.Sleep(time.Second * 30)
		}
	}
}

func (self *FullGCInvoker) invokeSystemGC(config *util.Config) {
	// curl -d "script=System.gc()" -X POST http://user:password@jenkins-host/ci/computer/%s/scriptText
	postBody := strings.NewReader(fmt.Sprintf(FullGCPostBody, url.QueryEscape(FullGCScript)))
	request, err := config.CIRequest("POST", fmt.Sprintf(FullGCURL, config.ClientName), postBody)
	if (err == nil) {
		response, err := config.CIClient().Do(request)
		if err != nil {
			util.GOut("gc", "Failed invoking full GC as Jenkins cannot be contacted. Cause: %v", err)
		} else if response.StatusCode != 200 {
			util.GOut("gc", "Failed invoking full GC as node request in Jenkins failed with %s", response.Status)
		}
	}
}

// Registering the restarter.
var _ = RegisterPreparer(new(FullGCInvoker))
