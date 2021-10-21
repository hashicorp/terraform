package rpcapi

import (
	"context"
	"fmt"
	"sync"

	"go.rpcplugin.org/rpcplugin"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/configs/configload"
	"github.com/hashicorp/terraform/internal/rpcapi/tfcore1"
	"github.com/hashicorp/terraform/internal/terraform"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

type tfcore1PluginServer struct {
	core          *terraform.Context
	cwd           string
	cwdModulesDir string

	configs      map[uint64]*configs.Config
	lastConfigID uint64
	configsMu    sync.Mutex
}

var _ tfcore1.TerraformServer = (*tfcore1PluginServer)(nil)

func (s *tfcore1PluginServer) OpenConfigCwd(ctx context.Context, req *tfcore1.OpenConfigCwd_Request) (*tfcore1.OpenConfigCwd_Response, error) {
	s.configsMu.Lock()
	defer s.configsMu.Unlock()

	startConfigID := s.lastConfigID
	newConfigID := s.lastConfigID + 1
	for ; newConfigID != 0 && s.configs[newConfigID] != nil; newConfigID++ {
		if newConfigID == startConfigID {
			// wrap around, so we've exhausted all the ids somehow! This should
			// never happen in any reasonable use of this API.
			return nil, status.Error(codes.ResourceExhausted, "no free configuration handles")
		}
	}

	var diags tfdiags.Diagnostics
	resp := &tfcore1.OpenConfigCwd_Response{}

	loader, err := configload.NewLoader(&configload.Config{
		ModulesDir: s.cwdModulesDir,
	})
	if err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Uninitialized Working Directory",
			fmt.Sprintf("Can't load configuration from this directory: %s.", err),
		))
		resp.Diagnostics = protoDiagnotics(diags)
		return resp, nil
	}

	config, hclDiags := loader.LoadConfig(".")
	diags = diags.Append(hclDiags)
	if hclDiags.HasErrors() {
		resp.Diagnostics = protoDiagnotics(diags)
		return resp, nil
	}

	s.configs[newConfigID] = config
	s.lastConfigID = newConfigID

	resp.ConfigId = newConfigID
	resp.Diagnostics = protoDiagnotics(diags) // might still have warnings

	return resp, nil
}

func (s *tfcore1PluginServer) CloseConfig(ctx context.Context, req *tfcore1.CloseConfig_Request) (*tfcore1.CloseConfig_Response, error) {
	s.configsMu.Lock()
	defer s.configsMu.Unlock()

	if _, exists := s.configs[req.ConfigId]; !exists {
		return nil, status.Errorf(codes.NotFound, "no open configuration has id %d", req.ConfigId)
	}

	delete(s.configs, req.ConfigId)

	return &tfcore1.CloseConfig_Response{}, nil
}

func (s *tfcore1PluginServer) getOpenConfig(id uint64) *configs.Config {
	s.configsMu.Lock()
	ret := s.configs[id]
	s.configsMu.Unlock()
	return ret
}

func newV1PluginServer(core *terraform.Context, cwd string, cwdModulesDir string) tfcore1.TerraformServer {
	return &tfcore1PluginServer{
		core:          core,
		cwd:           cwd,
		cwdModulesDir: cwdModulesDir,
		configs:       map[uint64]*configs.Config{},
	}
}

type version1 struct {
	getCoreOpts   func() *terraform.ContextOpts
	cwd           string
	cwdModulesDir string
}

var _ rpcplugin.ServerVersion = version1{}

func (p version1) RegisterServer(server *grpc.Server) error {
	coreOpts := p.getCoreOpts()
	core, diags := terraform.NewContext(coreOpts)
	if diags.HasErrors() {
		return fmt.Errorf("failed to instantiate Terraform Core: %w", diags.Err())
	}

	tfcore1.RegisterTerraformServer(server, newV1PluginServer(core, p.cwd, p.cwdModulesDir))
	return nil
}
