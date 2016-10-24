// +build darwin dragonfly freebsd netbsd openbsd solaris windows

package executor

import (
	dstructs "github.com/hashicorp/nomad/client/driver/structs"
)

// resourceContainerContext is a platform-specific struct for managing a
// resource container.
type resourceContainerContext struct {
}

func clientCleanup(ic *dstructs.IsolationConfig, pid int) error {
	return nil
}

func (rc *resourceContainerContext) executorCleanup() error {
	return nil
}

func (rc *resourceContainerContext) getIsolationConfig() *dstructs.IsolationConfig {
	return nil
}
