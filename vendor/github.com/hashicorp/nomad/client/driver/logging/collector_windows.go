package logging

import (
	"log"

	"github.com/hashicorp/nomad/client/allocdir"
	cstructs "github.com/hashicorp/nomad/client/driver/structs"
	"github.com/hashicorp/nomad/nomad/structs"
)

// LogCollectorContext holds context to configure the syslog server
type LogCollectorContext struct {
	// TaskName is the name of the Task
	TaskName string

	// AllocDir is the handle to do operations on the alloc dir of
	// the task
	AllocDir *allocdir.AllocDir

	// LogConfig provides configuration related to log rotation
	LogConfig *structs.LogConfig

	// PortUpperBound is the upper bound of the ports that we can use to start
	// the syslog server
	PortUpperBound uint

	// PortLowerBound is the lower bound of the ports that we can use to start
	// the syslog server
	PortLowerBound uint
}

// SyslogCollectorState holds the address and islation information of a launched
// syslog server
type SyslogCollectorState struct {
	IsolationConfig *cstructs.IsolationConfig
	Addr            string
}

// LogCollector is an interface which allows a driver to launch a log server
// and update log configuration
type LogCollector interface {
	LaunchCollector(ctx *LogCollectorContext) (*SyslogCollectorState, error)
	Exit() error
	UpdateLogConfig(logConfig *structs.LogConfig) error
}

// SyslogCollector is a LogCollector which starts a syslog server and does
// rotation to incoming stream
type SyslogCollector struct {
}

// NewSyslogCollector returns an implementation of the SyslogCollector
func NewSyslogCollector(logger *log.Logger) *SyslogCollector {
	return &SyslogCollector{}
}

// LaunchCollector launches a new syslog server and starts writing log lines to
// files and rotates them
func (s *SyslogCollector) LaunchCollector(ctx *LogCollectorContext) (*SyslogCollectorState, error) {
	return nil, nil
}

// Exit kills the syslog server
func (s *SyslogCollector) Exit() error {
	return nil
}

// UpdateLogConfig updates the log configuration
func (s *SyslogCollector) UpdateLogConfig(logConfig *structs.LogConfig) error {
	return nil
}
