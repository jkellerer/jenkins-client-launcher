// Copyright 2014 The jenkins-client-launcher Authors. All rights reserved.
// Use of this source code is governed by a MIT license that can be found in the LICENSE file.

package modes

import (
	"launcher/util"
	"os"
	"os/signal"
	"time"
)

const (
	ModeNone = iota
	ModeStarting
	ModeStarted
	ModeStopping
	ModeStopped
)

// Defines a callback that is notified when a mode is before start, started or stopped.
type ExecutableModeListener func(mode ExecutableMode, nextStatus int32, config *util.Config)

// Defines an interface for implementations of a run mode of this util.
type ExecutableMode interface {
util.ConfigVerifier

	// Returns the name of the mode.
	Name() (string)
	// Returns the status of the mode.
	Status() (int32)
	// Start the mode and returns after the mode has been started.
	Start(config *util.Config) (error)
	// Stops the mode.
	Stop()
}

var AllModes = []ExecutableMode{}

// Registers a run mode implementation.
func RegisterMode(mode ExecutableMode) ExecutableMode {
	AllModes = append(AllModes, mode)
	return mode
}

var allModeListeners = []ExecutableModeListener{}

// Registers a mode listener.
func RegisterModeListener(listener ExecutableModeListener) ExecutableModeListener {
	allModeListeners = append(allModeListeners, listener)
	return listener
}

func callListeners(mode ExecutableMode, nextStatus int32, config *util.Config) {
	for _, listener := range allModeListeners {
		listener(mode, nextStatus, config)
	}
}

// Returns the mode that is activated within the specified config instance.
func GetConfiguredMode(config *util.Config) ExecutableMode {
	for _, mode := range AllModes {
		if config.RunMode == mode.Name() {
			return mode
		}
	}
	panic("The configured mode '" + config.RunMode + "' is not implemented.")
}

// Runs the mode that is activated within the specified config instance,
// returning false if the mode stopped due to a kill or interrupt and true when the mode stopped due to an error.
func RunConfiguredMode(config *util.Config) bool {
	exitRequested := make(chan bool, 2)

	// Getting the configured run mode
	executableMode := GetConfiguredMode(config)
	util.Out("Starting mode %v", executableMode.Name())
	callListeners(executableMode, ModeStarting, config)
	err := executableMode.Start(config)

	if err != nil {
		util.Out("Failed to start mode '%v'; Cause: %v", executableMode.Name(), err)
		return false
	} else {
		callListeners(executableMode, ModeStarted, config)
		defer callListeners(executableMode, ModeStopped, config)
	}

	// Waiting on Interrupt or Kill and stop the run mode when received.
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, os.Kill)

	go func() {
		sig := <-signals // This channel receives only Interrupt or Kill and blocks until one is received.
		util.Out("Received signal: %v, stopping...", sig)
		exitRequested <- true
		if executableMode.Status() != ModeStopped {
			executableMode.Stop()
		}
	}()

	util.Out("STARTED mode %v", executableMode.Name())

	// Waiting for the run mode to stop
	for executableMode.Status() != ModeStopped {
		time.Sleep(100)
	}
	exitRequested <- false
	signals <- os.Interrupt

	util.Out("STOPPED mode %v", executableMode.Name())

	// Returning true if we want to re-run.
	isExistRequested := <-exitRequested
	<-exitRequested

	return !isExistRequested
}
