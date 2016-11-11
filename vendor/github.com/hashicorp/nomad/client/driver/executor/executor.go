package executor

import (
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/hashicorp/go-multierror"
	"github.com/mitchellh/go-ps"
	"github.com/shirou/gopsutil/process"

	"github.com/hashicorp/nomad/client/allocdir"
	"github.com/hashicorp/nomad/client/driver/env"
	"github.com/hashicorp/nomad/client/driver/logging"
	"github.com/hashicorp/nomad/client/stats"
	"github.com/hashicorp/nomad/command/agent/consul"
	shelpers "github.com/hashicorp/nomad/helper/stats"
	"github.com/hashicorp/nomad/nomad/structs"
	"github.com/hashicorp/nomad/nomad/structs/config"

	dstructs "github.com/hashicorp/nomad/client/driver/structs"
	cstructs "github.com/hashicorp/nomad/client/structs"
)

const (
	// pidScanInterval is the interval at which the executor scans the process
	// tree for finding out the pids that the executor and it's child processes
	// have forked
	pidScanInterval = 5 * time.Second
)

var (
	// The statistics the basic executor exposes
	ExecutorBasicMeasuredMemStats = []string{"RSS", "Swap"}
	ExecutorBasicMeasuredCpuStats = []string{"System Mode", "User Mode", "Percent"}
)

// Executor is the interface which allows a driver to launch and supervise
// a process
type Executor interface {
	LaunchCmd(command *ExecCommand, ctx *ExecutorContext) (*ProcessState, error)
	LaunchSyslogServer(ctx *ExecutorContext) (*SyslogServerState, error)
	Wait() (*ProcessState, error)
	ShutDown() error
	Exit() error
	UpdateLogConfig(logConfig *structs.LogConfig) error
	UpdateTask(task *structs.Task) error
	SyncServices(ctx *ConsulContext) error
	DeregisterServices() error
	Version() (*ExecutorVersion, error)
	Stats() (*cstructs.TaskResourceUsage, error)
}

// ConsulContext holds context to configure the Consul client and run checks
type ConsulContext struct {
	// ConsulConfig contains the configuration information for talking
	// with this Nomad Agent's Consul Agent.
	ConsulConfig *config.ConsulConfig

	// ContainerID is the ID of the container
	ContainerID string

	// TLSCert is the cert which docker client uses while interactng with the docker
	// daemon over TLS
	TLSCert string

	// TLSCa is the CA which the docker client uses while interacting with the docker
	// daeemon over TLS
	TLSCa string

	// TLSKey is the TLS key which the docker client uses while interacting with
	// the docker daemon
	TLSKey string

	// DockerEndpoint is the endpoint of the docker daemon
	DockerEndpoint string
}

// ExecutorContext holds context to configure the command user
// wants to run and isolate it
type ExecutorContext struct {
	// TaskEnv holds information about the environment of a Task
	TaskEnv *env.TaskEnvironment

	// AllocDir is the handle to do operations on the alloc dir of
	// the task
	AllocDir *allocdir.AllocDir

	// Task is the task whose executor is being launched
	Task *structs.Task

	// AllocID is the allocation id to which the task belongs
	AllocID string

	// A mapping of directories on the host OS to attempt to embed inside each
	// task's chroot.
	ChrootEnv map[string]string

	// Driver is the name of the driver that invoked the executor
	Driver string

	// PortUpperBound is the upper bound of the ports that we can use to start
	// the syslog server
	PortUpperBound uint

	// PortLowerBound is the lower bound of the ports that we can use to start
	// the syslog server
	PortLowerBound uint
}

// ExecCommand holds the user command, args, and other isolation related
// settings.
type ExecCommand struct {
	// Cmd is the command that the user wants to run.
	Cmd string

	// Args is the args of the command that the user wants to run.
	Args []string

	// FSIsolation determines whether the command would be run in a chroot.
	FSIsolation bool

	// User is the user which the executor uses to run the command.
	User string

	// ResourceLimits determines whether resource limits are enforced by the
	// executor.
	ResourceLimits bool
}

// ProcessState holds information about the state of a user process.
type ProcessState struct {
	Pid             int
	ExitCode        int
	Signal          int
	IsolationConfig *dstructs.IsolationConfig
	Time            time.Time
}

// nomadPid holds a pid and it's cpu percentage calculator
type nomadPid struct {
	pid           int
	cpuStatsTotal *stats.CpuStats
	cpuStatsUser  *stats.CpuStats
	cpuStatsSys   *stats.CpuStats
}

// SyslogServerState holds the address and islation information of a launched
// syslog server
type SyslogServerState struct {
	IsolationConfig *dstructs.IsolationConfig
	Addr            string
}

// ExecutorVersion is the version of the executor
type ExecutorVersion struct {
	Version string
}

func (v *ExecutorVersion) GoString() string {
	return v.Version
}

// UniversalExecutor is an implementation of the Executor which launches and
// supervises processes. In addition to process supervision it provides resource
// and file system isolation
type UniversalExecutor struct {
	cmd     exec.Cmd
	ctx     *ExecutorContext
	command *ExecCommand

	pids                map[int]*nomadPid
	pidLock             sync.RWMutex
	taskDir             string
	exitState           *ProcessState
	processExited       chan interface{}
	fsIsolationEnforced bool

	lre         *logging.FileRotator
	lro         *logging.FileRotator
	rotatorLock sync.Mutex

	shutdownCh chan struct{}

	syslogServer *logging.SyslogServer
	syslogChan   chan *logging.SyslogMessage

	resConCtx resourceContainerContext

	consulSyncer   *consul.Syncer
	consulCtx      *ConsulContext
	totalCpuStats  *stats.CpuStats
	userCpuStats   *stats.CpuStats
	systemCpuStats *stats.CpuStats
	logger         *log.Logger
}

// NewExecutor returns an Executor
func NewExecutor(logger *log.Logger) Executor {
	if err := shelpers.Init(); err != nil {
		logger.Printf("[FATAL] executor: unable to initialize stats: %v", err)
		return nil
	}

	exec := &UniversalExecutor{
		logger:         logger,
		processExited:  make(chan interface{}),
		totalCpuStats:  stats.NewCpuStats(),
		userCpuStats:   stats.NewCpuStats(),
		systemCpuStats: stats.NewCpuStats(),
		pids:           make(map[int]*nomadPid),
	}

	return exec
}

// Version returns the api version of the executor
func (e *UniversalExecutor) Version() (*ExecutorVersion, error) {
	return &ExecutorVersion{Version: "1.0.0"}, nil
}

// LaunchCmd launches a process and returns it's state. It also configures an
// applies isolation on certain platforms.
func (e *UniversalExecutor) LaunchCmd(command *ExecCommand, ctx *ExecutorContext) (*ProcessState, error) {
	e.logger.Printf("[DEBUG] executor: launching command %v %v", command.Cmd, strings.Join(command.Args, " "))

	e.ctx = ctx
	e.command = command

	// setting the user of the process
	if command.User != "" {
		e.logger.Printf("[DEBUG] executor: running command as %s", command.User)
		if err := e.runAs(command.User); err != nil {
			return nil, err
		}
	}

	// configuring the task dir
	if err := e.configureTaskDir(); err != nil {
		return nil, err
	}

	e.ctx.TaskEnv.Build()
	// configuring the chroot, resource container, and start the plugin
	// process in the chroot.
	if err := e.configureIsolation(); err != nil {
		return nil, err
	}
	// Apply ourselves into the resource container. The executor MUST be in
	// the resource container before the user task is started, otherwise we
	// are subject to a fork attack in which a process escapes isolation by
	// immediately forking.
	if err := e.applyLimits(os.Getpid()); err != nil {
		return nil, err
	}

	// Setup the loggers
	if err := e.configureLoggers(); err != nil {
		return nil, err
	}
	e.cmd.Stdout = e.lro
	e.cmd.Stderr = e.lre

	// Look up the binary path and make it executable
	absPath, err := e.lookupBin(ctx.TaskEnv.ReplaceEnv(command.Cmd))
	if err != nil {
		return nil, err
	}

	if err := e.makeExecutable(absPath); err != nil {
		return nil, err
	}

	path := absPath

	// Determine the path to run as it may have to be relative to the chroot.
	if e.fsIsolationEnforced {
		rel, err := filepath.Rel(e.taskDir, path)
		if err != nil {
			return nil, err
		}
		path = rel
	}

	// Set the commands arguments
	e.cmd.Path = path
	e.cmd.Args = append([]string{e.cmd.Path}, ctx.TaskEnv.ParseAndReplace(command.Args)...)
	e.cmd.Env = ctx.TaskEnv.EnvList()

	// Start the process
	if err := e.cmd.Start(); err != nil {
		return nil, err
	}
	go e.collectPids()
	go e.wait()
	ic := e.resConCtx.getIsolationConfig()
	return &ProcessState{Pid: e.cmd.Process.Pid, ExitCode: -1, IsolationConfig: ic, Time: time.Now()}, nil
}

// configureLoggers sets up the standard out/error file rotators
func (e *UniversalExecutor) configureLoggers() error {
	e.rotatorLock.Lock()
	defer e.rotatorLock.Unlock()

	logFileSize := int64(e.ctx.Task.LogConfig.MaxFileSizeMB * 1024 * 1024)
	if e.lro == nil {
		lro, err := logging.NewFileRotator(e.ctx.AllocDir.LogDir(), fmt.Sprintf("%v.stdout", e.ctx.Task.Name),
			e.ctx.Task.LogConfig.MaxFiles, logFileSize, e.logger)
		if err != nil {
			return err
		}
		e.lro = lro
	}

	if e.lre == nil {
		lre, err := logging.NewFileRotator(e.ctx.AllocDir.LogDir(), fmt.Sprintf("%v.stderr", e.ctx.Task.Name),
			e.ctx.Task.LogConfig.MaxFiles, logFileSize, e.logger)
		if err != nil {
			return err
		}
		e.lre = lre
	}
	return nil
}

// Wait waits until a process has exited and returns it's exitcode and errors
func (e *UniversalExecutor) Wait() (*ProcessState, error) {
	<-e.processExited
	return e.exitState, nil
}

// COMPAT: prior to Nomad 0.3.2, UpdateTask didn't exist.
// UpdateLogConfig updates the log configuration
func (e *UniversalExecutor) UpdateLogConfig(logConfig *structs.LogConfig) error {
	e.ctx.Task.LogConfig = logConfig
	if e.lro == nil {
		return fmt.Errorf("log rotator for stdout doesn't exist")
	}
	e.lro.MaxFiles = logConfig.MaxFiles
	e.lro.FileSize = int64(logConfig.MaxFileSizeMB * 1024 * 1024)

	if e.lre == nil {
		return fmt.Errorf("log rotator for stderr doesn't exist")
	}
	e.lre.MaxFiles = logConfig.MaxFiles
	e.lre.FileSize = int64(logConfig.MaxFileSizeMB * 1024 * 1024)
	return nil
}

func (e *UniversalExecutor) UpdateTask(task *structs.Task) error {
	e.ctx.Task = task

	// Updating Log Config
	fileSize := int64(task.LogConfig.MaxFileSizeMB * 1024 * 1024)
	e.lro.MaxFiles = task.LogConfig.MaxFiles
	e.lro.FileSize = fileSize
	e.lre.MaxFiles = task.LogConfig.MaxFiles
	e.lre.FileSize = fileSize

	// Re-syncing task with Consul agent
	if e.consulSyncer != nil {
		e.interpolateServices(e.ctx.Task)
		domain := consul.NewExecutorDomain(e.ctx.AllocID, task.Name)
		serviceMap := generateServiceKeys(e.ctx.AllocID, task.Services)
		e.consulSyncer.SetServices(domain, serviceMap)
	}
	return nil
}

// generateServiceKeys takes a list of interpolated Nomad Services and returns a map
// of ServiceKeys to Nomad Services.
func generateServiceKeys(allocID string, services []*structs.Service) map[consul.ServiceKey]*structs.Service {
	keys := make(map[consul.ServiceKey]*structs.Service, len(services))
	for _, service := range services {
		key := consul.GenerateServiceKey(service)
		keys[key] = service
	}
	return keys
}

func (e *UniversalExecutor) wait() {
	defer close(e.processExited)
	err := e.cmd.Wait()
	ic := e.resConCtx.getIsolationConfig()
	if err == nil {
		e.exitState = &ProcessState{Pid: 0, ExitCode: 0, IsolationConfig: ic, Time: time.Now()}
		return
	}
	exitCode := 1
	var signal int
	if exitErr, ok := err.(*exec.ExitError); ok {
		if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
			exitCode = status.ExitStatus()
			if status.Signaled() {
				// bash(1) uses the lower 7 bits of a uint8
				// to indicate normal program failure (see
				// <sysexits.h>). If a process terminates due
				// to a signal, encode the signal number to
				// indicate which signal caused the process
				// to terminate.  Mirror this exit code
				// encoding scheme.
				const exitSignalBase = 128
				signal = int(status.Signal())
				exitCode = exitSignalBase + signal
			}
		}
	} else {
		e.logger.Printf("[DEBUG] executor: unexpected Wait() error type: %v", err)
	}

	e.exitState = &ProcessState{Pid: 0, ExitCode: exitCode, Signal: signal, IsolationConfig: ic, Time: time.Now()}
}

var (
	// finishedErr is the error message received when trying to kill and already
	// exited process.
	finishedErr = "os: process already finished"
)

// ClientCleanup is the cleanup routine that a Nomad Client uses to remove the
// reminants of a child UniversalExecutor.
func ClientCleanup(ic *dstructs.IsolationConfig, pid int) error {
	return clientCleanup(ic, pid)
}

// Exit cleans up the alloc directory, destroys resource container and kills the
// user process
func (e *UniversalExecutor) Exit() error {
	var merr multierror.Error
	if e.syslogServer != nil {
		e.syslogServer.Shutdown()
	}
	e.lre.Close()
	e.lro.Close()

	if e.consulSyncer != nil {
		e.consulSyncer.Shutdown()
	}

	// If the executor did not launch a process, return.
	if e.command == nil {
		return nil
	}

	// Prefer killing the process via the resource container.
	if e.cmd.Process != nil && !e.command.ResourceLimits {
		proc, err := os.FindProcess(e.cmd.Process.Pid)
		if err != nil {
			e.logger.Printf("[ERR] executor: can't find process with pid: %v, err: %v",
				e.cmd.Process.Pid, err)
		} else if err := proc.Kill(); err != nil && err.Error() != finishedErr {
			merr.Errors = append(merr.Errors,
				fmt.Errorf("can't kill process with pid: %v, err: %v", e.cmd.Process.Pid, err))
		}
	}

	if e.command.ResourceLimits {
		if err := e.resConCtx.executorCleanup(); err != nil {
			merr.Errors = append(merr.Errors, err)
		}
	}

	if e.command.FSIsolation {
		if err := e.removeChrootMounts(); err != nil {
			merr.Errors = append(merr.Errors, err)
		}
	}
	return merr.ErrorOrNil()
}

// Shutdown sends an interrupt signal to the user process
func (e *UniversalExecutor) ShutDown() error {
	if e.cmd.Process == nil {
		return fmt.Errorf("executor.shutdown error: no process found")
	}
	proc, err := os.FindProcess(e.cmd.Process.Pid)
	if err != nil {
		return fmt.Errorf("executor.shutdown failed to find process: %v", err)
	}
	if runtime.GOOS == "windows" {
		if err := proc.Kill(); err != nil && err.Error() != finishedErr {
			return err
		}
		return nil
	}
	if err = proc.Signal(os.Interrupt); err != nil && err.Error() != finishedErr {
		return fmt.Errorf("executor.shutdown error: %v", err)
	}
	return nil
}

// SyncServices syncs the services of the task that the executor is running with
// Consul
func (e *UniversalExecutor) SyncServices(ctx *ConsulContext) error {
	e.logger.Printf("[INFO] executor: registering services")
	e.consulCtx = ctx
	if e.consulSyncer == nil {
		cs, err := consul.NewSyncer(ctx.ConsulConfig, e.shutdownCh, e.logger)
		if err != nil {
			return err
		}
		e.consulSyncer = cs
		go e.consulSyncer.Run()
	}
	e.interpolateServices(e.ctx.Task)
	e.consulSyncer.SetDelegatedChecks(e.createCheckMap(), e.createCheck)
	e.consulSyncer.SetAddrFinder(e.ctx.Task.FindHostAndPortFor)
	domain := consul.NewExecutorDomain(e.ctx.AllocID, e.ctx.Task.Name)
	serviceMap := generateServiceKeys(e.ctx.AllocID, e.ctx.Task.Services)
	e.consulSyncer.SetServices(domain, serviceMap)
	return nil
}

// DeregisterServices removes the services of the task that the executor is
// running from Consul
func (e *UniversalExecutor) DeregisterServices() error {
	e.logger.Printf("[INFO] executor: de-registering services and shutting down consul service")
	if e.consulSyncer != nil {
		return e.consulSyncer.Shutdown()
	}
	return nil
}

// pidStats returns the resource usage stats per pid
func (e *UniversalExecutor) pidStats() (map[string]*cstructs.ResourceUsage, error) {
	stats := make(map[string]*cstructs.ResourceUsage)
	e.pidLock.RLock()
	pids := make(map[int]*nomadPid, len(e.pids))
	for k, v := range e.pids {
		pids[k] = v
	}
	e.pidLock.RUnlock()
	for pid, np := range pids {
		p, err := process.NewProcess(int32(pid))
		if err != nil {
			e.logger.Printf("[DEBUG] executor: unable to create new process with pid: %v", pid)
			continue
		}
		ms := &cstructs.MemoryStats{}
		if memInfo, err := p.MemoryInfo(); err == nil {
			ms.RSS = memInfo.RSS
			ms.Swap = memInfo.Swap
			ms.Measured = ExecutorBasicMeasuredMemStats
		}

		cs := &cstructs.CpuStats{}
		if cpuStats, err := p.Times(); err == nil {
			cs.SystemMode = np.cpuStatsSys.Percent(cpuStats.System * float64(time.Second))
			cs.UserMode = np.cpuStatsUser.Percent(cpuStats.User * float64(time.Second))
			cs.Measured = ExecutorBasicMeasuredCpuStats

			// calculate cpu usage percent
			cs.Percent = np.cpuStatsTotal.Percent(cpuStats.Total() * float64(time.Second))
		}
		stats[strconv.Itoa(pid)] = &cstructs.ResourceUsage{MemoryStats: ms, CpuStats: cs}
	}

	return stats, nil
}

// configureTaskDir sets the task dir in the executor
func (e *UniversalExecutor) configureTaskDir() error {
	taskDir, ok := e.ctx.AllocDir.TaskDirs[e.ctx.Task.Name]
	e.taskDir = taskDir
	if !ok {
		return fmt.Errorf("couldn't find task directory for task %v", e.ctx.Task.Name)
	}
	e.cmd.Dir = taskDir
	return nil
}

// lookupBin looks for path to the binary to run by looking for the binary in
// the following locations, in-order: task/local/, task/, based on host $PATH.
// The return path is absolute.
func (e *UniversalExecutor) lookupBin(bin string) (string, error) {
	// Check in the local directory
	local := filepath.Join(e.taskDir, allocdir.TaskLocal, bin)
	if _, err := os.Stat(local); err == nil {
		return local, nil
	}

	// Check at the root of the task's directory
	root := filepath.Join(e.taskDir, bin)
	if _, err := os.Stat(root); err == nil {
		return root, nil
	}

	// Check the $PATH
	if host, err := exec.LookPath(bin); err == nil {
		return host, nil
	}

	return "", fmt.Errorf("binary %q could not be found", bin)
}

// makeExecutable makes the given file executable for root,group,others.
func (e *UniversalExecutor) makeExecutable(binPath string) error {
	if runtime.GOOS == "windows" {
		return nil
	}

	fi, err := os.Stat(binPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("binary %q does not exist", binPath)
		}
		return fmt.Errorf("specified binary is invalid: %v", err)
	}

	// If it is not executable, make it so.
	perm := fi.Mode().Perm()
	req := os.FileMode(0555)
	if perm&req != req {
		if err := os.Chmod(binPath, perm|req); err != nil {
			return fmt.Errorf("error making %q executable: %s", binPath, err)
		}
	}
	return nil
}

// getFreePort returns a free port ready to be listened on between upper and
// lower bounds
func (e *UniversalExecutor) getListener(lowerBound uint, upperBound uint) (net.Listener, error) {
	if runtime.GOOS == "windows" {
		return e.listenerTCP(lowerBound, upperBound)
	}

	return e.listenerUnix()
}

// listenerTCP creates a TCP listener using an unused port between an upper and
// lower bound
func (e *UniversalExecutor) listenerTCP(lowerBound uint, upperBound uint) (net.Listener, error) {
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
func (e *UniversalExecutor) listenerUnix() (net.Listener, error) {
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

// createCheckMap creates a map of checks that the executor will handle on it's
// own
func (e *UniversalExecutor) createCheckMap() map[string]struct{} {
	checks := map[string]struct{}{
		"script": struct{}{},
	}
	return checks
}

// createCheck creates NomadCheck from a ServiceCheck
func (e *UniversalExecutor) createCheck(check *structs.ServiceCheck, checkID string) (consul.Check, error) {
	if check.Type == structs.ServiceCheckScript && e.ctx.Driver == "docker" {
		return &DockerScriptCheck{
			id:          checkID,
			interval:    check.Interval,
			timeout:     check.Timeout,
			containerID: e.consulCtx.ContainerID,
			logger:      e.logger,
			cmd:         check.Command,
			args:        check.Args,
		}, nil
	}

	if check.Type == structs.ServiceCheckScript && (e.ctx.Driver == "exec" ||
		e.ctx.Driver == "raw_exec" || e.ctx.Driver == "java") {
		return &ExecScriptCheck{
			id:          checkID,
			interval:    check.Interval,
			timeout:     check.Timeout,
			cmd:         check.Command,
			args:        check.Args,
			taskDir:     e.taskDir,
			FSIsolation: e.command.FSIsolation,
		}, nil

	}
	return nil, fmt.Errorf("couldn't create check for %v", check.Name)
}

// interpolateServices interpolates tags in a service and checks with values from the
// task's environment.
func (e *UniversalExecutor) interpolateServices(task *structs.Task) {
	e.ctx.TaskEnv.Build()
	for _, service := range task.Services {
		for _, check := range service.Checks {
			if check.Type == structs.ServiceCheckScript {
				check.Name = e.ctx.TaskEnv.ReplaceEnv(check.Name)
				check.Command = e.ctx.TaskEnv.ReplaceEnv(check.Command)
				check.Args = e.ctx.TaskEnv.ParseAndReplace(check.Args)
				check.Path = e.ctx.TaskEnv.ReplaceEnv(check.Path)
				check.Protocol = e.ctx.TaskEnv.ReplaceEnv(check.Protocol)
			}
		}
		service.Name = e.ctx.TaskEnv.ReplaceEnv(service.Name)
		service.Tags = e.ctx.TaskEnv.ParseAndReplace(service.Tags)
	}
}

// collectPids collects the pids of the child processes that the executor is
// running every 5 seconds
func (e *UniversalExecutor) collectPids() {
	// Fire the timer right away when the executor starts from there on the pids
	// are collected every scan interval
	timer := time.NewTimer(0)
	defer timer.Stop()
	for {
		select {
		case <-timer.C:
			pids, err := e.getAllPids()
			if err != nil {
				e.logger.Printf("[DEBUG] executor: error collecting pids: %v", err)
			}
			e.pidLock.Lock()

			// Adding pids which are not being tracked
			for pid, np := range pids {
				if _, ok := e.pids[pid]; !ok {
					e.pids[pid] = np
				}
			}
			// Removing pids which are no longer present
			for pid := range e.pids {
				if _, ok := pids[pid]; !ok {
					delete(e.pids, pid)
				}
			}
			e.pidLock.Unlock()
			timer.Reset(pidScanInterval)
		case <-e.processExited:
			return
		}
	}
}

// scanPids scans all the pids on the machine running the current executor and
// returns the child processes of the executor.
func (e *UniversalExecutor) scanPids(parentPid int, allPids []ps.Process) (map[int]*nomadPid, error) {
	processFamily := make(map[int]struct{})
	processFamily[parentPid] = struct{}{}

	// A buffer for holding pids which haven't matched with any parent pid
	var pidsRemaining []ps.Process
	for {
		// flag to indicate if we have found a match
		foundNewPid := false

		for _, pid := range allPids {
			_, childPid := processFamily[pid.PPid()]

			// checking if the pid is a child of any of the parents
			if childPid {
				processFamily[pid.Pid()] = struct{}{}
				foundNewPid = true
			} else {
				// if it is not, then we add the pid to the buffer
				pidsRemaining = append(pidsRemaining, pid)
			}
			// scan only the pids which are left in the buffer
			allPids = pidsRemaining
		}

		// not scanning anymore if we couldn't find a single match
		if !foundNewPid {
			break
		}
	}
	res := make(map[int]*nomadPid)
	for pid := range processFamily {
		np := nomadPid{
			pid:           pid,
			cpuStatsTotal: stats.NewCpuStats(),
			cpuStatsUser:  stats.NewCpuStats(),
			cpuStatsSys:   stats.NewCpuStats(),
		}
		res[pid] = &np
	}
	return res, nil
}

// aggregatedResourceUsage aggregates the resource usage of all the pids and
// returns a TaskResourceUsage data point
func (e *UniversalExecutor) aggregatedResourceUsage(pidStats map[string]*cstructs.ResourceUsage) *cstructs.TaskResourceUsage {
	ts := time.Now().UTC().UnixNano()
	var (
		systemModeCPU, userModeCPU, percent float64
		totalRSS, totalSwap                 uint64
	)

	for _, pidStat := range pidStats {
		systemModeCPU += pidStat.CpuStats.SystemMode
		userModeCPU += pidStat.CpuStats.UserMode
		percent += pidStat.CpuStats.Percent

		totalRSS += pidStat.MemoryStats.RSS
		totalSwap += pidStat.MemoryStats.Swap
	}

	totalCPU := &cstructs.CpuStats{
		SystemMode: systemModeCPU,
		UserMode:   userModeCPU,
		Percent:    percent,
		Measured:   ExecutorBasicMeasuredCpuStats,
		TotalTicks: e.systemCpuStats.TicksConsumed(percent),
	}

	totalMemory := &cstructs.MemoryStats{
		RSS:      totalRSS,
		Swap:     totalSwap,
		Measured: ExecutorBasicMeasuredMemStats,
	}

	resourceUsage := cstructs.ResourceUsage{
		MemoryStats: totalMemory,
		CpuStats:    totalCPU,
	}
	return &cstructs.TaskResourceUsage{
		ResourceUsage: &resourceUsage,
		Timestamp:     ts,
		Pids:          pidStats,
	}
}
