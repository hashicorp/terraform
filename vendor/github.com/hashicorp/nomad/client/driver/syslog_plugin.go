package driver

import (
	"log"
	"net/rpc"

	"github.com/hashicorp/go-plugin"
	"github.com/hashicorp/nomad/client/driver/logging"
	"github.com/hashicorp/nomad/nomad/structs"
)

type SyslogCollectorRPC struct {
	client *rpc.Client
}

type LaunchCollectorArgs struct {
	Ctx *logging.LogCollectorContext
}

func (e *SyslogCollectorRPC) LaunchCollector(ctx *logging.LogCollectorContext) (*logging.SyslogCollectorState, error) {
	var ss *logging.SyslogCollectorState
	err := e.client.Call("Plugin.LaunchCollector", LaunchCollectorArgs{Ctx: ctx}, &ss)
	return ss, err
}

func (e *SyslogCollectorRPC) Exit() error {
	return e.client.Call("Plugin.Exit", new(interface{}), new(interface{}))
}

func (e *SyslogCollectorRPC) UpdateLogConfig(logConfig *structs.LogConfig) error {
	return e.client.Call("Plugin.UpdateLogConfig", logConfig, new(interface{}))
}

type SyslogCollectorRPCServer struct {
	Impl logging.LogCollector
}

func (s *SyslogCollectorRPCServer) LaunchCollector(args LaunchCollectorArgs,
	resp *logging.SyslogCollectorState) error {
	ss, err := s.Impl.LaunchCollector(args.Ctx)
	if ss != nil {
		*resp = *ss
	}
	return err
}

func (s *SyslogCollectorRPCServer) Exit(args interface{}, resp *interface{}) error {
	return s.Impl.Exit()
}

func (s *SyslogCollectorRPCServer) UpdateLogConfig(logConfig *structs.LogConfig, resp *interface{}) error {
	return s.Impl.UpdateLogConfig(logConfig)
}

type SyslogCollectorPlugin struct {
	logger *log.Logger
	Impl   *SyslogCollectorRPCServer
}

func (p *SyslogCollectorPlugin) Server(*plugin.MuxBroker) (interface{}, error) {
	if p.Impl == nil {
		p.Impl = &SyslogCollectorRPCServer{Impl: logging.NewSyslogCollector(p.logger)}
	}
	return p.Impl, nil
}

func (p *SyslogCollectorPlugin) Client(b *plugin.MuxBroker, c *rpc.Client) (interface{}, error) {
	return &SyslogCollectorRPC{client: c}, nil
}
