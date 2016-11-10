package driver

import (
	"encoding/gob"
	"log"
	"net/rpc"

	"github.com/hashicorp/go-plugin"
	"github.com/hashicorp/nomad/client/driver/executor"
	cstructs "github.com/hashicorp/nomad/client/structs"
	"github.com/hashicorp/nomad/nomad/structs"
)

// Registering these types since we have to serialize and de-serialize the Task
// structs over the wire between drivers and the executor.
func init() {
	gob.Register([]interface{}{})
	gob.Register(map[string]interface{}{})
	gob.Register([]map[string]string{})
	gob.Register([]map[string]int{})
}

type ExecutorRPC struct {
	client *rpc.Client
	logger *log.Logger
}

// LaunchCmdArgs wraps a user command and the args for the purposes of RPC
type LaunchCmdArgs struct {
	Cmd *executor.ExecCommand
	Ctx *executor.ExecutorContext
}

// LaunchSyslogServerArgs wraps the executor context for the purposes of RPC
type LaunchSyslogServerArgs struct {
	Ctx *executor.ExecutorContext
}

// SyncServicesArgs wraps the consul context for the purposes of RPC
type SyncServicesArgs struct {
	Ctx *executor.ConsulContext
}

func (e *ExecutorRPC) LaunchCmd(cmd *executor.ExecCommand, ctx *executor.ExecutorContext) (*executor.ProcessState, error) {
	var ps *executor.ProcessState
	err := e.client.Call("Plugin.LaunchCmd", LaunchCmdArgs{Cmd: cmd, Ctx: ctx}, &ps)
	return ps, err
}

func (e *ExecutorRPC) LaunchSyslogServer(ctx *executor.ExecutorContext) (*executor.SyslogServerState, error) {
	var ss *executor.SyslogServerState
	err := e.client.Call("Plugin.LaunchSyslogServer", LaunchSyslogServerArgs{Ctx: ctx}, &ss)
	return ss, err
}

func (e *ExecutorRPC) Wait() (*executor.ProcessState, error) {
	var ps executor.ProcessState
	err := e.client.Call("Plugin.Wait", new(interface{}), &ps)
	return &ps, err
}

func (e *ExecutorRPC) ShutDown() error {
	return e.client.Call("Plugin.ShutDown", new(interface{}), new(interface{}))
}

func (e *ExecutorRPC) Exit() error {
	return e.client.Call("Plugin.Exit", new(interface{}), new(interface{}))
}

func (e *ExecutorRPC) UpdateLogConfig(logConfig *structs.LogConfig) error {
	return e.client.Call("Plugin.UpdateLogConfig", logConfig, new(interface{}))
}

func (e *ExecutorRPC) UpdateTask(task *structs.Task) error {
	return e.client.Call("Plugin.UpdateTask", task, new(interface{}))
}

func (e *ExecutorRPC) SyncServices(ctx *executor.ConsulContext) error {
	return e.client.Call("Plugin.SyncServices", SyncServicesArgs{Ctx: ctx}, new(interface{}))
}

func (e *ExecutorRPC) DeregisterServices() error {
	return e.client.Call("Plugin.DeregisterServices", new(interface{}), new(interface{}))
}

func (e *ExecutorRPC) Version() (*executor.ExecutorVersion, error) {
	var version executor.ExecutorVersion
	err := e.client.Call("Plugin.Version", new(interface{}), &version)
	return &version, err
}

func (e *ExecutorRPC) Stats() (*cstructs.TaskResourceUsage, error) {
	var resourceUsage cstructs.TaskResourceUsage
	err := e.client.Call("Plugin.Stats", new(interface{}), &resourceUsage)
	return &resourceUsage, err
}

type ExecutorRPCServer struct {
	Impl   executor.Executor
	logger *log.Logger
}

func (e *ExecutorRPCServer) LaunchCmd(args LaunchCmdArgs, ps *executor.ProcessState) error {
	state, err := e.Impl.LaunchCmd(args.Cmd, args.Ctx)
	if state != nil {
		*ps = *state
	}
	return err
}

func (e *ExecutorRPCServer) LaunchSyslogServer(args LaunchSyslogServerArgs, ss *executor.SyslogServerState) error {
	state, err := e.Impl.LaunchSyslogServer(args.Ctx)
	if state != nil {
		*ss = *state
	}
	return err
}

func (e *ExecutorRPCServer) Wait(args interface{}, ps *executor.ProcessState) error {
	state, err := e.Impl.Wait()
	if state != nil {
		*ps = *state
	}
	return err
}

func (e *ExecutorRPCServer) ShutDown(args interface{}, resp *interface{}) error {
	return e.Impl.ShutDown()
}

func (e *ExecutorRPCServer) Exit(args interface{}, resp *interface{}) error {
	return e.Impl.Exit()
}

func (e *ExecutorRPCServer) UpdateLogConfig(args *structs.LogConfig, resp *interface{}) error {
	return e.Impl.UpdateLogConfig(args)
}

func (e *ExecutorRPCServer) UpdateTask(args *structs.Task, resp *interface{}) error {
	return e.Impl.UpdateTask(args)
}

func (e *ExecutorRPCServer) SyncServices(args SyncServicesArgs, resp *interface{}) error {
	return e.Impl.SyncServices(args.Ctx)
}

func (e *ExecutorRPCServer) DeregisterServices(args interface{}, resp *interface{}) error {
	return e.Impl.DeregisterServices()
}

func (e *ExecutorRPCServer) Version(args interface{}, version *executor.ExecutorVersion) error {
	ver, err := e.Impl.Version()
	if ver != nil {
		*version = *ver
	}
	return err
}

func (e *ExecutorRPCServer) Stats(args interface{}, resourceUsage *cstructs.TaskResourceUsage) error {
	ru, err := e.Impl.Stats()
	if ru != nil {
		*resourceUsage = *ru
	}
	return err
}

type ExecutorPlugin struct {
	logger *log.Logger
	Impl   *ExecutorRPCServer
}

func (p *ExecutorPlugin) Server(*plugin.MuxBroker) (interface{}, error) {
	if p.Impl == nil {
		p.Impl = &ExecutorRPCServer{Impl: executor.NewExecutor(p.logger), logger: p.logger}
	}
	return p.Impl, nil
}

func (p *ExecutorPlugin) Client(b *plugin.MuxBroker, c *rpc.Client) (interface{}, error) {
	return &ExecutorRPC{client: c, logger: p.logger}, nil
}
