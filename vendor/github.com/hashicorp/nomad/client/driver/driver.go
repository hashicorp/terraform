package driver

import (
	"fmt"
	"log"
	"path/filepath"
	"sync"

	"github.com/hashicorp/nomad/client/allocdir"
	"github.com/hashicorp/nomad/client/config"
	"github.com/hashicorp/nomad/client/driver/env"
	"github.com/hashicorp/nomad/client/fingerprint"
	"github.com/hashicorp/nomad/nomad/structs"

	dstructs "github.com/hashicorp/nomad/client/driver/structs"
	cstructs "github.com/hashicorp/nomad/client/structs"
)

// BuiltinDrivers contains the built in registered drivers
// which are available for allocation handling
var BuiltinDrivers = map[string]Factory{
	"docker":   NewDockerDriver,
	"exec":     NewExecDriver,
	"raw_exec": NewRawExecDriver,
	"java":     NewJavaDriver,
	"qemu":     NewQemuDriver,
	"rkt":      NewRktDriver,
}

// NewDriver is used to instantiate and return a new driver
// given the name and a logger
func NewDriver(name string, ctx *DriverContext) (Driver, error) {
	// Lookup the factory function
	factory, ok := BuiltinDrivers[name]
	if !ok {
		return nil, fmt.Errorf("unknown driver '%s'", name)
	}

	// Instantiate the driver
	f := factory(ctx)
	return f, nil
}

// Factory is used to instantiate a new Driver
type Factory func(*DriverContext) Driver

// Driver is used for execution of tasks. This allows Nomad
// to support many pluggable implementations of task drivers.
// Examples could include LXC, Docker, Qemu, etc.
type Driver interface {
	// Drivers must support the fingerprint interface for detection
	fingerprint.Fingerprint

	// Start is used to being task execution
	Start(ctx *ExecContext, task *structs.Task) (DriverHandle, error)

	// Open is used to re-open a handle to a task
	Open(ctx *ExecContext, handleID string) (DriverHandle, error)

	// Drivers must validate their configuration
	Validate(map[string]interface{}) error
}

// DriverContext is a means to inject dependencies such as loggers, configs, and
// node attributes into a Driver without having to change the Driver interface
// each time we do it. Used in conjection with Factory, above.
type DriverContext struct {
	taskName string
	config   *config.Config
	logger   *log.Logger
	node     *structs.Node
	taskEnv  *env.TaskEnvironment
}

// NewEmptyDriverContext returns a DriverContext with all fields set to their
// zero value.
func NewEmptyDriverContext() *DriverContext {
	return &DriverContext{
		taskName: "",
		config:   nil,
		node:     nil,
		logger:   nil,
		taskEnv:  nil,
	}
}

// NewDriverContext initializes a new DriverContext with the specified fields.
// This enables other packages to create DriverContexts but keeps the fields
// private to the driver. If we want to change this later we can gorename all of
// the fields in DriverContext.
func NewDriverContext(taskName string, config *config.Config, node *structs.Node,
	logger *log.Logger, taskEnv *env.TaskEnvironment) *DriverContext {
	return &DriverContext{
		taskName: taskName,
		config:   config,
		node:     node,
		logger:   logger,
		taskEnv:  taskEnv,
	}
}

// DriverHandle is an opaque handle into a driver used for task
// manipulation
type DriverHandle interface {
	// Returns an opaque handle that can be used to re-open the handle
	ID() string

	// WaitCh is used to return a channel used wait for task completion
	WaitCh() chan *dstructs.WaitResult

	// Update is used to update the task if possible and update task related
	// configurations.
	Update(task *structs.Task) error

	// Kill is used to stop the task
	Kill() error

	// Stats returns aggregated stats of the driver
	Stats() (*cstructs.TaskResourceUsage, error)
}

// ExecContext is shared between drivers within an allocation
type ExecContext struct {
	sync.Mutex

	// AllocDir contains information about the alloc directory structure.
	AllocDir *allocdir.AllocDir

	// Alloc ID
	AllocID string
}

// NewExecContext is used to create a new execution context
func NewExecContext(alloc *allocdir.AllocDir, allocID string) *ExecContext {
	return &ExecContext{AllocDir: alloc, AllocID: allocID}
}

// GetTaskEnv converts the alloc dir, the node, task and alloc into a
// TaskEnvironment.
func GetTaskEnv(allocDir *allocdir.AllocDir, node *structs.Node,
	task *structs.Task, alloc *structs.Allocation) (*env.TaskEnvironment, error) {

	tg := alloc.Job.LookupTaskGroup(alloc.TaskGroup)
	env := env.NewTaskEnvironment(node).
		SetTaskMeta(task.Meta).
		SetTaskGroupMeta(tg.Meta).
		SetJobMeta(alloc.Job.Meta).
		SetEnvvars(task.Env).
		SetTaskName(task.Name)

	if allocDir != nil {
		env.SetAllocDir(allocDir.SharedDir)
		taskdir, ok := allocDir.TaskDirs[task.Name]
		if !ok {
			return nil, fmt.Errorf("failed to get task directory for task %q", task.Name)
		}

		env.SetTaskLocalDir(filepath.Join(taskdir, allocdir.TaskLocal))
	}

	if task.Resources != nil {
		env.SetMemLimit(task.Resources.MemoryMB).
			SetCpuLimit(task.Resources.CPU).
			SetNetworks(task.Resources.Networks)
	}

	if alloc != nil {
		env.SetAlloc(alloc)
	}

	return env.Build(), nil
}

func mapMergeStrInt(maps ...map[string]int) map[string]int {
	out := map[string]int{}
	for _, in := range maps {
		for key, val := range in {
			out[key] = val
		}
	}
	return out
}

func mapMergeStrStr(maps ...map[string]string) map[string]string {
	out := map[string]string{}
	for _, in := range maps {
		for key, val := range in {
			out[key] = val
		}
	}
	return out
}
