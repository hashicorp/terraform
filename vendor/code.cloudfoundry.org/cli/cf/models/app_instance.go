package models

import "time"

type InstanceState string

const (
	InstanceStarting InstanceState = "starting"
	InstanceRunning  InstanceState = "running"
	InstanceFlapping InstanceState = "flapping"
	InstanceDown     InstanceState = "down"
	InstanceCrashed  InstanceState = "crashed"
)

type AppInstanceFields struct {
	State     InstanceState
	Details   string
	Since     time.Time
	CPUUsage  float64 // percentage
	DiskQuota int64   // in bytes
	DiskUsage int64
	MemQuota  int64
	MemUsage  int64
}
