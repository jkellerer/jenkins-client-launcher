// Copyright 2014 The jenkins-client-launcher Authors. All rights reserved.
// Use of this source code is governed by a MIT license that can be found in the LICENSE file.

package environment

import (
	"github.com/jkellerer/jenkins-client-launcher/launcher/util"
	"encoding/xml"
	"os"
	"strings"
	"fmt"
	"net/url"
	"encoding/json"
	"runtime"
)

const (
	ComputersURI         = "computer/api/xml"
	CreateNodeURI        = "computer/doCreateItem"
	ExpectedNodeType     = "hudson.slaves.DumbSlave$DescriptorImpl"
	ExpectedNodeLauncher = "hudson.slaves.JNLPLauncher"
)

type AllComputerNames struct {
	XMLName xml.Name  `xml:"computerSet"`
	Names   []string `xml:"computer>displayName"`
}

// Returns the names of all nodes that are registered in Jenkins.
func GetAllRegisteredNodesInJenkins(config *util.Config) (*AllComputerNames, error) {
	response, err := config.CIGet(ComputersURI)
	if err == nil && response.StatusCode == 200 {
		defer response.Body.Close()
		names := new(AllComputerNames)
		err = xml.NewDecoder(response.Body).Decode(names)
		return names, err
	} else {
		if err == nil && response != nil {
			err = fmt.Errorf(response.Status)
		}
		return nil, err
	}
}

// Defines an object which tries to find the correct node name for the machine that runs the util.
type NodeNameHandler struct {
	util.AnyConfigAcceptor
}

func (self *NodeNameHandler) Name() string {
	return "Node Name Normalizer"
}

func (self *NodeNameHandler) Prepare(config *util.Config) {
	if !config.HasCIConnection() {
		return
	}

	if foundNode, err := self.verifyNodeName(config); err == nil {

		if !foundNode {
			if config.CreateClientIfMissing {
				if err := self.createNode(config); err == nil {
					util.GOut("naming", "Created node '%s' in Jenkins.", config.ClientName)
				} else {
					util.GOut("naming", "Tried to create node '%s' in Jenkins but failed. Cause: %v", config.ClientName, err)
				}
				foundNode, _ = self.verifyNodeName(config)
			} else {
				util.GOut("naming", "Will not attempt to auto generate node '%s' in Jenkins. Enable this with '-create' or within the configuration.", config.ClientName)
			}
		}

		if foundNode {
			util.GOut("naming", "Found client node name in Jenkins, using '%v'.", config.ClientName)
		} else {
			util.GOut("naming", "Client node name '%v' was NOT FOUND in Jenkins. Likely the next operations will fail.", config.ClientName)
		}
	} else {
		util.GOut("nameing", "Failed to verify the client node name in Jenkins. Cause: %v", err)
	}
}

func (self *NodeNameHandler) verifyNodeName(config *util.Config) (bool, error) {
	clientName := config.ClientName
	if clientName == "" {
		if name, err := util.Hostname(); err == nil {
			clientName = name
			config.ClientName = name
		}
	}

	if nodes, err := GetAllRegisteredNodesInJenkins(config); err == nil {
		clientName = strings.ToLower(clientName)
		match, bestMatch := "", ""

		for _, computerName := range nodes.Names {
			name := strings.ToLower(computerName)

			if name == clientName {
				bestMatch, match = computerName, computerName
			} else if len(clientName) > 0 && strings.Index(name, clientName+".") == 0 {
				match = computerName
			}
		}

		if bestMatch != "" {
			config.ClientName = bestMatch
		} else if match != "" {
			config.ClientName = match
		}

		return bestMatch != "" || match != "", nil
	} else {
		return false, err
	}
}

func (self *NodeNameHandler) createNode(config *util.Config) error {
	toJson := func(v interface{}) string {
		if content, err := json.Marshal(v); err == nil {
			return string(content)
		} else {
			return fmt.Sprintf("{\"error\": \"%v\"}", err)
		}
	}

	cwd, _ := os.Getwd()
	mode := "EXCLUSIVE"                                    // NORMAL or EXCLUSIVE (tied jobs only)
	retention := "hudson.slaves.RetentionStrategy$Always"  // Always on

	params := make(url.Values)
	params.Set("name", config.ClientName)
	params.Set("type", ExpectedNodeType)
	params.Set("json", toJson(map[string]interface{}{
		"name": config.ClientName,
		"nodeDescription": fmt.Sprintf("JSL auto generated node '%s'.", config.ClientName),
		"numExecutors": 1,
		"remoteFS": cwd,
		"labelString": fmt.Sprintf("JSL %s %s", runtime.GOOS, runtime.GOARCH),
		"mode": mode,
		"type": ExpectedNodeType,
		"retentionStrategy": map[string]interface{}{ "stapler-class": retention, },
		"nodeProperties": map[string]interface{}{    "stapler-class-bag": true, },
		"launcher": map[string]interface{}{          "stapler-class": ExpectedNodeLauncher, },
	}))

	if response, err := config.CIGet(CreateNodeURI + "?" + params.Encode()); err == nil {
		response.Body.Close()
		if response.StatusCode == 200 {
			return nil
		} else {
			return fmt.Errorf("Create node failed. Jenkins returned %v", response.Status)
		}
	} else {
		return err
	}
}

// Registering the handler.
var _ = RegisterPreparer(new(NodeNameHandler))

