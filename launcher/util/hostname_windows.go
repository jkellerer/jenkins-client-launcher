package util

import (
	"os"
	"syscall"
	"unicode/utf16"
	"unsafe"
)

var (
	modkernel32            = syscall.NewLazyDLL("kernel32.dll")
	procGetComputerNameExW = modkernel32.NewProc("GetComputerNameExW")
)

const (
	computerNameNetBIOS     = 0
	computerNameDnsHostname = 1
	computerNameDnsDomain   = 2
)

// Returns the DNS hostname if available and falls back to the NetBIOS name (via "os.Hostname()") if DNS hostname is not available.
func Hostname() (name string, err error) {
	if name, err = dnsComputerName(); err == nil {
		return
	} else {
		return os.Hostname()
	}
}

func dnsComputerName() (name string, err error) {
	var (
		n uint32 = 1024
		b        = make([]uint16, n)
	)

	if e := getComputerNameEx(computerNameDnsHostname, &b[0], &n); e != nil {
		return "", e
	}
	return string(utf16.Decode(b[0:n])), nil
}

func getComputerNameEx(computerNameFormat uint32, buf *uint16, n *uint32) (err error) {
	r1, _, lastError := procGetComputerNameExW.Call(uintptr(computerNameFormat), uintptr(unsafe.Pointer(buf)), uintptr(unsafe.Pointer(n)))
	if r1 == 0 {
		if lastError != nil {
			err = lastError
		} else {
			err = syscall.EINVAL
		}
	}

	return
}

