package rpcapi

import (
	"bytes"
	"context"
	"fmt"
	"sync"

	"go.rpcplugin.org/rpcplugin"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/configs/configload"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/plans/planfile"
	"github.com/hashicorp/terraform/internal/rpcapi/tfcore1"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/states/statefile"
	"github.com/hashicorp/terraform/internal/terraform"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

type tfcore1PluginServer struct {
	core          *terraform.Context
	cwd           string
	cwdModulesDir string

	configs      map[uint64]*configs.Config
	configSnaps  map[uint64]*configload.Snapshot
	lastConfigID uint64
	configsMu    sync.Mutex

	plans       map[uint64]*plans.Plan
	planConfigs map[uint64]*configs.Config
	planSnaps   map[uint64]*configload.Snapshot
	lastPlanID  uint64
	plansMu     sync.Mutex
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
		resp.Diagnostics = diagnosticsToProto(diags)
		return resp, nil
	}

	config, snap, hclDiags := loader.LoadConfigWithSnapshot(s.cwd)
	diags = diags.Append(hclDiags)
	if hclDiags.HasErrors() {
		resp.Diagnostics = diagnosticsToProto(diags)
		return resp, nil
	}

	s.configs[newConfigID] = config
	s.configSnaps[newConfigID] = snap
	s.lastConfigID = newConfigID

	resp.ConfigId = newConfigID
	resp.Diagnostics = diagnosticsToProto(diags) // might still have warnings

	return resp, nil
}

func (s *tfcore1PluginServer) CloseConfig(ctx context.Context, req *tfcore1.CloseConfig_Request) (*tfcore1.CloseConfig_Response, error) {
	s.configsMu.Lock()
	defer s.configsMu.Unlock()

	if _, exists := s.configs[req.ConfigId]; !exists {
		return nil, status.Errorf(codes.NotFound, "no open configuration has id %d", req.ConfigId)
	}

	delete(s.configs, req.ConfigId)
	delete(s.configSnaps, req.ConfigId)

	return &tfcore1.CloseConfig_Response{}, nil
}

func (s *tfcore1PluginServer) ValidateConfig(ctx context.Context, req *tfcore1.ValidateConfig_Request) (*tfcore1.ValidateConfig_Response, error) {
	config := s.getOpenConfig(req.ConfigId)
	if config == nil {
		return nil, status.Errorf(codes.NotFound, "no open configuration has id %d", req.ConfigId)
	}

	diags := s.core.Validate(config)
	return &tfcore1.ValidateConfig_Response{
		Diagnostics: diagnosticsToProto(diags),
	}, nil
}

func (s *tfcore1PluginServer) CreatePlan(ctx context.Context, req *tfcore1.CreatePlan_Request) (*tfcore1.CreatePlan_Response, error) {
	var diags tfdiags.Diagnostics

	config := s.getOpenConfig(req.ConfigId)
	if config == nil {
		return nil, status.Errorf(codes.NotFound, "no open configuration has id %d", req.ConfigId)
	}
	configSnap := s.getOpenConfigSnapshot(req.ConfigId)
	if configSnap == nil {
		// Might get here if someone calls CloseConfig between our getOpenConfig
		// and getOpenConfigSnapshot calls.
		return nil, status.Errorf(codes.NotFound, "no open configuration has id %d", req.ConfigId)
	}

	var prevRunState *states.State
	if len(req.PrevRunState) != 0 {
		stateReader := bytes.NewReader(req.PrevRunState)
		stateFile, err := statefile.Read(stateReader)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid previous run state snapshot: %s", err)
		}
		prevRunState = stateFile.State
	} else {
		prevRunState = states.NewState()
	}

	opts, err := planOptsFromProto(req.Options)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid plan options: %s", err)
	}

	s.plansMu.Lock()
	defer s.plansMu.Unlock()
	newPlanID, err := s.nextPlanID()
	if err != nil {
		return nil, err
	}

	plan, diags := s.core.Plan(config, prevRunState, opts)
	if diags.HasErrors() {
		return &tfcore1.CreatePlan_Response{
			Diagnostics: diagnosticsToProto(diags),
		}, nil
	}

	// Because of how we're kinda-misuing the plan file format from
	// Terraform CLI as our export format, we need to put a stub backend
	// configuration in the plan to make the plan serializable. We won't
	// actually make any use of this when we reload the plan.
	plan.Backend = plans.Backend{
		Type:      "none",
		Config:    plans.DynamicValue{0}, // intentionally invalid
		Workspace: "default",
	}

	s.plans[newPlanID] = plan
	s.planConfigs[newPlanID] = config
	s.planSnaps[newPlanID] = configSnap
	s.lastPlanID = newPlanID

	protoOutputValues := make(map[string]*tfcore1.DynamicValue)
	for _, ovcs := range plan.Changes.Outputs {
		if !ovcs.Addr.Module.IsRoot() {
			continue
		}
		name := ovcs.Addr.OutputValue.Name

		// This is a bit silly: we decode the encoded output value only to
		// immediately re-encode it in what happens to be almost exactly
		// the same way. But it's what works with the abstractions we have
		// today.
		ovc, err := ovcs.Decode()
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to decode output value %q: %s", name, err)
		}

		protoOutputValues[name], err = dynamicValueToProto(ovc.After)
		if ovc.Sensitive {
			// The static sensitive flag supersedes any dynamically-detected one,
			// if set to true.
			protoOutputValues[name].Sensitive = true
		}
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to encode output value %q: %s", name, err)
		}
	}

	return &tfcore1.CreatePlan_Response{
		PlanId:              newPlanID,
		PlannedOutputValues: protoOutputValues,
		Diagnostics:         diagnosticsToProto(diags),
	}, nil
}

func (s *tfcore1PluginServer) DiscardPlan(ctx context.Context, req *tfcore1.DiscardPlan_Request) (*tfcore1.DiscardPlan_Response, error) {
	s.configsMu.Lock()
	defer s.configsMu.Unlock()

	if _, exists := s.plans[req.PlanId]; !exists {
		return nil, status.Errorf(codes.NotFound, "no open plan has id %d", req.PlanId)
	}

	delete(s.plans, req.PlanId)
	delete(s.planConfigs, req.PlanId)
	delete(s.planSnaps, req.PlanId)

	return &tfcore1.DiscardPlan_Response{}, nil
}

func (s *tfcore1PluginServer) ExportPlan(ctx context.Context, req *tfcore1.ExportPlan_Request) (*tfcore1.ExportPlan_Response, error) {
	plan := s.getOpenPlan(req.PlanId)
	if plan == nil {
		return nil, status.Errorf(codes.NotFound, "no open plan has id %d", req.PlanId)
	}
	configSnap := s.getOpenPlanConfigSnapshot(req.PlanId)
	if configSnap == nil {
		return nil, status.Errorf(codes.NotFound, "no open plan has id %d", req.PlanId)
	}

	// We're lightly misusing the statefile package here because we're really
	// just serializing the state alone and letting the caller be the one
	// to worry about file-level details like lineage and serial.
	prevRunStateFile := statefile.New(plan.PrevRunState, "", 0)
	priorStateFile := statefile.New(plan.PriorState, "", 0)

	var buf bytes.Buffer

	err := planfile.Write(planfile.CreateArgs{
		Plan:                 plan,
		ConfigSnapshot:       configSnap,
		PreviousRunStateFile: prevRunStateFile,
		StateFile:            priorStateFile,
		// NOTE: Intentionally no DependencyLocks here, because dependency
		// versions are not Terraform Core's concern. The caller is responsible
		// for managing those in whatever way is appropriate for its needs.
	}, &buf)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to serialize plan: %s", err)
	}

	return &tfcore1.ExportPlan_Response{
		RawPlan: buf.Bytes(),
	}, nil
}

func (s *tfcore1PluginServer) ImportPlan(ctx context.Context, req *tfcore1.ImportPlan_Request) (*tfcore1.ImportPlan_Response, error) {
	buf := bytes.NewReader(req.RawPlan)

	r, err := planfile.OpenReader(buf, int64(len(req.RawPlan)))
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid serialized plan file: %s", err)
	}
	defer r.Close()

	plan, err := r.ReadPlan()
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid serialized plan content: %s", err)
	}

	config, diags := r.ReadConfig()
	if diags.HasErrors() {
		return nil, status.Errorf(codes.InvalidArgument, "invalid configuration in plan: %s", err)
	}

	// NOTE: This is a bit silly because ReadConfig above already read the
	// config snapshot and parsed it for us. Maybe in future we'll extend
	// planfile.Reader with a method that can return both the snapshot and
	// the resulting configuration at the same time.
	configSnap, err := r.ReadConfigSnapshot()
	if diags.HasErrors() {
		return nil, status.Errorf(codes.InvalidArgument, "invalid configuration in plan: %s", err)
	}

	s.plansMu.Lock()
	defer s.plansMu.Unlock()
	newPlanID, err := s.nextPlanID()
	if err != nil {
		return nil, err
	}

	s.plans[newPlanID] = plan
	s.planConfigs[newPlanID] = config
	s.planSnaps[newPlanID] = configSnap
	s.lastPlanID = newPlanID

	return &tfcore1.ImportPlan_Response{
		PlanId: newPlanID,
	}, nil
}

func (s *tfcore1PluginServer) ApplyPlan(ctx context.Context, in *tfcore1.ApplyPlan_Request) (*tfcore1.ApplyPlan_Response, error) {
	return nil, status.Error(codes.Unimplemented, "not yet implemented")
}

func (s *tfcore1PluginServer) getOpenConfig(id uint64) *configs.Config {
	s.configsMu.Lock()
	ret := s.configs[id]
	s.configsMu.Unlock()
	return ret
}

func (s *tfcore1PluginServer) getOpenConfigSnapshot(id uint64) *configload.Snapshot {
	s.configsMu.Lock()
	ret := s.configSnaps[id]
	s.configsMu.Unlock()
	return ret
}

func (s *tfcore1PluginServer) getOpenPlan(id uint64) *plans.Plan {
	s.configsMu.Lock()
	ret := s.plans[id]
	s.configsMu.Unlock()
	return ret
}

func (s *tfcore1PluginServer) getOpenPlanConfig(id uint64) *configs.Config {
	s.configsMu.Lock()
	ret := s.planConfigs[id]
	s.configsMu.Unlock()
	return ret
}

func (s *tfcore1PluginServer) getOpenPlanConfigSnapshot(id uint64) *configload.Snapshot {
	s.configsMu.Lock()
	ret := s.planSnaps[id]
	s.configsMu.Unlock()
	return ret
}

// call nextPlanID only while already holding s.plansMu, and then keep holding
// s.plansMu until updating s.lastPlanID to record having used the allocated
// ID.
func (s *tfcore1PluginServer) nextPlanID() (uint64, error) {
	startPlanID := s.lastPlanID
	newPlanID := s.lastPlanID + 1
	for ; newPlanID != 0 && s.plans[newPlanID] != nil; newPlanID++ {
		if newPlanID == startPlanID {
			// wrap around, so we've exhausted all the ids somehow! This should
			// never happen in any reasonable use of this API.
			return 0, status.Error(codes.ResourceExhausted, "no free plan handles")
		}
	}
	return newPlanID, nil
}

func newV1PluginServer(core *terraform.Context, cwd string, cwdModulesDir string) tfcore1.TerraformServer {
	return &tfcore1PluginServer{
		core:          core,
		cwd:           cwd,
		cwdModulesDir: cwdModulesDir,
		configs:       map[uint64]*configs.Config{},
		configSnaps:   map[uint64]*configload.Snapshot{},
		plans:         map[uint64]*plans.Plan{},
		planConfigs:   map[uint64]*configs.Config{},
		planSnaps:     map[uint64]*configload.Snapshot{},
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
