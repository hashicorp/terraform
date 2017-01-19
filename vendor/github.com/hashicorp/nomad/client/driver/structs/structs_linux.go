package structs

import cgroupConfig "github.com/opencontainers/runc/libcontainer/configs"

// IsolationConfig has information about the isolation mechanism the executor
// uses to put resource constraints and isolation on the user process
type IsolationConfig struct {
	Cgroup      *cgroupConfig.Cgroup
	CgroupPaths map[string]string
}
