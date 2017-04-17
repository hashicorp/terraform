// +build !linux

package sysinfo

import "os"

// Hostname returns the host name.
func Hostname() (string, error) {
	return os.Hostname()
}
