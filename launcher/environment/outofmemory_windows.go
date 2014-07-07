package environment

import (
	"os"
	"fmt"
)

const (
	outOfMemoryErrorMarker = ".oom-restart"
)

// Returns the OS specific value for the OnOutOfMemoryError command to execute.
func (self *OutOfMemoryErrorRestarter) createOOMErrorTriggerCommand() string {
	cwd, _ := os.Getwd()
	cmd := os.Getenv("ComSpec")
	if cmd != "" {
		return fmt.Sprintf("\"%s\" /c echo 1 > \"%s\\%s\"", cmd, cwd, outOfMemoryErrorMarker)
	} else {
		return ""
	}
}

// Returns true if a OOM error triggered a restart and resets the error state to false.
// Executing this method multiple times when OOM error was triggered returns true first and false
// with every subsequent call until the error is triggered again.
func (self *OutOfMemoryErrorRestarter) oomErrorTriggered() bool {
	if fi, err := os.Stat(outOfMemoryErrorMarker); err == nil && !fi.IsDir() {
		os.Remove(outOfMemoryErrorMarker)
		return true
	}
	return false
}
