package util

import "os"

// Alias to "os.Hostname()".
func Hostname() (name string, err error) {
	return os.Hostname()
}
