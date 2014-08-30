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
	FullGCScript   = "3.times{ System.gc() }"
	FullGCPostBody = "script=%s"
)

// Defines an object which triggers a periodic restart of the Jenkins client when enabled.
type FullGCInvoker struct {
	tickers []*time.Ticker
}

func (self *FullGCInvoker) Name() string {
	return "Full GC Invoker"
}

func (self *FullGCInvoker) IsConfigAcceptable(config *util.Config) (bool) {
	if config.ForceFullGC && !config.HasCIConnection() {
		util.GOut("gc", "WARN: No Jenkins URI defined. System.GC() cannot be called inside the Jenkins client.");
		return false;
	}
	return true;
}

func (self *FullGCInvoker) Prepare(config *util.Config) {
	if self.tickers != nil {
		for _, ticker := range self.tickers {
			ticker.Stop()
		}
	} else {
		self.tickers = []*time.Ticker{}
	}

	if !config.ForceFullGC {
		return
	}

	util.GOut("gc", "Periodic forced full GC is enabled.")

	if (config.ForceFullGCIntervalMinutes > 0) {
		self.tickers = append(self.tickers, self.scheduleGCInvoker(config, config.ForceFullGCIntervalMinutes, false))
	}

	if (config.ForceFullGCIDLEIntervalMinutes > 0) {
		self.tickers = append(self.tickers, self.scheduleGCInvoker(config, config.ForceFullGCIDLEIntervalMinutes, true))
	}
}

func (self *FullGCInvoker) scheduleGCInvoker(config *util.Config, intervalMinutes int64, expectedIDLEState bool) (ticker *time.Ticker) {
	ticker = time.NewTicker(time.Minute*time.Duration(intervalMinutes))

	go func() {
		for _ = range ticker.C {
			if util.NodeIsIdle.Get() == expectedIDLEState {
				self.invokeSystemGC(config)
			}
		}
	}()

	return;
}

func (self *FullGCInvoker) invokeSystemGC(config *util.Config) {
	// curl -d "script=System.gc()" -X POST http://user:password@jenkins-host/ci/computer/%s/scriptText
	postBody := strings.NewReader(fmt.Sprintf(FullGCPostBody, url.QueryEscape(FullGCScript)))
	request, err := config.CIRequest("POST", fmt.Sprintf(FullGCURL, config.ClientName), postBody)
	if err == nil {
		if response, err := config.CIClient().Do(request); err == nil {
			response.Body.Close()
			if response.StatusCode != 200 {
				util.GOut("gc", "ERROR: Failed invoking full GC as node request in Jenkins failed with %s", response.Status)
			}
		} else {
			util.GOut("gc", "ERROR: Failed invoking full GC as Jenkins cannot be contacted. Cause: %v", err)
		}
	}
}

// Registering the restarter.
var _ = RegisterPreparer(new(FullGCInvoker))
