package sysinfo

import (
	"os"
	"syscall"
)

// Hostname returns the host name.
func Hostname() (string, error) {
	// Try the builtin API first, which is designed to match the output of
	// /bin/hostname, and fallback to uname(2) if that fails to match the
	// behavior of gethostname(2) as implemented by glibc. On Linux, all
	// these method should result in the same value because sethostname(2)
	// limits the hostname to 64 bytes, the same size of the nodename field
	// returned by uname(2). Note that is correspondence is not true on
	// other platforms.
	//
	// os.Hostname failures should be exceedingly rare, however some systems
	// configure SELinux to deny read access to /proc/sys/kernel/hostname.
	// Redhat's OpenShift platform for example. os.Hostname can also fail if
	// some or all of /proc has been hidden via chroot(2) or manipulation of
	// the current processes' filesystem namespace via the cgroups APIs.
	// Docker is an example of a tool that can configure such an
	// environment.
	name, err := os.Hostname()
	if err == nil {
		return name, nil
	}

	var uts syscall.Utsname
	if err2 := syscall.Uname(&uts); err2 != nil {
		// The man page documents only one possible error for uname(2),
		// suggesting that as long as the buffer given is valid, the
		// call will never fail. Return the original error in the hope
		// it provides more relevant information about why the hostname
		// can't be retrieved.
		return "", err
	}

	// Convert Nodename to a Go string.
	buf := make([]byte, 0, len(uts.Nodename))
	for _, c := range uts.Nodename {
		if c == 0 {
			break
		}
		buf = append(buf, byte(c))
	}

	return string(buf), nil
}
