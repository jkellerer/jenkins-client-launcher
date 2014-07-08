// Copyright 2014 The jenkins-client-launcher Authors. All rights reserved.
// Use of this source code is governed by a MIT license that can be found in the LICENSE file.

package environment

import (
	"os"
	"fmt"
)

// Returns the OS specific value for the OnOutOfMemoryError command to execute.
func (self *OutOfMemoryErrorRestarter) createOOMErrorTriggerCommand() string {
	cmd := os.Getenv("ComSpec")
	if cmd != "" {
		return fmt.Sprintf("\"%s\" /c echo 1 > \"%s\"", cmd, self.outOfMemoryErrorMarker)
	} else {
		return ""
	}
}
