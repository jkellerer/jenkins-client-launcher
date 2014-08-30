// Copyright 2014 The jenkins-client-launcher Authors. All rights reserved.
// Use of this source code is governed by a MIT license that can be found in the LICENSE file.

package util

import (
	"os"
	"fmt"
	"strings"
	"github.com/shiena/ansicolor"
	"sync"
)

var outputMutex = &sync.Mutex{}
var coloredStdOut = ansicolor.NewAnsiColorWriter(os.Stdout)
var defaultMessageColor = "\x1b[32m"
var launcherPrefix = fmt.Sprintf("%s>>%sJCL%s:%s", "\x1b[34m\x1b[1m", "\x1b[21m\x1b[34m", "\x1b[39m", "\x1b[39m")
var launcherGroupPrefix = fmt.Sprintf("%s>>%sJCL%s(%s%s%s):%s", "\x1b[34m\x1b[1m", "\x1b[21m\x1b[34m", "\x1b[39m", "\x1b[35m\x1b[1m", "%s", "\x1b[21m\x1b[39m", "\x1b[39m")

// Prints a message to the app's console output with optional Printf styled substitutions.
func Out(message string, a ...interface{}) {
	outputMutex.Lock()
	defer outputMutex.Unlock()
	fmt.Fprintln(coloredStdOut, launcherPrefix, formatOut(message, defaultMessageColor, a != nil, a...))
}

// Prints a message to the app's console output with optional Printf styled substitutions.
// "group" is used to group messages of the same kind.
func GOut(group string, message string, a ...interface{}) {
	outputMutex.Lock()
	defer outputMutex.Unlock()
	fmt.Fprintln(coloredStdOut, fmt.Sprintf(launcherGroupPrefix, strings.ToLower(group)), formatOut(message, defaultMessageColor, a != nil, a...))
}

// Prints a message to the app's console output with optional Printf styled substitutions and without a JCL prefix.
func FlatOut(message string, a ...interface{}) {
	outputMutex.Lock()
	defer outputMutex.Unlock()
	fmt.Fprintln(coloredStdOut, "", formatOut(message, "\x1b[0m", a != nil, a...))
}

func formatOut(message string, defaultColor string, applyArgs bool, args ...interface{}) string {
	msg := message;

	if applyArgs {
		formattedArgs := make([]interface{}, len(args))
		for index, value := range args {
			formattedArgs[index] = fmt.Sprintf("\x1b[1m%v\x1b[21m", value);
		}
		msg = fmt.Sprintf(message, formattedArgs...)
	}

	if strings.Contains(msg, "ERROR") || strings.Contains(msg, "PANIC") {
		msg = fmt.Sprintf("\x1b[31m%s\x1b[0m", msg)
	} else if strings.Contains(msg, "WARN") {
		msg = fmt.Sprintf("\x1b[33m%s\x1b[0m", msg)
	} else {
		msg = fmt.Sprintf("%s%s\x1b[0m", defaultColor, msg)
	}

	return msg
}
