// Copyright 2014 The jenkins-client-launcher Authors. All rights reserved.
// Use of this source code is governed by a MIT license that can be found in the LICENSE file.

package util

import (
	"fmt"
	"strings"
)

// Prints a message to the app's console output with optional Printf styled substitutions.
func Out(message string, a ...interface{}) {
	fmt.Println(">> LAUNCHER:", formatOut(message, a != nil, a...))
}

// Prints a message to the app's console output with optional Printf styled substitutions.
// "group" is used to group messages of the same kind.
func GOut(group string, message string, a ...interface{}) {
	fmt.Println(fmt.Sprintf(">> LAUNCHER(%s):", strings.ToLower(group)), formatOut(message, a != nil, a...))
}

func formatOut(message string, applyArgs bool, args ...interface{}) string {
	msg := message;
	if applyArgs {
		msg = fmt.Sprintf(message, args...)
	}
	return msg
}
