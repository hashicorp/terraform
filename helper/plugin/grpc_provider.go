package plugin

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/zclconf/go-cty/cty"
	ctyconvert "github.com/zclconf/go-cty/cty/convert"
	"github.com/zclconf/go-cty/cty/msgpack"
	context "golang.org/x/net/context"

	"github.com/hashicorp/terraform/config/hcl2shim"
	"github.com/hashicorp/terraform/configs/configschema"
	"github.com/hashicorp/terraform/helper/schema"
	proto "github.com/hashicorp/terraform/internal/tfplugin5"
	"github.com/hashicorp/terraform/plugin/convert"
	"github.com/hashicorp/terraform/terraform"
)

const newExtraKey = "_new_extra_shim"

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

func (s *GRPCProviderServer) PrepareProviderConfig(_ context.Context, req *proto.PrepareProviderConfig_Request) (*proto.PrepareProviderConfig_Response, error) {
	resp := &proto.PrepareProviderConfig_Response{}

	block := s.getProviderSchemaBlock()

	configVal, err := msgpack.Unmarshal(req.Config.Msgpack, block.ImpliedType())
	if err != nil {
		resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, err)
		return resp, nil
	}

	// lookup any required, top-level attributes that are Null, and see if we
	// have a Default value available.
	configVal, err = cty.Transform(configVal, func(path cty.Path, val cty.Value) (cty.Value, error) {
		// we're only looking for top-level attributes
		if len(path) != 1 {
			return val, nil
		}

		// nothing to do if we already have a value
		if !val.IsNull() {
			return val, nil
		}

		// get the Schema definition for this attribute
		getAttr, ok := path[0].(cty.GetAttrStep)
		// these should all exist, but just ignore anything strange
		if !ok {
			return val, nil
		}

		attrSchema := s.provider.Schema[getAttr.Name]
		// continue to ignore anything that doesn't match
		if attrSchema == nil {
			return val, nil
		}

		// this is deprecated, so don't set it
		if attrSchema.Deprecated != "" || attrSchema.Removed != "" {
			return val, nil
		}

		// find a default value if it exists
		def, err := attrSchema.DefaultValue()
		if err != nil {
			resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, fmt.Errorf("error getting default for %q: %s", getAttr.Name, err))
			return val, err
		}

		// no default
		if def == nil {
			return val, nil
		}

		// create a cty.Value and make sure it's the correct type
		tmpVal := hcl2shim.HCL2ValueFromConfigValue(def)

		// helper/schema used to allow setting "" to a bool
		if val.Type() == cty.Bool && tmpVal.RawEquals(cty.StringVal("")) {
			// return a warning about the conversion
			resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, "provider set empty string as default value for bool "+getAttr.Name)
			tmpVal = cty.False
		}

		val, err = ctyconvert.Convert(tmpVal, val.Type())
		if err != nil {
			resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, fmt.Errorf("error setting default for %q: %s", getAttr.Name, err))
		}

		return val, err
	})
	if err != nil {
		// any error here was already added to the diagnostics
		return resp, nil
	}

	configVal, err = block.CoerceValue(configVal)
	if err != nil {
		resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, err)
		return resp, nil
	}

	config := terraform.NewResourceConfigShimmed(configVal, block)

	warns, errs := s.provider.Validate(config)
	resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, convert.WarnsAndErrsToProto(warns, errs))

	preparedConfigMP, err := msgpack.Marshal(configVal, block.ImpliedType())
	if err != nil {
		resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, err)
		return resp, nil
	}

	resp.PreparedConfig = &proto.DynamicValue{Msgpack: preparedConfigMP}

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

	warns, errs := s.provider.ValidateDataSource(req.TypeName, config)
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
	if len(req.RawState.Json) > 0 {
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

	instanceState, err := res.ShimInstanceStateFromValue(stateVal)
	if err != nil {
		resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, err)
		return resp, nil
	}

	newInstanceState, err := res.RefreshWithoutUpgrade(instanceState, s.provider.Meta())
	if err != nil {
		resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, err)
		return resp, nil
	}

	if newInstanceState == nil || newInstanceState.ID == "" {
		// The old provider API used an empty id to signal that the remote
		// object appears to have been deleted, but our new protocol expects
		// to see a null value (in the cty sense) in that case.
		newStateMP, err := msgpack.Marshal(cty.NullVal(block.ImpliedType()), block.ImpliedType())
		if err != nil {
			resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, err)
		}
		resp.NewState = &proto.DynamicValue{
			Msgpack: newStateMP,
		}
		return resp, nil
	}

	// helper/schema should always copy the ID over, but do it again just to be safe
	newInstanceState.Attributes["id"] = newInstanceState.ID

	newInstanceState.Attributes = normalizeFlatmapContainers(instanceState.Attributes, newInstanceState.Attributes)

	newStateVal, err := hcl2shim.HCL2ValueFromFlatmap(newInstanceState.Attributes, block.ImpliedType())
	if err != nil {
		resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, err)
		return resp, nil
	}

	newStateVal = copyTimeoutValues(newStateVal, stateVal)

	newStateMP, err := msgpack.Marshal(newStateVal, block.ImpliedType())
	if err != nil {
		resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, err)
		return resp, nil
	}

	resp.NewState = &proto.DynamicValue{
		Msgpack: newStateMP,
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

	priorState, err := res.ShimInstanceStateFromValue(priorStateVal)
	if err != nil {
		resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, err)
		return resp, nil
	}

	priorPrivate := make(map[string]interface{})
	if len(req.PriorPrivate) > 0 {
		if err := json.Unmarshal(req.PriorPrivate, &priorPrivate); err != nil {
			resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, err)
			return resp, nil
		}
	}

	priorState.Meta = priorPrivate

	// turn the proposed state into a legacy configuration
	cfg := terraform.NewResourceConfigShimmed(proposedNewStateVal, block)

	diff, err := s.provider.SimpleDiff(info, priorState, cfg)
	if err != nil {
		resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, err)
		return resp, nil
	}

	if diff == nil {
		// schema.Provider.Diff returns nil if it ends up making a diff with no
		// changes, but our new interface wants us to return an actual change
		// description that _shows_ there are no changes. This is usually the
		// PriorSate, however if there was no prior state and no diff, then we
		// use the ProposedNewState.
		if !priorStateVal.IsNull() {
			resp.PlannedState = req.PriorState
		} else {
			resp.PlannedState = req.ProposedNewState
		}
		return resp, nil
	}

	// strip out non-diffs
	for k, v := range diff.Attributes {
		if v.New == v.Old && !v.NewComputed && !v.NewRemoved {
			delete(diff.Attributes, k)
		}
	}

	if priorState == nil {
		priorState = &terraform.InstanceState{}
	}

	// now we need to apply the diff to the prior state, so get the planned state
	plannedAttrs, err := diff.Apply(priorState.Attributes, block)

	plannedAttrs = normalizeFlatmapContainers(priorState.Attributes, plannedAttrs)

	plannedStateVal, err := hcl2shim.HCL2ValueFromFlatmap(plannedAttrs, block.ImpliedType())
	if err != nil {
		resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, err)
		return resp, nil
	}

	plannedStateVal, err = block.CoerceValue(plannedStateVal)
	if err != nil {
		resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, err)
		return resp, nil
	}
	if err != nil {
		resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, err)
		return resp, nil
	}

	plannedStateVal = copyTimeoutValues(plannedStateVal, proposedNewStateVal)

	plannedMP, err := msgpack.Marshal(plannedStateVal, block.ImpliedType())
	if err != nil {
		resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, err)
		return resp, nil
	}
	resp.PlannedState = &proto.DynamicValue{
		Msgpack: plannedMP,
	}

	// Now we need to store any NewExtra values, which are where any actual
	// StateFunc modified config fields are hidden.
	privateMap := diff.Meta
	if privateMap == nil {
		privateMap = map[string]interface{}{}
	}

	newExtra := map[string]interface{}{}

	for k, v := range diff.Attributes {
		if v.NewExtra != nil {
			newExtra[k] = v.NewExtra
		}
	}
	privateMap[newExtraKey] = newExtra

	// the Meta field gets encoded into PlannedPrivate
	plannedPrivate, err := json.Marshal(privateMap)
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

	// If anything requires a new resource already, or the "id" field indicates
	// that we will be creating a new resource, then we need to add that to
	// RequiresReplace so that core can tell if the instance is being replaced
	// even if changes are being suppressed via "ignore_changes".
	id := plannedStateVal.GetAttr("id")
	if len(requiresNew) > 0 || id.IsNull() || !id.IsKnown() {
		requiresNew = append(requiresNew, "id")
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

	priorState, err := res.ShimInstanceStateFromValue(priorStateVal)
	if err != nil {
		resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, err)
		return resp, nil
	}

	private := make(map[string]interface{})
	if len(req.PlannedPrivate) > 0 {
		if err := json.Unmarshal(req.PlannedPrivate, &private); err != nil {
			resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, err)
			return resp, nil
		}
	}

	var diff *terraform.InstanceDiff
	destroy := false

	// a null state means we are destroying the instance
	if plannedStateVal.IsNull() {
		destroy = true
		diff = &terraform.InstanceDiff{
			Attributes: make(map[string]*terraform.ResourceAttrDiff),
			Meta:       make(map[string]interface{}),
			Destroy:    true,
		}
	} else {
		diff, err = schema.DiffFromValues(priorStateVal, plannedStateVal, stripResourceModifiers(res))
		if err != nil {
			resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, err)
			return resp, nil
		}
	}

	if diff == nil {
		diff = &terraform.InstanceDiff{
			Attributes: make(map[string]*terraform.ResourceAttrDiff),
			Meta:       make(map[string]interface{}),
		}
	}

	// add NewExtra Fields that may have been stored in the private data
	if newExtra := private[newExtraKey]; newExtra != nil {
		for k, v := range newExtra.(map[string]interface{}) {
			d := diff.Attributes[k]

			if d == nil {
				d = &terraform.ResourceAttrDiff{}
			}

			d.NewExtra = v
			diff.Attributes[k] = d
		}
	}

	// strip out non-diffs
	for k, v := range diff.Attributes {
		if v.New == v.Old && !v.NewComputed && !v.NewRemoved && v.NewExtra == "" {
			delete(diff.Attributes, k)
		}
	}

	if private != nil {
		diff.Meta = private
	}

	newInstanceState, err := s.provider.Apply(info, priorState, diff)
	if err != nil {
		resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, err)
		return resp, nil
	}

	if newInstanceState != nil {
		// here we use the planned state to check for unknown/zero containers values
		// when normalizing the flatmap.
		plannedState := hcl2shim.FlatmapValueFromHCL2(plannedStateVal)
		newInstanceState.Attributes = normalizeFlatmapContainers(plannedState, newInstanceState.Attributes)
	}

	newStateVal := cty.NullVal(block.ImpliedType())

	// We keep the null val if we destroyed the resource, otherwise build the
	// entire object, even if the new state was nil.
	if !destroy {
		newStateVal, err = schema.StateValueFromInstanceState(newInstanceState, block.ImpliedType())
		if err != nil {
			resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, err)
			return resp, nil
		}
	}

	newStateVal = copyTimeoutValues(newStateVal, plannedStateVal)

	newStateMP, err := msgpack.Marshal(newStateVal, block.ImpliedType())
	if err != nil {
		resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, err)
		return resp, nil
	}
	resp.NewState = &proto.DynamicValue{
		Msgpack: newStateMP,
	}

	if newInstanceState != nil {
		meta, err := json.Marshal(newInstanceState.Meta)
		if err != nil {
			resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, err)
			return resp, nil
		}
		resp.Private = meta
	}

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

	newStateVal = copyTimeoutValues(newStateVal, configVal)

	newStateMP, err := msgpack.Marshal(newStateVal, block.ImpliedType())
	if err != nil {
		resp.Diagnostics = convert.AppendProtoDiag(resp.Diagnostics, err)
		return resp, nil
	}
	resp.State = &proto.DynamicValue{
		Msgpack: newStateMP,
	}
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

// normalizeFlatmapContainers removes empty containers, and fixes counts in a
// set of flatmapped attributes. The prior value is used to determine if there
// could be zero-length flatmap containers which we need to preserve. This
// allows a provider to set an empty computed container in the state without
// creating perpetual diff.
func normalizeFlatmapContainers(prior map[string]string, attrs map[string]string) map[string]string {
	keyRx := regexp.MustCompile(`.\.[%#]$`)

	// while we can't determine if the value was actually computed here, we will
	// trust that our shims stored and retrieved a zero-value container
	// correctly.
	zeros := map[string]bool{}
	for k, v := range prior {
		if keyRx.MatchString(k) && (v == "0" || v == hcl2shim.UnknownVariableValue) {
			zeros[k] = true
		}
	}

	// find container keys
	var keys []string
	for k, v := range attrs {
		if !keyRx.MatchString(k) {
			continue
		}

		if v == hcl2shim.UnknownVariableValue {
			// if the index value indicates the container is unknown, skip
			// updating the counts.
			continue
		}

		keys = append(keys, k)
	}

	// sort the keys in reverse, so that we check the longest subkeys first
	sort.Slice(keys, func(i, j int) bool {
		a, b := keys[i], keys[j]

		if strings.HasPrefix(a, b) {
			return true
		}

		if strings.HasPrefix(b, a) {
			return false
		}

		return a > b
	})

	for _, k := range keys {
		prefix := k[:len(k)-1]
		indexes := map[string]int{}
		for cand := range attrs {
			if cand == k {
				continue
			}

			if strings.HasPrefix(cand, prefix) {
				idx := cand[len(prefix):]
				dot := strings.Index(idx, ".")
				if dot > 0 {
					idx = idx[:dot]
				}
				indexes[idx]++
			}
		}

		switch {
		case len(indexes) == 0 && zeros[k]:
			// if there were no keys, but the value was known to be zero, the provider
			// must have set the computed value to an empty container, and we
			// need to leave it in the flatmap.
			attrs[k] = "0"
		case len(indexes) > 0:
			attrs[k] = strconv.Itoa(len(indexes))
		default:
			delete(attrs, k)
		}
	}

	return attrs
}

// helper/schema throws away timeout values from the config and stores them in
// the Private/Meta fields. we need to copy those values into the planned state
// so that core doesn't see a perpetual diff with the timeout block.
func copyTimeoutValues(to cty.Value, from cty.Value) cty.Value {
	// if `from` is null, then there are no attributes, and if `to` is null we
	// are planning to remove it altogether.
	if from.IsNull() || to.IsNull() {
		return to
	}

	fromAttrs := from.AsValueMap()
	timeouts, ok := fromAttrs[schema.TimeoutsConfigKey]

	// no timeouts to copy
	// timeouts shouldn't be unknown, but don't copy possibly invalid values
	if !ok || timeouts.IsNull() || !timeouts.IsWhollyKnown() {
		return to
	}

	toAttrs := to.AsValueMap()
	toAttrs[schema.TimeoutsConfigKey] = timeouts

	return cty.ObjectVal(toAttrs)
}

// stripResourceModifiers takes a *schema.Resource and returns a deep copy with all
// StateFuncs and CustomizeDiffs removed. This will be used during apply to
// create a diff from a planned state where the diff modifications have already
// been applied.
func stripResourceModifiers(r *schema.Resource) *schema.Resource {
	if r == nil {
		return nil
	}
	// start with a shallow copy
	newResource := new(schema.Resource)
	*newResource = *r

	newResource.CustomizeDiff = nil
	newResource.Schema = map[string]*schema.Schema{}

	for k, s := range r.Schema {
		newResource.Schema[k] = stripSchema(s)
	}

	return newResource
}

func stripSchema(s *schema.Schema) *schema.Schema {
	if s == nil {
		return nil
	}
	// start with a shallow copy
	newSchema := new(schema.Schema)
	*newSchema = *s

	newSchema.StateFunc = nil

	switch e := newSchema.Elem.(type) {
	case *schema.Schema:
		newSchema.Elem = stripSchema(e)
	case *schema.Resource:
		newSchema.Elem = stripResourceModifiers(e)
	}

	return newSchema
}
