// Copyright 2014 The jenkins-client-launcher Authors. All rights reserved.
// Use of this source code is governed by a MIT license that can be found in the LICENSE file.

// +build !windows

package util

import "os"

// Alias to "os.Hostname()".
func Hostname() (name string, err error) {
	return os.Hostname()
}
