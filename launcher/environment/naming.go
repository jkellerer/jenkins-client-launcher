// Copyright 2014 The jenkins-client-launcher Authors. All rights reserved.
// Use of this source code is governed by a MIT license that can be found in the LICENSE file.

package environment

import (
	"launcher/util"
	"encoding/xml"
	"os"
	"strings"
	"fmt"
)

const (
	ComputersURI = "computer/api/xml"
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

	clientName := config.ClientName
	if clientName == "" {
		if name, err := os.Hostname(); err == nil {
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

		if bestMatch != "" || match != "" {
			util.GOut("naming", "Found client name in Jenkins, using '%v'.", config.ClientName)
		} else {
			util.GOut("naming", "Client name '%v' was NOT FOUND in Jenkins. Likely the next operations will fail.", config.ClientName)
		}
	} else {
		util.GOut("nameing", "Failed to verify the client name in Jenkins. Cause: %v", err)
	}
}

// Registering the handler.
var _ = RegisterPreparer(new(NodeNameHandler))

