// Copyright 2014 The jenkins-client-launcher Authors. All rights reserved.
// Use of this source code is governed by a MIT license that can be found in the LICENSE file.

package util

import (
	"time"
	"os"
	"syscall"
)

func GetFileLastTouched(info os.FileInfo) (lastTouched time.Time) {
	lastTouched = info.ModTime()

	if value, ok := info.Sys().(*syscall.Win32FileAttributeData); ok {
		times := []time.Time{
			time.Unix(0, value.CreationTime.Nanoseconds()),
			time.Unix(0, value.LastAccessTime.Nanoseconds()),
			time.Unix(0, value.LastWriteTime.Nanoseconds()),
		}

		for _, time := range times {
			if lastTouched.Before(time) {
				lastTouched = time
			}
		}
	}

	return
}

