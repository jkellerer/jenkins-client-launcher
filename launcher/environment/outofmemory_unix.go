package environment

import (
	"fmt"
)

// Returns the OS specific value for the OnOutOfMemoryError command to execute.
func (self *OutOfMemoryErrorRestarter) createOOMErrorTriggerCommand() string {
	return fmt.Sprintf("/bin/echo 1 > \"%s\"", self.outOfMemoryErrorMarker)
}
