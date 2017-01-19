// +build darwin dragonfly freebsd netbsd openbsd solaris windows

package executor

import (
	"os"

	cstructs "github.com/hashicorp/nomad/client/structs"
	"github.com/mitchellh/go-ps"
)

func (e *UniversalExecutor) configureChroot() error {
	return nil
}

func (e *UniversalExecutor) removeChrootMounts() error {
	return nil
}

func (e *UniversalExecutor) runAs(userid string) error {
	return nil
}

func (e *UniversalExecutor) applyLimits(pid int) error {
	return nil
}

func (e *UniversalExecutor) configureIsolation() error {
	return nil
}

func (e *UniversalExecutor) Stats() (*cstructs.TaskResourceUsage, error) {
	pidStats, err := e.pidStats()
	if err != nil {
		return nil, err
	}
	return e.aggregatedResourceUsage(pidStats), nil
}

func (e *UniversalExecutor) getAllPids() (map[int]*nomadPid, error) {
	allProcesses, err := ps.Processes()
	if err != nil {
		return nil, err
	}
	return e.scanPids(os.Getpid(), allProcesses)
}
