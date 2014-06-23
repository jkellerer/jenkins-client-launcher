// Copyright 2014 The jenkins-client-launcher Authors. All rights reserved.
// Use of this source code is governed by a MIT license that can be found in the LICENSE file.

package util

import (
	"testing"
	"os"
)

func TestLoadsDefaultConfigWhenFileIsMissing(t *testing.T) {
	in, out := NewConfig("~non-existing.xml").String(), NewDefaultConfig().String()
	if in != out {
		t.Errorf("NewConfig(~non-existing.xml) = %v, want %v", in, out)
	}
}

func TestNeedsSaveIsTrueWhenFileIsMissing(t *testing.T) {
	in, out := NewConfig("~non-existing.xml").NeedsSave, true
	if in != out {
		t.Errorf("NewConfig(~non-existing.xml).NeedsSave = %v, want %v", in, out)
	}
}

func TestRunModeDefaultsToClient(t *testing.T) {
	in, out := NewConfig("~non-existing.xml").RunMode, "client"
	if in != out {
		t.Errorf("NewConfig(~non-existing.xml).RunMode = %v, want %v", in, out)
	}
}

func TestCanSafeConfig(t *testing.T) {
	defer os.Remove("~existing.xml")

	config, defaultConfig := NewDefaultConfig(), NewDefaultConfig()
	config.RestartTriggerTokens = []string {"1", "2"}
	config.Save("~existing.xml")
	config = NewConfig("~existing.xml")

	in, out := config.String(), defaultConfig.String()
	if in == out {
		t.Errorf("NewConfig(~existing.xml) = %v, should not be %v", in, out)
	}
}
