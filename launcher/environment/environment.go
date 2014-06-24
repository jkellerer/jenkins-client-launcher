// Copyright 2014 The jenkins-client-launcher Authors. All rights reserved.
// Use of this source code is governed by a MIT license that can be found in the LICENSE file.

package environment

import (
	"github.com/jkellerer/jenkins-client-launcher/launcher/util"
)

// Defines an interface for implementations that prepare or monitor the environment.
type EnvironmentPreparer interface {
util.ConfigVerifier
	// Returns the name of the mode.
	Name() (string)
	// Prepares the environment.
	Prepare(config *util.Config)
}

// Contains all registered preparers.
var AllEnvironmentPreparers = []EnvironmentPreparer{}

// Registers a new preparer and returns it.
func RegisterPreparer(preparer EnvironmentPreparer) EnvironmentPreparer {
	AllEnvironmentPreparers = append(AllEnvironmentPreparers, preparer)
	return preparer
}

// Passes all registered preparers to the specified callback.
func VisitAllPreparers(fn func(EnvironmentPreparer)) {
	for _, preparer := range AllEnvironmentPreparers {
		fn(preparer)
	}
}

// Runs all registered preparers.
func RunPreparers(config *util.Config) {
	count := len(AllEnvironmentPreparers); countdown := make(chan bool, count)

	VisitAllPreparers(func(p EnvironmentPreparer) {
		if !p.IsConfigAcceptable(config) {
			panic("Environment preparer " + p.Name() + " does not accept the current configuration. Please adjust the config.")
		}
	})

	VisitAllPreparers(func(p EnvironmentPreparer) {
		util.GOut("ENV", "Preparing %v", p.Name())
		go func() {
			defer func() {
				countdown <- true
			}()
			p.Prepare(config)
		}()
	})

	for ; count > 0; count-- {
		<-countdown
	}

	util.GOut("ENV", "Finished preparing the environment.")
}
