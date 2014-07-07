// Copyright 2014 The jenkins-client-launcher Authors. All rights reserved.
// Use of this source code is governed by a MIT license that can be found in the LICENSE file.

package util

import (
	"encoding/xml"
	"os"
	"net/http"
	"crypto/tls"
	"fmt"
	"regexp"
	"strings"
	"io"
	"io/ioutil"
)

// Implemented by types that verify if the configuration is valid for them.
type ConfigVerifier interface {
	// Returns true if the configuration is acceptable, false otherwise.
	IsConfigAcceptable(config *Config) (bool)
}

// Helper to use by types that accept any config but require an implementation of ConfigVerifier.
type AnyConfigAcceptor struct {
}

func (_ *AnyConfigAcceptor) IsConfigAcceptable(config *Config) (bool) {
	return true
}

// Allows to configure console monitoring.
type ConsoleMonitor struct {
	RestartTriggerTokens []string `xml:"console>errorTokens>token"`
}

// Returns true if the given line contains a restart trigger token.
func (self *ConsoleMonitor) IsRestartTriggered(line string) bool {
	for _, token := range self.RestartTriggerTokens {
		if strings.Contains(line, token) {
			return true
		}
	}
	return false
}

const (
	JenkinsConnectionDescription = `
<ci>
  Specifies the connection between this node and Jenkins CI server:

  - url:                Is the base address of jenkins, e.g. "http://jenkins-host/jenkins".

  - noCertificateCheck: Toggles whether certificates are verified.
                        Enabling this option makes HTTPS connections as secure as HTTP connections.
                        (Use with caution!)
</ci>
`)

type JenkinsConnection struct {
	CIHostURI        string `xml:"ci>url"`
	CIAcceptAnyCert  bool   `xml:"ci>noCertificateCheck"`
	CIUsername       string `xml:"ci>auth>user"`
	CIPassword       string `xml:"ci>auth>password"`
	ciCrumbHeader    string `xml:"-"`
	ciCrumbValue     string `xml:"-"`
}

// Returns true if the configuration has a Jenkins url.
func (self *JenkinsConnection) HasCIConnection() bool {
	isHttpUrl, _ := regexp.MatchString("^(?i)http(s|)://.+", self.CIHostURI);
	return isHttpUrl
}

// Returns a HTTP client that is configured to connect with Jenkins.
func (self *JenkinsConnection) CIClient() *http.Client {
	client := http.DefaultClient

	if self.CIAcceptAnyCert {
		tr := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		client = &http.Client{Transport: tr}
	}

	return client
}

// Returns a request object which may be used with CIClient to do a HTTP request.
func (self *JenkinsConnection) CIRequest(method, path string, body io.Reader) (request *http.Request, err error) {
	if request, err = http.NewRequest(method, fmt.Sprintf("%v/%v", self.CIHostURI, path), body); err != nil {
		return
	}

	// Add support for basic auth
	if len(self.CIUsername) > 0 && len(self.CIPassword) > 0 {
		request.SetBasicAuth(self.CIUsername, self.CIPassword)
	}

	// Add support for cross site forgery protected Jenkins instances.
	if !strings.EqualFold(method, "GET") {
		if self.ciCrumbHeader == "" {
			self.ciCrumbHeader = "-"
			if response, err := self.CIGet("/crumbIssuer/api/xml?xpath=concat(//crumbRequestField,%22:%22,//crumb)"); err == nil {
				defer response.Body.Close()
				if response.StatusCode == 200 {
					if content, err := ioutil.ReadAll(response.Body); err == nil {
						if v := strings.SplitN(string(content), ":", 2); len(v) == 2 {
							self.ciCrumbHeader, self.ciCrumbValue = v[0], v[1]
							GOut("Security", "%s: %s", self.ciCrumbHeader, self.ciCrumbValue)
						}
					}
				}
			}
		}

		if self.ciCrumbHeader != "-" {
			request.Header.Set(self.ciCrumbHeader, self.ciCrumbValue)
		}
	}

	return
}

// Issues a HTTP-GET request on Jenkins using the specified request path (= path + query string).
func (self *JenkinsConnection) CIGet(path string) (*http.Response, error) {
	if request, err := self.CIRequest("GET", path, nil); err != nil {
		return nil, err
	} else {
		return self.CIClient().Do(request)
	}
}

const (
	SSHServerDescription = `
<sshServer>
  Configures the server port of the SSH server (only for run mode 'ssh-server')
</sshServer>
`)

type SSHServer struct {
	SSHListenAddress     string `xml:"sshServer>address"`
	SSHListenPort        uint16 `xml:"sshServer>port"`
	SSHUsername          string `xml:"sshServer>auth>user"`
	SSHPassword          string `xml:"sshServer>auth>password"`
}

const (
	JavaOptionsDescription = `
<java>
  Configures the Java environment that is used to bootstrap the Jenkins Client:

  - args>arg:      Enumerates additional options (each wrapped in one <arg>OPT</arg>) that are used
                   to start java.
                   The default options try to optimise GC for low footprint instead of performance in
                   order to leave more memory for IO and forked build tasks.

  - forceFullGC:   Allows to enable periodic calls to "System.gc()" to reduce the overall memory
                   usage of the Jenkins Client.
</java>
`)

type JavaOptions struct {
	JavaArgs                   []string `xml:"java>args>arg"`
	ForceFullGC                bool     `xml:"java>forceFullGC>enabled"`
	ForceFullGCOnlyWhenIDLE    bool     `xml:"java>forceFullGC>onlyWhenIdle"`
	ForceFullGCIntervalMinutes int64    `xml:"java>forceFullGC>interval>minutes"`
}

const (
	ClientOptionsDescription = `
<client>
  Configures the Jenkins client and runtime behaviour:

  - name:          Is the name of this node. (Defaults to [hostname])

  - secretKey:     The client specific secret key used to communicate with Jenkins.
                   When empty, JCL will fetch it from Jenkins.

  - passAuth:      Toggles whether CI auth credentials are passed to the Jenkins client.

  - monitoring:    Toggles whether JCL monitors the Jenkins client and restarts it on failure:
                   - stateOnServer: When enabled JCL watches the node state on Jenkins and
                                    triggers a restart when the node appears offline.
                   - console:       When enabled JCL watches the console output and triggers
                                    a restart when one of the configured error tokens are found.

  - restart:       Controls how restarts of the Jenkins client are triggered.
                   - handleReconnects:  When enabled let JCL handle reconnects on server outage
                                        instead of allowing the Jenkins client to handle it by
                                        itself.
                   - sleepOnFailure:    Number of seconds to sleep between 2 attempts to restart.
                                        This sleep time is ramped up (multiplied) with the number
                                        of restart attempts in a row.
                   - periodic:          Allows to trigger a restart per interval
                                        (e.g. once a week).
</client>
`)

type ClientOptions struct {
	ClientName                            string `xml:"client>name"`
	SecretKey                             string `xml:"client>secretKey"`
	PassCIAuth                            bool   `xml:"client>passAuth"`
	ClientMonitorStateOnServer            bool   `xml:"client>monitoring>stateOnServer>enabled"`
	ClientMonitorStateOnServerMaxFailures int16  `xml:"client>monitoring>stateOnServer>maxFailures"`
	ClientMonitorConsole                  bool   `xml:"client>monitoring>console>enabled"`
	HandleReconnectsInLauncher            bool   `xml:"client>restart>handleReconnects"`
	SleepTimeSecondsBetweenFailures       int64  `xml:"client>restart>sleepOnFailure>seconds"`
	PeriodicClientRestartEnabled          bool   `xml:"client>restart>periodic>enabled"`
	PeriodicClientRestartOnlyWhenIDLE     bool   `xml:"client>restart>periodic>onlyWhenIdle"`
	PeriodicClientRestartIntervalHours    int64  `xml:"client>restart>periodic>interval>hours"`
}

const (
	MaintenanceDescription = `
<maintenance>
  Configures additional maintenance tasks that JCL can perform to keep the node online:

  - cleanTempLocation:  Toggles whether JCL will clean the temporary folder from files that
                        haven't been modified for "ttl>hours".
                        Files can be excluded from cleaning using GLOB style patterns, following
                        the syntax defined for "http://golang.org/pkg/path/filepath/#Match"
                        Example:
                        <exclusions>
                          <exclusion>*.dll</exclusion>
                          <exclusion>*\mypath\*</exclusion>
                        </exclusions>
</maintenance>
`)

type Maintenance struct {
	CleanTempLocation                bool     `xml:"maintenance>cleanTempLocation>enabled"`
	CleanTempLocationOnlyWhenIDLE    bool     `xml:"maintenance>cleanTempLocation>onlyWhenIdle"`
	CleanTempLocationIntervalHours   int64    `xml:"maintenance>cleanTempLocation>interval>hours"`
	CleanTempLocationTTLHours        int64    `xml:"maintenance>cleanTempLocation>ttl>hours"`
	CleanTempLocationExclusions      []string `xml:"maintenance>cleanTempLocation>exclusions>exclusion"`
}

const (
	ConfigDescription = `

Configuration file for Jenkins Client Launcher (JCL)
`)

type Config struct {
	XMLName           xml.Name `xml:"config"`
	RunMode           string   `xml:"runMode,attr"`
	Autostart         bool     `xml:"autostart,attr"`
	ConfigDescription string   `xml:",comment"`

	NeedsSave         bool      `xml:"-"`

	JenkinsConnection
	ClientOptions
	JavaOptions
	SSHServer
	ConsoleMonitor
	Maintenance
}

// Returns a new instance of config with default values.
func NewDefaultConfig() *Config {
	hostname, err := os.Hostname()
	if err != nil { hostname = "" }

	config := &Config{
		NeedsSave: true,
		Autostart: false,
		RunMode: "client",
		ConfigDescription: ConfigDescription +
				JenkinsConnectionDescription +
				ClientOptionsDescription +
				JavaOptionsDescription +
				SSHServerDescription +
				MaintenanceDescription,
		JenkinsConnection: JenkinsConnection{
			CIHostURI: "",
			CIUsername: "admin", CIPassword: "changeit", CIAcceptAnyCert: false,
		},
		ClientOptions: ClientOptions{
			ClientName: hostname,
			ClientMonitorStateOnServer: true,
			ClientMonitorStateOnServerMaxFailures: 2,
			ClientMonitorConsole: true,
			SecretKey: "",
			PassCIAuth: false,
			HandleReconnectsInLauncher: false,
			SleepTimeSecondsBetweenFailures: 30,
			PeriodicClientRestartEnabled: false,
			PeriodicClientRestartOnlyWhenIDLE: true,
			PeriodicClientRestartIntervalHours: 48,
		},
		JavaOptions: JavaOptions{
			// Configuring java to spend more time in garbage collection instead of using more memory.
			// We want the memory for IO cache and other build processes and not to be wasted in unused heap.
			JavaArgs: []string {
				"-Xms10m",
				"-XX:GCTimeRatio=8",
				"-XX:+ClassUnloading",
				"-XX:+UseParallelOldGC",
				"-XX:+UseParallelOldGCCompacting",
				"-XX:+UseMaximumCompactionOnSystemGC",
			},
			ForceFullGC: true,
			ForceFullGCOnlyWhenIDLE: true,
			ForceFullGCIntervalMinutes: 5,
		},
		SSHServer: SSHServer{
			SSHListenAddress: "0.0.0.0",
			SSHListenPort: 2022,
			SSHUsername: "ssh",
			SSHPassword: "changeit",
		},
		ConsoleMonitor: ConsoleMonitor{
			RestartTriggerTokens: []string{
				"java.lang.OutOfMemoryError",
				"I/O error in channel channel",
				"The server rejected the connection",
				"java.net.SocketTimeoutException",
			},
		},
		Maintenance: Maintenance{
			CleanTempLocation: true,
			CleanTempLocationOnlyWhenIDLE: true,
			CleanTempLocationIntervalHours: 4,
			CleanTempLocationTTLHours: 48,
		},
	}

	return config;
}

// Returns a new instance that is initialized from the specified file.
// If the file cannot be loaded the returned config will be similar to what NewDefaultConfig() returns.
func NewConfig(fileName string) *Config {
	config := NewDefaultConfig();

	file, err := os.Open(fileName)
	defer file.Close()

	if err == nil {
		fileInfo, err := file.Stat()

		if err == nil && !fileInfo.IsDir() {
			Out("Loading configuration from %v", fileName)

			lists := []*[]string {&config.CleanTempLocationExclusions, &config.RestartTriggerTokens, &config.JavaArgs}
			captures := config.captureLists(lists...)

			if err := xml.NewDecoder(file).Decode(config); err == nil {
				config.NeedsSave = false;
			}

			config.restoreListsIfEmpty(captures, lists...)
		}
	}

	if err != nil {
		Out("Using default configuration. Loading '%v' failed: %v", fileName, err)
	}

	return config;
}

// Captures the specified array lists and returns them as a single 2d array.
// After capturing the values the source lists are reset to new empty arrays.
func (self *Config) captureLists(lists ...*[]string) [][]string {
	captures := make([][]string, len(lists))
	for index, list := range lists {
		captures[index] = *list
		*list = []string{}
	}

	return captures
}

// Restores the specified lists from a previously captured state if the lists are still empty.
func (self *Config) restoreListsIfEmpty(captures [][]string, lists ...*[]string) {
	for index, list := range lists {
		if len(*list) == 0 {
			*list = captures[index]
		}
	}
}

// Converts the config to a XML string.
func (self *Config) String() string {
	value, _ := xml.MarshalIndent(self, "", "    ")
	return string(value)
}

// Saves the config to the specified file.
func (self *Config) Save(fileName string) {
	Out("Saving new configuration to '%v'", fileName)

	file, err := os.Create(fileName)
	defer file.Close()

	if err == nil {
		enc := xml.NewEncoder(file)
		enc.Indent("", "    ")

		if err := enc.Encode(self); err != nil {
			Out("Failed writing configuration to %v \nError: %v", file, err)
		}
	}
}
