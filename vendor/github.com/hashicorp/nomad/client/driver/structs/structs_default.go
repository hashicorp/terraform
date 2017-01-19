// +build darwin dragonfly freebsd netbsd openbsd solaris windows

package structs

// IsolationConfig has information about the isolation mechanism the executor
// uses to put resource constraints and isolation on the user process.  The
// default implementation is empty.  Platforms that support resource isolation
// (e.g. Linux's Cgroups) should build their own platform-specific copy.  This
// information is transmitted via RPC so it is not permissable to change the
// API.
type IsolationConfig struct {
}
