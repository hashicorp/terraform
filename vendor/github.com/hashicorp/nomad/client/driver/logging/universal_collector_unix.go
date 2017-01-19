// +build darwin dragonfly freebsd linux netbsd openbsd solaris

package logging

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"log/syslog"
	"net"
	"os"
	"runtime"

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
	addr      net.Addr
	logConfig *structs.LogConfig
	ctx       *LogCollectorContext

	lro        *FileRotator
	lre        *FileRotator
	server     *SyslogServer
	syslogChan chan *SyslogMessage
	taskDir    string

	logger *log.Logger
}

// NewSyslogCollector returns an implementation of the SyslogCollector
func NewSyslogCollector(logger *log.Logger) *SyslogCollector {
	return &SyslogCollector{logger: logger, syslogChan: make(chan *SyslogMessage, 2048)}
}

// LaunchCollector launches a new syslog server and starts writing log lines to
// files and rotates them
func (s *SyslogCollector) LaunchCollector(ctx *LogCollectorContext) (*SyslogCollectorState, error) {
	l, err := s.getListener(ctx.PortLowerBound, ctx.PortUpperBound)
	if err != nil {
		return nil, err
	}
	s.logger.Printf("[DEBUG] sylog-server: launching syslog server on addr: %v", l.Addr().String())
	s.ctx = ctx
	// configuring the task dir
	if err := s.configureTaskDir(); err != nil {
		return nil, err
	}

	s.server = NewSyslogServer(l, s.syslogChan, s.logger)
	go s.server.Start()
	logFileSize := int64(ctx.LogConfig.MaxFileSizeMB * 1024 * 1024)

	lro, err := NewFileRotator(ctx.AllocDir.LogDir(), fmt.Sprintf("%v.stdout", ctx.TaskName),
		ctx.LogConfig.MaxFiles, logFileSize, s.logger)

	if err != nil {
		return nil, err
	}
	s.lro = lro

	lre, err := NewFileRotator(ctx.AllocDir.LogDir(), fmt.Sprintf("%v.stderr", ctx.TaskName),
		ctx.LogConfig.MaxFiles, logFileSize, s.logger)
	if err != nil {
		return nil, err
	}
	s.lre = lre

	go s.collectLogs(lre, lro)
	syslogAddr := fmt.Sprintf("%s://%s", l.Addr().Network(), l.Addr().String())
	return &SyslogCollectorState{Addr: syslogAddr}, nil
}

func (s *SyslogCollector) collectLogs(we io.Writer, wo io.Writer) {
	for logParts := range s.syslogChan {
		// If the severity of the log line is err then we write to stderr
		// otherwise all messages go to stdout
		if logParts.Severity == syslog.LOG_ERR {
			s.lre.Write(logParts.Message)
			s.lre.Write([]byte{'\n'})
		} else {
			s.lro.Write(logParts.Message)
			s.lro.Write([]byte{'\n'})
		}
	}
}

// Exit kills the syslog server
func (s *SyslogCollector) Exit() error {
	s.server.Shutdown()
	s.lre.Close()
	s.lro.Close()
	return nil
}

// UpdateLogConfig updates the log configuration
func (s *SyslogCollector) UpdateLogConfig(logConfig *structs.LogConfig) error {
	s.ctx.LogConfig = logConfig
	if s.lro == nil {
		return fmt.Errorf("log rotator for stdout doesn't exist")
	}
	s.lro.MaxFiles = logConfig.MaxFiles
	s.lro.FileSize = int64(logConfig.MaxFileSizeMB * 1024 * 1024)

	if s.lre == nil {
		return fmt.Errorf("log rotator for stderr doesn't exist")
	}
	s.lre.MaxFiles = logConfig.MaxFiles
	s.lre.FileSize = int64(logConfig.MaxFileSizeMB * 1024 * 1024)
	return nil
}

// configureTaskDir sets the task dir in the SyslogCollector
func (s *SyslogCollector) configureTaskDir() error {
	taskDir, ok := s.ctx.AllocDir.TaskDirs[s.ctx.TaskName]
	if !ok {
		return fmt.Errorf("couldn't find task directory for task %v", s.ctx.TaskName)
	}
	s.taskDir = taskDir
	return nil
}

// getFreePort returns a free port ready to be listened on between upper and
// lower bounds
func (s *SyslogCollector) getListener(lowerBound uint, upperBound uint) (net.Listener, error) {
	if runtime.GOOS == "windows" {
		return s.listenerTCP(lowerBound, upperBound)
	}

	return s.listenerUnix()
}

// listenerTCP creates a TCP listener using an unused port between an upper and
// lower bound
func (s *SyslogCollector) listenerTCP(lowerBound uint, upperBound uint) (net.Listener, error) {
	for i := lowerBound; i <= upperBound; i++ {
		addr, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("localhost:%v", i))
		if err != nil {
			return nil, err
		}
		l, err := net.ListenTCP("tcp", addr)
		if err != nil {
			continue
		}
		return l, nil
	}
	return nil, fmt.Errorf("No free port found")
}

// listenerUnix creates a Unix domain socket
func (s *SyslogCollector) listenerUnix() (net.Listener, error) {
	f, err := ioutil.TempFile("", "plugin")
	if err != nil {
		return nil, err
	}
	path := f.Name()

	if err := f.Close(); err != nil {
		return nil, err
	}
	if err := os.Remove(path); err != nil {
		return nil, err
	}

	return net.Listen("unix", path)
}
