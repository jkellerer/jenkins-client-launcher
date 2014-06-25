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

type JenkinsConnection struct {
	CIHostURIDescription   string `xml:",comment"`
	CIHostURI              string `xml:"ci>url"`
	CIAcceptAnyCert        bool   `xml:"ci>noCertificateCheck"`
	CIUsername             string `xml:"ci>auth>user"`
	CIPassword             string `xml:"ci>auth>password"`
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

// Issues a HTTP-GET request on Jenkins using the specified request path (= path + query string).
func (self *JenkinsConnection) CIGet(path string) (*http.Response, error) {
	request, err := http.NewRequest("GET", fmt.Sprintf("%v/%v", self.CIHostURI, path), nil)
	if err != nil {
		return nil, err
	}

	if len(self.CIUsername) > 0 && len(self.CIPassword) > 0 {
		request.SetBasicAuth(self.CIUsername, self.CIPassword)
	}

	return self.CIClient().Do(request)
}

type SSHServer struct {
	SSHListenAddress string `xml:"sshServer>address"`
	SSHListenPort    uint16 `xml:"sshServer>port"`
	SSHUsername      string `xml:"sshServer>auth>user"`
	SSHPassword      string `xml:"sshServer>auth>password"`
}

type JavaOptions struct {
	JavaArgs  []string `xml:"java>args>arg"`
}

type ClientOptions struct {
	ClientName                            string `xml:"client>name"`
	SecretKey                             string `xml:"client>secretKey"`
	PassCIAuth                            bool   `xml:"client>passAuth"`
	ClientMonitorStateOnServer            bool   `xml:"client>monitoring>stateOnServer>enabled"`
	ClientMonitorStateOnServerMaxFailures int16  `xml:"client>monitoring>stateOnServer>maxFailures"`
	ClientMonitorConsole                  bool   `xml:"client>monitoring>console>enabled"`
	HandleReconnectsInLauncher            bool   `xml:"client>restart>handleReconnects"`
	SleepTimeSecondsBetweenFailures       int64  `xml:"client>restart>sleepOnFailure>seconds"`
}

type Maintenance struct {
	CleanTempLocation                bool     `xml:"maintenance>cleanTempLocation>enabled"`
	CleanTempLocationIntervalHours   int64    `xml:"maintenance>cleanTempLocation>interval>hours"`
	CleanTempLocationTTLHours        int64    `xml:"maintenance>cleanTempLocation>ttl>hours"`
	CleanTempLocationExclusions      []string `xml:"maintenance>cleanTempLocation>exclusions>exclusion"`
}

type Config struct {
	NeedsSave    bool      `xml:"-"`

	XMLName         xml.Name `xml:"config"`
	RunMode         string   `xml:"runMode,attr"`
	Autostart       bool     `xml:"autostart,attr"`
	Comment         string   `xml:",comment"`

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
		Comment: "",
		JenkinsConnection: JenkinsConnection{
			CIHostURIDescription: "",
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
		},
		JavaOptions: JavaOptions{
			// Configuring java to spend more time in garbage collection instead of using more memory.
			// We want the memory for IO cache and other build processes and not to be wasted in unused heap.
			// TODO: Add support for "jcmd <pid> GC.run" to call GC explicitly on schedule or when the node is known to be IDLE.
			JavaArgs: []string {
				"-Xms10m",
				"-XX:+UseSerialGC",
				"-XX:GCTimeRatio=5",
				"-XX:MaxGCPauseMillis=5000",
				"-XX:MaxGCMinorPauseMillis=333",
				"-XX:MaxHeapFreeRatio=25",
				"-XX:MinHeapFreeRatio=10",
				"-XX:+CMSClassUnloadingEnabled",
			},
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
			CleanTempLocationIntervalHours: 4,
			CleanTempLocationTTLHours: 24,
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
