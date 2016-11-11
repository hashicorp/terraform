package driver

import (
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/go-plugin"
	"github.com/hashicorp/nomad/client/allocdir"
	"github.com/hashicorp/nomad/client/config"
	"github.com/hashicorp/nomad/client/driver/executor"
	dstructs "github.com/hashicorp/nomad/client/driver/structs"
	cstructs "github.com/hashicorp/nomad/client/structs"
	"github.com/hashicorp/nomad/helper/discover"
	"github.com/hashicorp/nomad/helper/fields"
	"github.com/hashicorp/nomad/nomad/structs"
	"github.com/mitchellh/mapstructure"
)

const (
	// The key populated in Node Attributes to indicate the presence of the Exec
	// driver
	execDriverAttr = "driver.exec"
)

// ExecDriver fork/execs tasks using as many of the underlying OS's isolation
// features.
type ExecDriver struct {
	DriverContext
}

type ExecDriverConfig struct {
	Command string   `mapstructure:"command"`
	Args    []string `mapstructure:"args"`
}

// execHandle is returned from Start/Open as a handle to the PID
type execHandle struct {
	pluginClient    *plugin.Client
	executor        executor.Executor
	isolationConfig *dstructs.IsolationConfig
	userPid         int
	allocDir        *allocdir.AllocDir
	killTimeout     time.Duration
	maxKillTimeout  time.Duration
	logger          *log.Logger
	waitCh          chan *dstructs.WaitResult
	doneCh          chan struct{}
	version         string
}

// NewExecDriver is used to create a new exec driver
func NewExecDriver(ctx *DriverContext) Driver {
	return &ExecDriver{DriverContext: *ctx}
}

// Validate is used to validate the driver configuration
func (d *ExecDriver) Validate(config map[string]interface{}) error {
	fd := &fields.FieldData{
		Raw: config,
		Schema: map[string]*fields.FieldSchema{
			"command": &fields.FieldSchema{
				Type:     fields.TypeString,
				Required: true,
			},
			"args": &fields.FieldSchema{
				Type: fields.TypeArray,
			},
		},
	}

	if err := fd.Validate(); err != nil {
		return err
	}

	return nil
}

func (d *ExecDriver) Periodic() (bool, time.Duration) {
	return true, 15 * time.Second
}

func (d *ExecDriver) Start(ctx *ExecContext, task *structs.Task) (DriverHandle, error) {
	var driverConfig ExecDriverConfig
	if err := mapstructure.WeakDecode(task.Config, &driverConfig); err != nil {
		return nil, err
	}

	// Get the command to be ran
	command := driverConfig.Command
	if err := validateCommand(command, "args"); err != nil {
		return nil, err
	}

	// Set the host environment variables.
	filter := strings.Split(d.config.ReadDefault("env.blacklist", config.DefaultEnvBlacklist), ",")
	d.taskEnv.AppendHostEnvvars(filter)

	// Get the task directory for storing the executor logs.
	taskDir, ok := ctx.AllocDir.TaskDirs[d.DriverContext.taskName]
	if !ok {
		return nil, fmt.Errorf("Could not find task directory for task: %v", d.DriverContext.taskName)
	}

	bin, err := discover.NomadExecutable()
	if err != nil {
		return nil, fmt.Errorf("unable to find the nomad binary: %v", err)
	}
	pluginLogFile := filepath.Join(taskDir, fmt.Sprintf("%s-executor.out", task.Name))
	pluginConfig := &plugin.ClientConfig{
		Cmd: exec.Command(bin, "executor", pluginLogFile),
	}

	exec, pluginClient, err := createExecutor(pluginConfig, d.config.LogOutput, d.config)
	if err != nil {
		return nil, err
	}
	executorCtx := &executor.ExecutorContext{
		TaskEnv:   d.taskEnv,
		Driver:    "exec",
		AllocDir:  ctx.AllocDir,
		AllocID:   ctx.AllocID,
		ChrootEnv: d.config.ChrootEnv,
		Task:      task,
	}

	ps, err := exec.LaunchCmd(&executor.ExecCommand{
		Cmd:            command,
		Args:           driverConfig.Args,
		FSIsolation:    true,
		ResourceLimits: true,
		User:           getExecutorUser(task),
	}, executorCtx)
	if err != nil {
		pluginClient.Kill()
		return nil, err
	}
	d.logger.Printf("[DEBUG] driver.exec: started process via plugin with pid: %v", ps.Pid)

	// Return a driver handle
	maxKill := d.DriverContext.config.MaxKillTimeout
	h := &execHandle{
		pluginClient:    pluginClient,
		userPid:         ps.Pid,
		executor:        exec,
		allocDir:        ctx.AllocDir,
		isolationConfig: ps.IsolationConfig,
		killTimeout:     GetKillTimeout(task.KillTimeout, maxKill),
		maxKillTimeout:  maxKill,
		logger:          d.logger,
		version:         d.config.Version,
		doneCh:          make(chan struct{}),
		waitCh:          make(chan *dstructs.WaitResult, 1),
	}
	if err := exec.SyncServices(consulContext(d.config, "")); err != nil {
		d.logger.Printf("[ERR] driver.exec: error registering services with consul for task: %q: %v", task.Name, err)
	}
	go h.run()
	return h, nil
}

type execId struct {
	Version         string
	KillTimeout     time.Duration
	MaxKillTimeout  time.Duration
	UserPid         int
	TaskDir         string
	AllocDir        *allocdir.AllocDir
	IsolationConfig *dstructs.IsolationConfig
	PluginConfig    *PluginReattachConfig
}

func (d *ExecDriver) Open(ctx *ExecContext, handleID string) (DriverHandle, error) {
	id := &execId{}
	if err := json.Unmarshal([]byte(handleID), id); err != nil {
		return nil, fmt.Errorf("Failed to parse handle '%s': %v", handleID, err)
	}

	pluginConfig := &plugin.ClientConfig{
		Reattach: id.PluginConfig.PluginConfig(),
	}
	exec, client, err := createExecutor(pluginConfig, d.config.LogOutput, d.config)
	if err != nil {
		merrs := new(multierror.Error)
		merrs.Errors = append(merrs.Errors, err)
		d.logger.Println("[ERR] driver.exec: error connecting to plugin so destroying plugin pid and user pid")
		if e := destroyPlugin(id.PluginConfig.Pid, id.UserPid); e != nil {
			merrs.Errors = append(merrs.Errors, fmt.Errorf("error destroying plugin and userpid: %v", e))
		}
		if id.IsolationConfig != nil {
			ePid := pluginConfig.Reattach.Pid
			if e := executor.ClientCleanup(id.IsolationConfig, ePid); e != nil {
				merrs.Errors = append(merrs.Errors, fmt.Errorf("destroying cgroup failed: %v", e))
			}
		}
		if e := ctx.AllocDir.UnmountAll(); e != nil {
			merrs.Errors = append(merrs.Errors, e)
		}
		return nil, fmt.Errorf("error connecting to plugin: %v", merrs.ErrorOrNil())
	}

	ver, _ := exec.Version()
	d.logger.Printf("[DEBUG] driver.exec : version of executor: %v", ver.Version)
	// Return a driver handle
	h := &execHandle{
		pluginClient:    client,
		executor:        exec,
		userPid:         id.UserPid,
		allocDir:        id.AllocDir,
		isolationConfig: id.IsolationConfig,
		logger:          d.logger,
		version:         id.Version,
		killTimeout:     id.KillTimeout,
		maxKillTimeout:  id.MaxKillTimeout,
		doneCh:          make(chan struct{}),
		waitCh:          make(chan *dstructs.WaitResult, 1),
	}
	if err := exec.SyncServices(consulContext(d.config, "")); err != nil {
		d.logger.Printf("[ERR] driver.exec: error registering services with consul: %v", err)
	}
	go h.run()
	return h, nil
}

func (h *execHandle) ID() string {
	id := execId{
		Version:         h.version,
		KillTimeout:     h.killTimeout,
		MaxKillTimeout:  h.maxKillTimeout,
		PluginConfig:    NewPluginReattachConfig(h.pluginClient.ReattachConfig()),
		UserPid:         h.userPid,
		AllocDir:        h.allocDir,
		IsolationConfig: h.isolationConfig,
	}

	data, err := json.Marshal(id)
	if err != nil {
		h.logger.Printf("[ERR] driver.exec: failed to marshal ID to JSON: %s", err)
	}
	return string(data)
}

func (h *execHandle) WaitCh() chan *dstructs.WaitResult {
	return h.waitCh
}

func (h *execHandle) Update(task *structs.Task) error {
	// Store the updated kill timeout.
	h.killTimeout = GetKillTimeout(task.KillTimeout, h.maxKillTimeout)
	h.executor.UpdateTask(task)

	// Update is not possible
	return nil
}

func (h *execHandle) Kill() error {
	if err := h.executor.ShutDown(); err != nil {
		if h.pluginClient.Exited() {
			return nil
		}
		return fmt.Errorf("executor Shutdown failed: %v", err)
	}

	select {
	case <-h.doneCh:
		return nil
	case <-time.After(h.killTimeout):
		if h.pluginClient.Exited() {
			return nil
		}
		if err := h.executor.Exit(); err != nil {
			return fmt.Errorf("executor Exit failed: %v", err)
		}

		return nil
	}
}

func (h *execHandle) Stats() (*cstructs.TaskResourceUsage, error) {
	return h.executor.Stats()
}

func (h *execHandle) run() {
	ps, err := h.executor.Wait()
	close(h.doneCh)

	// If the exitcode is 0 and we had an error that means the plugin didn't
	// connect and doesn't know the state of the user process so we are killing
	// the user process so that when we create a new executor on restarting the
	// new user process doesn't have collisions with resources that the older
	// user pid might be holding onto.
	if ps.ExitCode == 0 && err != nil {
		if h.isolationConfig != nil {
			ePid := h.pluginClient.ReattachConfig().Pid
			if e := executor.ClientCleanup(h.isolationConfig, ePid); e != nil {
				h.logger.Printf("[ERR] driver.exec: destroying resource container failed: %v", e)
			}
		}
		if e := h.allocDir.UnmountAll(); e != nil {
			h.logger.Printf("[ERR] driver.exec: unmounting dev,proc and alloc dirs failed: %v", e)
		}
	}
	h.waitCh <- dstructs.NewWaitResult(ps.ExitCode, ps.Signal, err)
	close(h.waitCh)
	// Remove services
	if err := h.executor.DeregisterServices(); err != nil {
		h.logger.Printf("[ERR] driver.exec: failed to deregister services: %v", err)
	}

	if err := h.executor.Exit(); err != nil {
		h.logger.Printf("[ERR] driver.exec: error destroying executor: %v", err)
	}
	h.pluginClient.Kill()
}
