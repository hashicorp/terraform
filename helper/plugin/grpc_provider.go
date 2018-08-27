package plugin

import (
	"encoding/json"
	"errors"
	"strconv"

	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/msgpack"
	context "golang.org/x/net/context"

	"github.com/hashicorp/terraform/config/hcl2shim"
	"github.com/hashicorp/terraform/configs/configschema"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/plugin/convert"
	"github.com/hashicorp/terraform/plugin/proto"
	"github.com/hashicorp/terraform/terraform"
)

// NewGRPCProviderServerShim wraps a terraform.ResourceProvider in a
// proto.ProviderServer implementation. If the provided provider is not a
// *schema.Provider, this will return nil,
func NewGRPCProviderServerShim(p terraform.ResourceProvider) *GRPCProviderServer {
	sp, ok := p.(*schema.Provider)
	if !ok {
		return nil
	}

	return &GRPCProviderServer{
		provider: sp,
	}
}

// GRPCProviderServer handles the server, or plugin side of the rpc connection.
type GRPCProviderServer struct {
	provider *schema.Provider
}

func (s *GRPCProviderServer) GetSchema(_ context.Context, req *proto.GetProviderSchema_Request) (*proto.GetProviderSchema_Response, error) {
	resp := &proto.GetProviderSchema_Response{
		ResourceSchemas:   make(map[string]*proto.Schema),
		DataSourceSchemas: make(map[string]*proto.Schema),
	}

	resp.Provider = &proto.Schema{
		Block: convert.ConfigSchemaToProto(s.getProviderSchemaBlock()),
	}

	for typ, res := range s.provider.ResourcesMap {
		resp.ResourceSchemas[typ] = &proto.Schema{
			Version: int64(res.SchemaVersion),
			Block:   convert.ConfigSchemaToProto(res.CoreConfigSchema()),
		}
	}

	for typ, dat := range s.provider.DataSourcesMap {
		resp.DataSourceSchemas[typ] = &proto.Schema{
			Version: int64(dat.SchemaVersion),
			Block:   convert.ConfigSchemaToProto(dat.CoreConfigSchema()),
		}
	}

	return resp, nil
}

func (s *GRPCProviderServer) getProviderSchemaBlock() *configschema.Block {
	return schema.InternalMap(s.provider.Schema).CoreConfigSchema()
}

func (s *GRPCProviderServer) getResourceSchemaBlock(name string) *configschema.Block {
	res := s.provider.ResourcesMap[name]
	return res.CoreConfigSchema()
}

func (s *GRPCProviderServer) getDatasourceSchemaBlock(name string) *configschema.Block {
	dat := s.provider.DataSourcesMap[name]
	return dat.CoreConfigSchema()
}

func (s *GRPCProviderServer) ValidateProviderConfig(_ context.Context, req *proto.ValidateProviderConfig_Request) (*proto.ValidateProviderConfig_Response, error) {
	resp := &proto.ValidateProviderConfig_Response{}

	block := s.getProviderSchemaBlock()

	configVal, err := msgpack.Unmarshal(req.Config.Msgpack, block.ImpliedType())
	if err != nil {
		resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, err)
		return resp, nil
	}

	config := terraform.NewResourceConfigShimmed(configVal, block)

	warns, errs := s.provider.Validate(config)
	resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, convert.WarnsAndErrsToProto(warns, errs))

	return resp, nil
}

func (s *GRPCProviderServer) ValidateResourceTypeConfig(_ context.Context, req *proto.ValidateResourceTypeConfig_Request) (*proto.ValidateResourceTypeConfig_Response, error) {
	resp := &proto.ValidateResourceTypeConfig_Response{}

	block := s.getResourceSchemaBlock(req.TypeName)

	configVal, err := msgpack.Unmarshal(req.Config.Msgpack, block.ImpliedType())
	if err != nil {
		resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, err)
		return resp, nil
	}

	config := terraform.NewResourceConfigShimmed(configVal, block)

	warns, errs := s.provider.ValidateResource(req.TypeName, config)
	resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, convert.WarnsAndErrsToProto(warns, errs))

	return resp, nil
}

func (s *GRPCProviderServer) ValidateDataSourceConfig(_ context.Context, req *proto.ValidateDataSourceConfig_Request) (*proto.ValidateDataSourceConfig_Response, error) {
	resp := &proto.ValidateDataSourceConfig_Response{}

	block := s.getDatasourceSchemaBlock(req.TypeName)

	configVal, err := msgpack.Unmarshal(req.Config.Msgpack, block.ImpliedType())
	if err != nil {
		resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, err)
		return resp, nil
	}

	config := terraform.NewResourceConfigShimmed(configVal, block)

	warns, errs := s.provider.ValidateResource(req.TypeName, config)
	resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, convert.WarnsAndErrsToProto(warns, errs))

	return resp, nil
}

func (s *GRPCProviderServer) UpgradeResourceState(_ context.Context, req *proto.UpgradeResourceState_Request) (*proto.UpgradeResourceState_Response, error) {
	resp := &proto.UpgradeResourceState_Response{}

	res := s.provider.ResourcesMap[req.TypeName]
	block := res.CoreConfigSchema()

	version := int(req.Version)

	var jsonMap map[string]interface{}
	var err error

	// if there's a JSON state, we need to decode it.
	if req.RawState.Json != nil {
		err = json.Unmarshal(req.RawState.Json, &jsonMap)
		if err != nil {
			resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, err)
			return resp, nil
		}
	}

	// We first need to upgrade a flatmap state if it exists.
	// There should never be both a JSON and Flatmap state in the request.
	if req.RawState.Flatmap != nil {
		jsonMap, version, err = s.upgradeFlatmapState(version, req.RawState.Flatmap, res)
		if err != nil {
			resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, err)
			return resp, nil
		}
	}

	// complete the upgrade of the JSON states
	jsonMap, err = s.upgradeJSONState(version, jsonMap, res)
	if err != nil {
		resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, err)
		return resp, nil
	}

	// now we need to turn the state into the default json representation, so
	// that it can be re-decoded using the actual schema.
	val, err := schema.JSONMapToStateValue(jsonMap, block)
	if err != nil {
		resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, err)
		return resp, nil
	}

	// encode the final state to the expected msgpack format
	newStateMP, err := msgpack.Marshal(val, block.ImpliedType())
	if err != nil {
		resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, err)
		return resp, nil
	}

	resp.UpgradedState = &proto.DynamicValue{Msgpack: newStateMP}
	return resp, nil
}

// upgradeFlatmapState takes a legacy flatmap state, upgrades it using Migrate
// state if necessary, and converts it to the new JSON state format decoded as a
// map[string]interface{}.
// upgradeFlatmapState returns the json map along with the corresponding schema
// version.
func (s *GRPCProviderServer) upgradeFlatmapState(version int, m map[string]string, res *schema.Resource) (map[string]interface{}, int, error) {
	// this will be the version we've upgraded so, defaulting to the given
	// version in case no migration was called.
	upgradedVersion := version

	// first determine if we need to call the legacy MigrateState func
	requiresMigrate := version < res.SchemaVersion

	schemaType := res.CoreConfigSchema().ImpliedType()

	// if there are any StateUpgraders, then we need to only compare
	// against the first version there
	if len(res.StateUpgraders) > 0 {
		requiresMigrate = version < res.StateUpgraders[0].Version
	}

	if requiresMigrate {
		if res.MigrateState == nil {
			return nil, 0, errors.New("cannot upgrade state, missing MigrateState function")
		}

		is := &terraform.InstanceState{
			ID:         m["id"],
			Attributes: m,
			Meta: map[string]interface{}{
				"schema_version": strconv.Itoa(version),
			},
		}

		is, err := res.MigrateState(version, is, s.provider.Meta())
		if err != nil {
			return nil, 0, err
		}

		// re-assign the map in case there was a copy made, making sure to keep
		// the ID
		m := is.Attributes
		m["id"] = is.ID

		// if there are further upgraders, then we've only updated that far
		if len(res.StateUpgraders) > 0 {
			schemaType = res.StateUpgraders[0].Type
			upgradedVersion = res.StateUpgraders[0].Version
		}
	} else {
		// the schema version may be newer than the MigrateState functions
		// handled and older than the current, but still stored in the flatmap
		// form. If that's the case, we need to find the correct schema type to
		// convert the state.
		for _, upgrader := range res.StateUpgraders {
			if upgrader.Version == version {
				schemaType = upgrader.Type
				break
			}
		}
	}

	// now we know the state is up to the latest version that handled the
	// flatmap format state. Now we can upgrade the format and continue from
	// there.
	newConfigVal, err := hcl2shim.HCL2ValueFromFlatmap(m, schemaType)
	if err != nil {
		return nil, 0, err
	}

	jsonMap, err := schema.StateValueToJSONMap(newConfigVal, schemaType)
	return jsonMap, upgradedVersion, err
}

func (s *GRPCProviderServer) upgradeJSONState(version int, m map[string]interface{}, res *schema.Resource) (map[string]interface{}, error) {
	var err error

	for _, upgrader := range res.StateUpgraders {
		if version != upgrader.Version {
			continue
		}

		m, err = upgrader.Upgrade(m, s.provider.Meta())
		if err != nil {
			return nil, err
		}
		version++
	}

	return m, nil
}

func (s *GRPCProviderServer) Stop(_ context.Context, _ *proto.Stop_Request) (*proto.Stop_Response, error) {
	resp := &proto.Stop_Response{}

	err := s.provider.Stop()
	if err != nil {
		resp.Error = err.Error()
	}

	return resp, nil
}

func (s *GRPCProviderServer) Configure(_ context.Context, req *proto.Configure_Request) (*proto.Configure_Response, error) {
	resp := &proto.Configure_Response{}

	block := s.getProviderSchemaBlock()

	configVal, err := msgpack.Unmarshal(req.Config.Msgpack, block.ImpliedType())
	if err != nil {
		resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, err)
		return resp, nil
	}

	config := terraform.NewResourceConfigShimmed(configVal, block)
	err = s.provider.Configure(config)
	resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, err)

	return resp, nil
}

func (s *GRPCProviderServer) ReadResource(_ context.Context, req *proto.ReadResource_Request) (*proto.ReadResource_Response, error) {
	resp := &proto.ReadResource_Response{}

	res := s.provider.ResourcesMap[req.TypeName]
	block := res.CoreConfigSchema()

	stateVal, err := msgpack.Unmarshal(req.CurrentState.Msgpack, block.ImpliedType())
	if err != nil {
		resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, err)
		return resp, nil
	}

	instanceState := schema.InstanceStateFromStateValue(stateVal, res.SchemaVersion)

	newInstanceState, err := res.RefreshWithoutUpgrade(instanceState, s.provider.Meta())
	if err != nil {
		resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, err)
		return resp, nil
	}

	// helper/schema should always copy the ID over, but do it again just to be safe
	newInstanceState.Attributes["id"] = newInstanceState.ID

	newConfigVal, err := hcl2shim.HCL2ValueFromFlatmap(newInstanceState.Attributes, block.ImpliedType())
	if err != nil {
		resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, err)
		return resp, nil
	}

	newConfigMP, err := msgpack.Marshal(newConfigVal, block.ImpliedType())
	if err != nil {
		resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, err)
		return resp, nil
	}

	resp.NewState = &proto.DynamicValue{
		Msgpack: newConfigMP,
	}

	return resp, nil
}

func (s *GRPCProviderServer) PlanResourceChange(_ context.Context, req *proto.PlanResourceChange_Request) (*proto.PlanResourceChange_Response, error) {
	resp := &proto.PlanResourceChange_Response{}

	res := s.provider.ResourcesMap[req.TypeName]
	block := res.CoreConfigSchema()

	priorStateVal, err := msgpack.Unmarshal(req.PriorState.Msgpack, block.ImpliedType())
	if err != nil {
		resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, err)
		return resp, nil
	}

	proposedNewStateVal, err := msgpack.Unmarshal(req.ProposedNewState.Msgpack, block.ImpliedType())
	if err != nil {
		resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, err)
		return resp, nil
	}

	info := &terraform.InstanceInfo{
		Type: req.TypeName,
	}

	priorState := schema.InstanceStateFromStateValue(priorStateVal, res.SchemaVersion)

	// turn the propsed state into a legacy configuration
	config := terraform.NewResourceConfigShimmed(proposedNewStateVal, block)

	diff, err := s.provider.Diff(info, priorState, config)
	if err != nil {
		resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, err)
		return resp, nil
	}

	// now we need to apply the diff to the prior state, so get the planned state
	plannedStateVal, err := schema.ApplyDiff(priorStateVal, diff, block)
	if err != nil {
		resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, err)
		return resp, nil
	}

	plannedMP, err := msgpack.Marshal(plannedStateVal, block.ImpliedType())
	if err != nil {
		resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, err)
		return resp, nil
	}
	resp.PlannedState = &proto.DynamicValue{
		Msgpack: plannedMP,
	}

	// the Meta field gets encoded into PlannedPrivate
	plannedPrivate, err := json.Marshal(diff.Meta)
	if err != nil {
		resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, err)
		return resp, nil
	}
	resp.PlannedPrivate = plannedPrivate

	// collect the attributes that require instance replacement, and convert
	// them to cty.Paths.
	var requiresNew []string
	for attr, d := range diff.Attributes {
		if d.RequiresNew {
			requiresNew = append(requiresNew, attr)
		}
	}

	requiresReplace, err := hcl2shim.RequiresReplace(requiresNew, block.ImpliedType())
	if err != nil {
		resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, err)
		return resp, nil
	}

	// convert these to the protocol structures
	for _, p := range requiresReplace {
		resp.RequiresReplace = append(resp.RequiresReplace, pathToAttributePath(p))
	}

	return resp, nil
}

func (s *GRPCProviderServer) ApplyResourceChange(_ context.Context, req *proto.ApplyResourceChange_Request) (*proto.ApplyResourceChange_Response, error) {
	resp := &proto.ApplyResourceChange_Response{}

	res := s.provider.ResourcesMap[req.TypeName]
	block := res.CoreConfigSchema()

	priorStateVal, err := msgpack.Unmarshal(req.PriorState.Msgpack, block.ImpliedType())
	if err != nil {
		resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, err)
		return resp, nil
	}

	plannedStateVal, err := msgpack.Unmarshal(req.PlannedState.Msgpack, block.ImpliedType())
	if err != nil {
		resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, err)
		return resp, nil
	}

	info := &terraform.InstanceInfo{
		Type: req.TypeName,
	}

	priorState := schema.InstanceStateFromStateValue(priorStateVal, res.SchemaVersion)

	var private map[string]interface{}
	if err := json.Unmarshal(req.PlannedPrivate, &private); err != nil {
		resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, err)
		return resp, nil
	}

	diff, err := schema.DiffFromValues(priorStateVal, plannedStateVal, res)
	if err != nil {
		resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, err)
		return resp, nil
	}

	newInstanceState, err := s.provider.Apply(info, priorState, diff)
	if err != nil {
		resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, err)
		return resp, nil
	}

	newStateVal, err := schema.StateValueFromInstanceState(newInstanceState, block.ImpliedType())
	if err != nil {
		resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, err)
		return resp, nil
	}

	newStateMP, err := msgpack.Marshal(newStateVal, block.ImpliedType())
	if err != nil {
		resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, err)
		return resp, nil
	}
	resp.NewState = &proto.DynamicValue{
		Msgpack: newStateMP,
	}

	meta, err := json.Marshal(newInstanceState.Meta)
	if err != nil {
		resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, err)
		return resp, nil
	}
	resp.Private = meta

	return resp, nil
}

func (s *GRPCProviderServer) ImportResourceState(_ context.Context, req *proto.ImportResourceState_Request) (*proto.ImportResourceState_Response, error) {
	resp := &proto.ImportResourceState_Response{}

	block := s.getResourceSchemaBlock(req.TypeName)

	info := &terraform.InstanceInfo{
		Type: req.TypeName,
	}

	newInstanceStates, err := s.provider.ImportState(info, req.Id)
	if err != nil {
		resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, err)
		return resp, nil
	}

	for _, is := range newInstanceStates {
		// copy the ID again just to be sure it wasn't missed
		is.Attributes["id"] = is.ID

		newStateVal, err := hcl2shim.HCL2ValueFromFlatmap(is.Attributes, block.ImpliedType())
		if err != nil {
			resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, err)
			return resp, nil
		}

		newStateMP, err := msgpack.Marshal(newStateVal, block.ImpliedType())
		if err != nil {
			resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, err)
			return resp, nil
		}

		meta, err := json.Marshal(is.Meta)
		if err != nil {
			resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, err)
			return resp, nil
		}

		// the legacy implementation could only import one type at a time
		importedResource := &proto.ImportResourceState_ImportedResource{
			TypeName: req.TypeName,
			State: &proto.DynamicValue{
				Msgpack: newStateMP,
			},
			Private: meta,
		}

		resp.ImportedResources = append(resp.ImportedResources, importedResource)
	}

	return resp, nil
}

func (s *GRPCProviderServer) ReadDataSource(_ context.Context, req *proto.ReadDataSource_Request) (*proto.ReadDataSource_Response, error) {
	resp := &proto.ReadDataSource_Response{}

	res := s.provider.DataSourcesMap[req.TypeName]
	block := res.CoreConfigSchema()

	configVal, err := msgpack.Unmarshal(req.Config.Msgpack, block.ImpliedType())
	if err != nil {
		resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, err)
		return resp, nil
	}

	info := &terraform.InstanceInfo{
		Type: req.TypeName,
	}

	config := terraform.NewResourceConfigShimmed(configVal, block)

	// we need to still build the diff separately with the Read method to match
	// the old behavior
	diff, err := s.provider.ReadDataDiff(info, config)
	if err != nil {
		resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, err)
		return resp, nil
	}

	// now we can get the new complete data source
	newInstanceState, err := s.provider.ReadDataApply(info, diff)
	if err != nil {
		resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, err)
		return resp, nil
	}

	newStateVal, err := schema.StateValueFromInstanceState(newInstanceState, block.ImpliedType())
	if err != nil {
		resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, err)
		return resp, nil
	}

	newStateMP, err := msgpack.Marshal(newStateVal, block.ImpliedType())
	if err != nil {
		resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, err)
		return resp, nil
	}
	resp.State.Msgpack = newStateMP
	return resp, nil
}

func pathToAttributePath(path cty.Path) *proto.AttributePath {
	var steps []*proto.AttributePath_Step

	for _, step := range path {
		switch s := step.(type) {
		case cty.GetAttrStep:
			steps = append(steps, &proto.AttributePath_Step{
				Selector: &proto.AttributePath_Step_AttributeName{
					AttributeName: s.Name,
				},
			})
		case cty.IndexStep:
			ty := s.Key.Type()
			switch ty {
			case cty.Number:
				i, _ := s.Key.AsBigFloat().Int64()
				steps = append(steps, &proto.AttributePath_Step{
					Selector: &proto.AttributePath_Step_ElementKeyInt{
						ElementKeyInt: i,
					},
				})
			case cty.String:
				steps = append(steps, &proto.AttributePath_Step{
					Selector: &proto.AttributePath_Step_ElementKeyString{
						ElementKeyString: s.Key.AsString(),
					},
				})
			}
		}
	}

	return &proto.AttributePath{Steps: steps}
}
