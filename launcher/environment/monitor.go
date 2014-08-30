// Copyright 2014 The jenkins-client-launcher Authors. All rights reserved.
// Use of this source code is governed by a MIT license that can be found in the LICENSE file.

package environment

import (
	"github.com/jkellerer/jenkins-client-launcher/launcher/util"
	"github.com/jkellerer/jenkins-client-launcher/launcher/modes"
	"fmt"
	"encoding/xml"
	"time"
)

const (
	NodeMonitoringURI = "computer/%s/api/xml"
)

// The interval when the jenkins node is monitored.
var nodeMonitoringInterval = time.Second * 15

// The max number of offline results in a row until a reconnect is forced.
var maxOfflineCountBeforeRestart = int16(4)

type JenkinsNodeStatus struct {
	DisplayName          string `xml:"displayName"`
	Idle                 bool `xml:"idle"`
	Offline              bool `xml:"offline"`
	TemporarilyOffline   bool `xml:"temporarilyOffline"`
}

// Returns the current offline and idle status of this Jenkins node from the Jenkins server.
func GetJenkinsNodeStatus(config *util.Config) (*JenkinsNodeStatus, error) {
	if response, err := config.CIGet(fmt.Sprintf(NodeMonitoringURI, config.ClientName)); err == nil {
		defer response.Body.Close()
		if response.StatusCode == 200 {
			status := &JenkinsNodeStatus{}
			err = xml.NewDecoder(response.Body).Decode(status)
			return status, err
		} else {
			return nil, fmt.Errorf(response.Status)
		}
	} else {
		return nil, err
	}
}

// Implements a monitor that issues a rest call on jenkins to see whether the node is online within jenkins.
type JenkinsNodeMonitor struct {
	ticker *time.Ticker
	onlineShown  bool
	offlineCount int16
}

func (self *JenkinsNodeMonitor) IsConfigAcceptable(config *util.Config) (bool) {
	if config.ClientMonitorStateOnServer && !config.HasCIConnection() {
		util.GOut("monitor", "No Jenkins URI defined. Cannot monitor this node within Jenkins.");
		return false;
	}
	return true;
}

func (self *JenkinsNodeMonitor) Name() string {
	return "Jenkins Node Monitor"
}

func (self *JenkinsNodeMonitor) Prepare(config *util.Config) {
	if self.ticker != nil {
		self.ticker.Stop()
	}

	if config.ClientMonitorStateOnServer {
		maxOfflineCountBeforeRestart = config.ClientMonitorStateOnServerMaxFailures

		self.ticker = time.NewTicker(nodeMonitoringInterval)

		go func() {
			// Run in schedule
			for _ = range self.ticker.C {
				self.monitor(config)
			}
		}()
	} else {
		// Setting IDLE to always true if active monitoring is disabled.
		util.NodeIsIdle.Set(true)
	}
}

// Checks if both, this side and the remote side show the node as connected and increments a offline count if not.
// Forces a restart of the connector when offline count reaches the threshold.
func (self *JenkinsNodeMonitor) monitor(config *util.Config) {
	if self.isThisSideConnected(config) {
		if connected, idle, serverReachable := self.isServerSideConnected(config); connected {
			util.NodeIsIdle.Set(idle)
			self.offlineCount = 0

			if !self.onlineShown {
				util.GOut("monitor", "Node is online in Jenkins.")
				self.onlineShown = true
			}
		} else {
			util.NodeIsIdle.Set(true)

			if serverReachable {
				self.offlineCount++
			}

			if self.offlineCount > 3 * maxOfflineCountBeforeRestart {
				self.offlineCount = maxOfflineCountBeforeRestart
			}

			if self.offlineCount == maxOfflineCountBeforeRestart {
				self.forceReconnect(config)
			}

			util.GOut("monitor", "WARN: Node is OFFLINE in Jenkins.")
			self.onlineShown = false
		}
	} else {
		util.NodeIsIdle.Set(true)
		self.offlineCount = 0

		if self.onlineShown {
			util.GOut("monitor", "WARN: Node went OFFLINE locally.")
			self.onlineShown = false
		}
	}
}

// Checks if the run mode is in started state.
// Also updates the global "util.NodeIsIdle" state to true if run mode is not in started state.
func (self *JenkinsNodeMonitor) isThisSideConnected(config *util.Config) bool {
	return modes.GetConfiguredMode(config).Status().Get() == modes.ModeStarted
}

// Checks if Jenkins shows this node as connected and returns the node's IDLE state as second return value.
func (self *JenkinsNodeMonitor) isServerSideConnected(config *util.Config) (connected bool, idle bool, serverReachable bool) {
	if status, err := GetJenkinsNodeStatus(config); err == nil {
		return !status.Offline, status.Idle, true
	} else {
		util.GOut("monitor", "ERROR: Failed to monitor node %v using %v. Cause: %v", config.ClientName, config.CIHostURI, err)
		return false, true, false
	}
}

// Forces a reconnect with Jenkins by stopping the current mode.
func (self *JenkinsNodeMonitor) forceReconnect(config *util.Config) {
	if self.isThisSideConnected(config) {
		util.GOut("monitor", "WARN: This node appears dead in Jenkins, forcing a reconnect.")
		modes.GetConfiguredMode(config).Stop()
	}
}

// Registering the monitor.
var _ = RegisterPreparer(new(JenkinsNodeMonitor))
