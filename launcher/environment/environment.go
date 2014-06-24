// Copyright 2014 The jenkins-client-launcher Authors. All rights reserved.
// Use of this source code is governed by a MIT license that can be found in the LICENSE file.

package environment

import (
	"github.com/jkellerer/jenkins-client-launcher/launcher/util"
	"sync"
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
	work := new(sync.WaitGroup)
	work.Add(len(AllEnvironmentPreparers))

	enabledPreparers := map[string]bool{}

	VisitAllPreparers(func(p EnvironmentPreparer) {
		if p.IsConfigAcceptable(config) {
			enabledPreparers[p.Name()] = true
		} else {
			util.GOut("ENV", "WARN: Environment preparer " + p.Name() + " does not accept the current config and it won't operate.")
			util.GOut("ENV", "Adjust the configuration to prevent the warning above.")
		}
	})

	VisitAllPreparers(func(p EnvironmentPreparer) {
		if enabledPreparers[p.Name()] {
			util.GOut("ENV", "Preparing %v", p.Name())
			go func() {
				defer func() {
					work.Done()
				}()
				p.Prepare(config)
			}()
		} else {
			util.GOut("ENV", "Skipping %v", p.Name())
			work.Done()
		}
	})

	work.Wait()

	util.GOut("ENV", "Finished preparing the environment.")
}
