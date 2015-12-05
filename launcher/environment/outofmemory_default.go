// Copyright 2014 The jenkins-client-launcher Authors. All rights reserved.
// Use of this source code is governed by a MIT license that can be found in the LICENSE file.

// +build !windows

package environment

import (
	"fmt"
)

// Returns the OS specific value for the OnOutOfMemoryError command to execute.
func (self *OutOfMemoryErrorRestarter) createOOMErrorTriggerCommand() string {
	return fmt.Sprintf("/bin/echo 1 > \"%s\"", self.outOfMemoryErrorMarker)
}
