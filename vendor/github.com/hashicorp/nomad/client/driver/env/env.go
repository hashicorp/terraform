package env

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	hargs "github.com/hashicorp/nomad/helper/args"
	"github.com/hashicorp/nomad/nomad/structs"
)

// A set of environment variables that are exported by each driver.
const (
	// AllocDir is the environment variable with the path to the alloc directory
	// that is shared across tasks within a task group.
	AllocDir = "NOMAD_ALLOC_DIR"

	// TaskLocalDir is the environment variable with the path to the tasks local
	// directory where it can store data that is persisted to the alloc is
	// removed.
	TaskLocalDir = "NOMAD_TASK_DIR"

	// MemLimit is the environment variable with the tasks memory limit in MBs.
	MemLimit = "NOMAD_MEMORY_LIMIT"

	// CpuLimit is the environment variable with the tasks CPU limit in MHz.
	CpuLimit = "NOMAD_CPU_LIMIT"

	// AllocID is the environment variable for passing the allocation ID.
	AllocID = "NOMAD_ALLOC_ID"

	// AllocName is the environment variable for passing the allocation name.
	AllocName = "NOMAD_ALLOC_NAME"

	// TaskName is the environment variable for passing the task name.
	TaskName = "NOMAD_TASK_NAME"

	// AllocIndex is the environment variable for passing the allocation index.
	AllocIndex = "NOMAD_ALLOC_INDEX"

	// AddrPrefix is the prefix for passing both dynamic and static port
	// allocations to tasks.
	// E.g$NOMAD_ADDR_http=127.0.0.1:80
	AddrPrefix = "NOMAD_ADDR_"

	// IpPrefix is the prefix for passing the IP of a port allocation to a task.
	IpPrefix = "NOMAD_IP_"

	// PortPrefix is the prefix for passing the port allocation to a task.
	PortPrefix = "NOMAD_PORT_"

	// HostPortPrefix is the prefix for passing the host port when a portmap is
	// specified.
	HostPortPrefix = "NOMAD_HOST_PORT_"

	// MetaPrefix is the prefix for passing task meta data.
	MetaPrefix = "NOMAD_META_"
)

// The node values that can be interpreted.
const (
	nodeIdKey    = "node.unique.id"
	nodeDcKey    = "node.datacenter"
	nodeNameKey  = "node.unique.name"
	nodeClassKey = "node.class"

	// Prefixes used for lookups.
	nodeAttributePrefix = "attr."
	nodeMetaPrefix      = "meta."
)

// TaskEnvironment is used to expose information to a task via environment
// variables and provide interpolation of Nomad variables.
type TaskEnvironment struct {
	Env           map[string]string
	TaskMeta      map[string]string
	TaskGroupMeta map[string]string
	JobMeta       map[string]string
	AllocDir      string
	TaskDir       string
	CpuLimit      int
	MemLimit      int
	TaskName      string
	AllocIndex    int
	AllocId       string
	AllocName     string
	Node          *structs.Node
	Networks      []*structs.NetworkResource
	PortMap       map[string]int

	// taskEnv is the variables that will be set in the tasks environment
	TaskEnv map[string]string

	// nodeValues is the values that are allowed for interprolation from the
	// node.
	NodeValues map[string]string
}

func NewTaskEnvironment(node *structs.Node) *TaskEnvironment {
	return &TaskEnvironment{Node: node, AllocIndex: -1}
}

// ParseAndReplace takes the user supplied args replaces any instance of an
// environment variable or nomad variable in the args with the actual value.
func (t *TaskEnvironment) ParseAndReplace(args []string) []string {
	replaced := make([]string, len(args))
	for i, arg := range args {
		replaced[i] = hargs.ReplaceEnv(arg, t.TaskEnv, t.NodeValues)
	}

	return replaced
}

// ReplaceEnv takes an arg and replaces all occurrences of environment variables
// and nomad variables.  If the variable is found in the passed map it is
// replaced, otherwise the original string is returned.
func (t *TaskEnvironment) ReplaceEnv(arg string) string {
	return hargs.ReplaceEnv(arg, t.TaskEnv, t.NodeValues)
}

// Build must be called after all the tasks environment values have been set.
func (t *TaskEnvironment) Build() *TaskEnvironment {
	t.NodeValues = make(map[string]string)
	t.TaskEnv = make(map[string]string)

	// Build the meta with the following precedence: task, task group, job.
	for _, meta := range []map[string]string{t.JobMeta, t.TaskGroupMeta, t.TaskMeta} {
		for k, v := range meta {
			t.TaskEnv[fmt.Sprintf("%s%s", MetaPrefix, strings.ToUpper(k))] = v
		}
	}

	// Build the ports
	for _, network := range t.Networks {
		for label, value := range network.MapLabelToValues(nil) {
			t.TaskEnv[fmt.Sprintf("%s%s", IpPrefix, label)] = network.IP
			t.TaskEnv[fmt.Sprintf("%s%s", HostPortPrefix, label)] = strconv.Itoa(value)
			if forwardedPort, ok := t.PortMap[label]; ok {
				value = forwardedPort
			}
			t.TaskEnv[fmt.Sprintf("%s%s", PortPrefix, label)] = fmt.Sprintf("%d", value)
			IPPort := fmt.Sprintf("%s:%d", network.IP, value)
			t.TaskEnv[fmt.Sprintf("%s%s", AddrPrefix, label)] = IPPort

		}
	}

	// Build the directories
	if t.AllocDir != "" {
		t.TaskEnv[AllocDir] = t.AllocDir
	}
	if t.TaskDir != "" {
		t.TaskEnv[TaskLocalDir] = t.TaskDir
	}

	// Build the resource limits
	if t.MemLimit != 0 {
		t.TaskEnv[MemLimit] = strconv.Itoa(t.MemLimit)
	}
	if t.CpuLimit != 0 {
		t.TaskEnv[CpuLimit] = strconv.Itoa(t.CpuLimit)
	}

	// Build the tasks ids
	if t.AllocId != "" {
		t.TaskEnv[AllocID] = t.AllocId
	}
	if t.AllocName != "" {
		t.TaskEnv[AllocName] = t.AllocName
	}
	if t.AllocIndex != -1 {
		t.TaskEnv[AllocIndex] = strconv.Itoa(t.AllocIndex)
	}
	if t.TaskName != "" {
		t.TaskEnv[TaskName] = t.TaskName
	}

	// Build the node
	if t.Node != nil {
		// Set up the node values.
		t.NodeValues[nodeIdKey] = t.Node.ID
		t.NodeValues[nodeDcKey] = t.Node.Datacenter
		t.NodeValues[nodeNameKey] = t.Node.Name
		t.NodeValues[nodeClassKey] = t.Node.NodeClass

		// Set up the attributes.
		for k, v := range t.Node.Attributes {
			t.NodeValues[fmt.Sprintf("%s%s", nodeAttributePrefix, k)] = v
		}

		// Set up the meta.
		for k, v := range t.Node.Meta {
			t.NodeValues[fmt.Sprintf("%s%s", nodeMetaPrefix, k)] = v
		}
	}

	// Interpret the environment variables
	interpreted := make(map[string]string, len(t.Env))
	for k, v := range t.Env {
		interpreted[k] = hargs.ReplaceEnv(v, t.NodeValues, t.TaskEnv)
	}

	for k, v := range interpreted {
		t.TaskEnv[k] = v
	}

	return t
}

// EnvList returns a list of strings with NAME=value pairs.
func (t *TaskEnvironment) EnvList() []string {
	env := []string{}
	for k, v := range t.TaskEnv {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}

	return env
}

// EnvMap returns a copy of the tasks environment variables.
func (t *TaskEnvironment) EnvMap() map[string]string {
	m := make(map[string]string, len(t.TaskEnv))
	for k, v := range t.TaskEnv {
		m[k] = v
	}

	return m
}

// Builder methods to build the TaskEnvironment
func (t *TaskEnvironment) SetAllocDir(dir string) *TaskEnvironment {
	t.AllocDir = dir
	return t
}

func (t *TaskEnvironment) ClearAllocDir() *TaskEnvironment {
	t.AllocDir = ""
	return t
}

func (t *TaskEnvironment) SetTaskLocalDir(dir string) *TaskEnvironment {
	t.TaskDir = dir
	return t
}

func (t *TaskEnvironment) ClearTaskLocalDir() *TaskEnvironment {
	t.TaskDir = ""
	return t
}

func (t *TaskEnvironment) SetMemLimit(limit int) *TaskEnvironment {
	t.MemLimit = limit
	return t
}

func (t *TaskEnvironment) ClearMemLimit() *TaskEnvironment {
	t.MemLimit = 0
	return t
}

func (t *TaskEnvironment) SetCpuLimit(limit int) *TaskEnvironment {
	t.CpuLimit = limit
	return t
}

func (t *TaskEnvironment) ClearCpuLimit() *TaskEnvironment {
	t.CpuLimit = 0
	return t
}

func (t *TaskEnvironment) SetNetworks(networks []*structs.NetworkResource) *TaskEnvironment {
	t.Networks = networks
	return t
}

func (t *TaskEnvironment) clearNetworks() *TaskEnvironment {
	t.Networks = nil
	return t
}

func (t *TaskEnvironment) SetPortMap(portMap map[string]int) *TaskEnvironment {
	t.PortMap = portMap
	return t
}

func (t *TaskEnvironment) clearPortMap() *TaskEnvironment {
	t.PortMap = nil
	return t
}

// Takes a map of meta values to be passed to the task. The keys are capatilized
// when the environent variable is set.
func (t *TaskEnvironment) SetTaskMeta(m map[string]string) *TaskEnvironment {
	t.TaskMeta = m
	return t
}

func (t *TaskEnvironment) ClearTaskMeta() *TaskEnvironment {
	t.TaskMeta = nil
	return t
}

func (t *TaskEnvironment) SetTaskGroupMeta(m map[string]string) *TaskEnvironment {
	t.TaskGroupMeta = m
	return t
}

func (t *TaskEnvironment) ClearTaskGroupMeta() *TaskEnvironment {
	t.TaskGroupMeta = nil
	return t
}

func (t *TaskEnvironment) SetJobMeta(m map[string]string) *TaskEnvironment {
	t.JobMeta = m
	return t
}

func (t *TaskEnvironment) ClearJobMeta() *TaskEnvironment {
	t.JobMeta = nil
	return t
}

func (t *TaskEnvironment) SetEnvvars(m map[string]string) *TaskEnvironment {
	t.Env = m
	return t
}

// Appends the given environment variables.
func (t *TaskEnvironment) AppendEnvvars(m map[string]string) *TaskEnvironment {
	if t.Env == nil {
		t.Env = make(map[string]string, len(m))
	}

	for k, v := range m {
		t.Env[k] = v
	}
	return t
}

// AppendHostEnvvars adds the host environment variables to the tasks. The
// filter parameter can be use to filter host environment from entering the
// tasks.
func (t *TaskEnvironment) AppendHostEnvvars(filter []string) *TaskEnvironment {
	hostEnv := os.Environ()
	if t.Env == nil {
		t.Env = make(map[string]string, len(hostEnv))
	}

	// Index the filtered environment variables.
	index := make(map[string]struct{}, len(filter))
	for _, f := range filter {
		index[f] = struct{}{}
	}

	for _, e := range hostEnv {
		parts := strings.SplitN(e, "=", 2)
		key, value := parts[0], parts[1]

		// Skip filtered environment variables
		if _, filtered := index[key]; filtered {
			continue
		}

		// Don't override the tasks environment variables.
		if _, existing := t.Env[key]; !existing {
			t.Env[key] = value
		}
	}

	return t
}

func (t *TaskEnvironment) ClearEnvvars() *TaskEnvironment {
	t.Env = nil
	return t
}

// Helper method for setting all fields from an allocation.
func (t *TaskEnvironment) SetAlloc(alloc *structs.Allocation) *TaskEnvironment {
	t.AllocId = alloc.ID
	t.AllocName = alloc.Name
	t.AllocIndex = alloc.Index()
	return t
}

// Helper method for clearing all fields from an allocation.
func (t *TaskEnvironment) ClearAlloc(alloc *structs.Allocation) *TaskEnvironment {
	return t.ClearAllocId().ClearAllocName().ClearAllocIndex()
}

func (t *TaskEnvironment) SetAllocIndex(index int) *TaskEnvironment {
	t.AllocIndex = index
	return t
}

func (t *TaskEnvironment) ClearAllocIndex() *TaskEnvironment {
	t.AllocIndex = -1
	return t
}

func (t *TaskEnvironment) SetAllocId(id string) *TaskEnvironment {
	t.AllocId = id
	return t
}

func (t *TaskEnvironment) ClearAllocId() *TaskEnvironment {
	t.AllocId = ""
	return t
}

func (t *TaskEnvironment) SetAllocName(name string) *TaskEnvironment {
	t.AllocName = name
	return t
}

func (t *TaskEnvironment) ClearAllocName() *TaskEnvironment {
	t.AllocName = ""
	return t
}

func (t *TaskEnvironment) SetTaskName(name string) *TaskEnvironment {
	t.TaskName = name
	return t
}

func (t *TaskEnvironment) ClearTaskName() *TaskEnvironment {
	t.TaskName = ""
	return t
}
