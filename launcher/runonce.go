package launcher

import (
	"os"
	"strconv"
	"io/ioutil"
	"time"
)

const (
	PidName = "launcher.pid"
)

var pidUpdateRate = time.Second * 3
var pidStaleDuration = time.Second * 5

// Checks if another launcher is already running inside the current working path.
// The returned state should be closed before leaving the application to avoid stale PID files.
func CheckIfAlreadyRunning() AlreadyRunning {
	if previousPid, pidLoadError := readPidFile(); pidLoadError == nil && isProcessExisting(previousPid) {
		return AlreadyRunning(true)
	}

	value := AlreadyRunning(false)
	writePidFile()
	go value.keepPIDFileAlive()
	return value
}

// Declares a closable boolean that describes whether the launcher is already running or not.
type AlreadyRunning bool

// Closes the running state (this also removes the current PID file).
func (running *AlreadyRunning) Close() {
	if !*running {
		removePidFile()
	}
}

func (running *AlreadyRunning) keepPIDFileAlive() {
	if !*running {
		for {
			if fi, err := os.Stat(PidName); err == nil && fi.Size() > 0 {
				os.Chtimes(PidName, time.Now(), time.Now())
				time.Sleep(pidUpdateRate)
			} else {
				return
			}
		}
	}
}

// Returns true if the specified pid points to a running process.
func isProcessExisting(pid int) bool {
	if p, err := os.FindProcess(pid); err == nil {
		defer p.Release()
		return true
	}
	return false
}

// Reads the pid that is stored inside the default pid file.
func readPidFile() (int, error) {
	if content, err := ioutil.ReadFile(PidName); err == nil {
		if fi, err := os.Stat(PidName); err == nil && fi.ModTime().After(time.Now().Add(-pidStaleDuration)) {
			return strconv.Atoi(string(content))
		} else {
			return -1, err
		}
	} else {
		return -1, err
	}
}

// Removes the pid file.
func removePidFile() {
	_ = os.Remove(PidName)
}

// Writes the pid of this process into the default pid file.
func writePidFile() {
	pid := os.Getpid()
	_ = ioutil.WriteFile(PidName, []byte(strconv.Itoa(pid)), os.ModeTemporary)
}
