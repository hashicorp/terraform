package sysinfo

import (
	"time"
)

// Usage contains process process times.
type Usage struct {
	System time.Duration
	User   time.Duration
}
