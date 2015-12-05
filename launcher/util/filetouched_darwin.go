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

	if value, ok := info.Sys().(*syscall.Stat_t); ok {
		times := []time.Time{
			time.Unix(int64(value.Atimespec.Sec), int64(value.Atimespec.Nsec)),
			time.Unix(int64(value.Atimespec.Sec), int64(value.Atimespec.Nsec)),
		}

		for _, time := range times {
			if lastTouched.Before(time) {
				lastTouched = time
			}
		}
	}

	return
}

