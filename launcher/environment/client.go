// Copyright 2014 The jenkins-client-launcher Authors. All rights reserved.
// Use of this source code is governed by a MIT license that can be found in the LICENSE file.

package environment

import (
	"github.com/jkellerer/jenkins-client-launcher/launcher/modes"
	"github.com/jkellerer/jenkins-client-launcher/launcher/util"
	"io"
	"fmt"
	"os"
	"path/filepath"
	"net/http"
	"time"
)

const (
	ClientJarName         = "slave.jar"
	ClientJarDownloadName = "~slave.jar.download"
	ClientJarURL          = "jnlpJars/slave.jar"
)

// Implements a downloader that ensures that the latest Jenkins client (slave.jar) is
// downloaded before the client mode starts.
type JenkinsClientDownloader struct {
	util.AnyConfigAcceptor
}

func (self *JenkinsClientDownloader) Name() string {
	return "Jenkins Client Downloader"
}

func (self *JenkinsClientDownloader) Prepare(config *util.Config) {
	util.ClientJar, _ = filepath.Abs(ClientJarName)

	modes.RegisterModeListener(func(mode modes.ExecutableMode, nextStatus int32, config *util.Config) {
		if mode.Name() == "client" && nextStatus == modes.ModeStarting && config.HasCIConnection() {
			if err := self.downloadJar(config); err != nil {
				jar, e := os.Open(ClientJarName); defer jar.Close()
				if os.IsNotExist(e) {
					panic(fmt.Sprintf("No jenkins client: %s", err))
				} else {
					util.GOut("DOWNLOAD", "%s", err)
				}
			}
		}
	})
}

func (self *JenkinsClientDownloader) downloadJar(config *util.Config) error {
	util.GOut("DOWNLOAD", "Getting latest Jenkins client %v", (config.CIHostURI+"/"+ClientJarURL))

	// Create the HTTP request.
	request, err := config.CIRequest("GET", ClientJarURL, nil)
	if err != nil {
		return err
	}

	if fi, err := os.Stat(ClientJarName); err == nil {
		request.Header.Add("If-Modified-Since", fi.ModTime().Format(http.TimeFormat))
	}

	// Perform the HTTP request.
	var source io.ReadCloser
	sourceTime := time.Now()
	if response, err := config.CIClient().Do(request); err == nil {
		defer response.Body.Close()

		source = response.Body

		if response.StatusCode == 304 {
			util.GOut("DOWNLOAD", "Jenkins client is up-to-date, no need to download.")
			return nil
		} else if response.StatusCode != 200 {
			return fmt.Errorf("Failed downloading jenkins client. Cause: HTTP-%v %v", response.StatusCode, response.Status)
		}

		if value := response.Header.Get("Last-Modified"); value != "" {
			if time, err := http.ParseTime(value); err == nil {
				sourceTime = time
			}
		}
	} else {
		return fmt.Errorf("Failed downloading jenkins client. Connect failed. Cause: %v", err)
	}

	target, err := os.Create(ClientJarDownloadName); defer target.Close()

	if err != nil {
		return fmt.Errorf("Failed downloading jenkins client. Cannot create local file. Cause: %v", err)
	}

	if _, err = io.Copy(target, source); err == nil {
		target.Close()
		if err = os.Remove(ClientJarName); err == nil || os.IsNotExist(err) {
			if err = os.Rename(ClientJarDownloadName, ClientJarName); err == nil {
				os.Chtimes(ClientJarName, sourceTime, sourceTime)
			}
		}
		return err
	} else {
		return fmt.Errorf("Failed downloading jenkins client. Transfer failed. Cause: %v", err)
	}
}

// Registering the downloader.
var _ = RegisterPreparer(new(JenkinsClientDownloader))
