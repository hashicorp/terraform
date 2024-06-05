// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package rpcapi

import (
	"fmt"
	"os/exec"
	"sort"

	"github.com/apparentlymart/go-versions/versions"
	"github.com/hashicorp/go-plugin"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/getproviders"
	"github.com/hashicorp/terraform/internal/logging"
	tfplugin "github.com/hashicorp/terraform/internal/plugin"
	tfplugin6 "github.com/hashicorp/terraform/internal/plugin6"
	"github.com/hashicorp/terraform/internal/providercache"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/rpcapi/rawrpc/rawdependencies1"
	"github.com/hashicorp/terraform/internal/rpcapi/rawrpc/rawschema"
)

// This file contains helper functions and supporting logic for
// Dependencies.GetProviderSchema. The function entry point is in
// dependencies.go with all of the other Dependencies functions.

// loadProviderSchema attempts to load the schema for a given provider.
//
// If the providerAddr is for a built-in provider then version must be
// [versions.Unspecified] and cacheDir may be nil, although that's not
// required.
//
// If providerAddr is for a non-builtin provider then both version and
// cacheDir are required.
func loadProviderSchema(providerAddr addrs.Provider, version getproviders.Version, cacheDir *providercache.Dir) (providers.GetProviderSchemaResponse, error) {
	var provider providers.Interface
	switch {
	case providerAddr.IsBuiltIn():
		if version != versions.Unspecified {
			return providers.GetProviderSchemaResponse{}, fmt.Errorf("built-in providers are unversioned")
		}

		var err error
		provider, err = unconfiguredBuiltinProviderInstance(providerAddr)
		if err != nil {
			return providers.GetProviderSchemaResponse{}, err
		}

	default:
		cached := cacheDir.ProviderVersion(providerAddr, version)
		if cached == nil {
			return providers.GetProviderSchemaResponse{}, fmt.Errorf("provider cache does not include %s v%s", providerAddr, version)
		}

		var err error
		provider, err = unconfiguredProviderPluginInstance(cached)
		if err != nil {
			return providers.GetProviderSchemaResponse{}, fmt.Errorf("failed to launch provider plugin: %w", err)
		}
	}

	resp := provider.GetProviderSchema()
	return resp, nil
}

func unconfiguredProviderPluginInstance(cached *providercache.CachedProvider) (providers.Interface, error) {
	execFile, err := cached.ExecutableFile()
	if err != nil {
		return nil, err
	}

	config := &plugin.ClientConfig{
		HandshakeConfig:  tfplugin.Handshake,
		Logger:           logging.NewProviderLogger(""),
		AllowedProtocols: []plugin.Protocol{plugin.ProtocolGRPC},
		Managed:          true,
		Cmd:              exec.Command(execFile),
		AutoMTLS:         true,
		VersionedPlugins: tfplugin.VersionedPlugins,
	}

	client := plugin.NewClient(config)
	rpcClient, err := client.Client()
	if err != nil {
		return nil, err
	}

	raw, err := rpcClient.Dispense(tfplugin.ProviderPluginName)
	if err != nil {
		return nil, err
	}

	// store the client so that the plugin can kill the child process
	protoVer := client.NegotiatedVersion()
	switch protoVer {
	case 5:
		p := raw.(*tfplugin.GRPCProvider)
		p.PluginClient = client
		p.Addr = cached.Provider
		return p, nil
	case 6:
		p := raw.(*tfplugin6.GRPCProvider)
		p.PluginClient = client
		p.Addr = cached.Provider
		return p, nil
	default:
		panic("unsupported protocol version")
	}
}

func unconfiguredBuiltinProviderInstance(addr addrs.Provider) (providers.Interface, error) {
	if !addr.IsBuiltIn() {
		panic("unconfiguredBuiltinProviderInstance for non-builtin provider")
	}
	factory, ok := builtinProviders[addr.Type]
	if !ok {
		return nil, fmt.Errorf("this version of Terraform does not support provider %s", addr)
	}
	return factory(), nil
}

func providerSchemaToProto(schemaResp providers.GetProviderSchemaResponse) *rawdependencies1.ProviderSchema {
	// Due to some historical poor design planning, the provider protocol uses
	// different terminology than the user-facing terminology for Terraform
	// Core and the Terraform language, and so part of our job here is to
	// map between the two so that rpcapi uses Terraform Core's words
	// rather than the provider protocol's words.
	//
	// This result currently includes only the subset of the schema information
	// that would be needed to successfully interpret DynamicValue messages
	// returned from other rpcapi operations. Exporting the full provider
	// protocol schema model here would tightly couple the rpcapi to the
	// provider protocol, forcing them to always change together, which is
	// undesirable since each one has a different target audience and therefore
	// will probably follow different evolutionary paths. For example, Terraform
	// can support multiple provider protocol versions concurrently but will
	// probably not want to make a new rpcapi protocol major version each time
	// a new provider protocol version is added or removed.

	mrtSchemas := make(map[string]*rawschema.Schema, len(schemaResp.ResourceTypes))
	drtSchemas := make(map[string]*rawschema.Schema, len(schemaResp.DataSources))

	for name, elem := range schemaResp.ResourceTypes {
		mrtSchemas[name] = schemaElementToProto(elem)
	}
	for name, elem := range schemaResp.DataSources {
		drtSchemas[name] = schemaElementToProto(elem)
	}

	return &rawdependencies1.ProviderSchema{
		ProviderConfig:       schemaElementToProto(schemaResp.Provider),
		ManagedResourceTypes: mrtSchemas,
		DataResourceTypes:    drtSchemas,
	}
}

func schemaElementToProto(elem providers.Schema) *rawschema.Schema {
	return &rawschema.Schema{
		Block: schemaBlockToProto(elem.Block),
	}
}

func schemaBlockToProto(block *configschema.Block) *rawschema.Schema_Block {
	if block == nil {
		return &rawschema.Schema_Block{}
	}
	attributes := make([]*rawschema.Schema_Attribute, 0, len(block.Attributes))
	for name, attr := range block.Attributes {
		attributes = append(attributes, schemaAttributeToProto(name, attr))
	}
	sort.Slice(attributes, func(i, j int) bool {
		return attributes[i].Name < attributes[j].Name
	})
	blockTypes := make([]*rawschema.Schema_NestedBlock, 0, len(block.BlockTypes))
	for typeName, blockType := range block.BlockTypes {
		blockTypes = append(blockTypes, schemaNestedBlockToProto(typeName, blockType))
	}
	sort.Slice(blockTypes, func(i, j int) bool {
		return blockTypes[i].TypeName < blockTypes[j].TypeName
	})
	return &rawschema.Schema_Block{
		Deprecated:  block.Deprecated,
		Description: schemaDocstringToProto(block.Description, block.DescriptionKind),
		Attributes:  attributes,
		BlockTypes:  blockTypes,
	}
}

func schemaAttributeToProto(name string, attr *configschema.Attribute) *rawschema.Schema_Attribute {
	var err error
	var typeBytes []byte
	var objectType *rawschema.Schema_Object
	if attr.NestedType != nil {
		objectType = schemaNestedObjectTypeToProto(attr.NestedType)
	} else {
		typeBytes, err = attr.Type.MarshalJSON()
		if err != nil {
			// Should never happen because types we get here are either from
			// inside this program (for built-in providers) or already transited
			// through the plugin protocol's equivalent of this serialization.
			panic(fmt.Sprintf("can't encode %#v as JSON: %s", attr.Type, err))
		}
	}

	return &rawschema.Schema_Attribute{
		Name:        name,
		Type:        typeBytes,
		NestedType:  objectType,
		Description: schemaDocstringToProto(attr.Description, attr.DescriptionKind),
		Required:    attr.Required,
		Optional:    attr.Optional,
		Computed:    attr.Computed,
		Sensitive:   attr.Sensitive,
		Deprecated:  attr.Deprecated,
	}
}

func schemaNestedBlockToProto(typeName string, blockType *configschema.NestedBlock) *rawschema.Schema_NestedBlock {
	var protoNesting rawschema.Schema_NestedBlock_NestingMode
	switch blockType.Nesting {
	case configschema.NestingSingle:
		protoNesting = rawschema.Schema_NestedBlock_SINGLE
	case configschema.NestingGroup:
		protoNesting = rawschema.Schema_NestedBlock_GROUP
	case configschema.NestingList:
		protoNesting = rawschema.Schema_NestedBlock_LIST
	case configschema.NestingSet:
		protoNesting = rawschema.Schema_NestedBlock_SET
	case configschema.NestingMap:
		protoNesting = rawschema.Schema_NestedBlock_MAP
	default:
		// The above should be exhaustive for all configschema.NestingMode variants
		panic(fmt.Sprintf("invalid structural attribute nesting mode %s", blockType.Nesting))
	}

	return &rawschema.Schema_NestedBlock{
		TypeName: typeName,
		Block:    schemaBlockToProto(&blockType.Block),
		Nesting:  protoNesting,
	}
}

func schemaNestedObjectTypeToProto(objType *configschema.Object) *rawschema.Schema_Object {
	var protoNesting rawschema.Schema_Object_NestingMode
	switch objType.Nesting {
	case configschema.NestingSingle:
		protoNesting = rawschema.Schema_Object_SINGLE
	case configschema.NestingList:
		protoNesting = rawschema.Schema_Object_LIST
	case configschema.NestingSet:
		protoNesting = rawschema.Schema_Object_SET
	case configschema.NestingMap:
		protoNesting = rawschema.Schema_Object_MAP
	default:
		// The above should be exhaustive for all configschema.NestingMode variants
		panic(fmt.Sprintf("invalid structural attribute nesting mode %s", objType.Nesting))
	}

	attributes := make([]*rawschema.Schema_Attribute, 0, len(objType.Attributes))
	for name, attr := range objType.Attributes {
		attributes = append(attributes, schemaAttributeToProto(name, attr))
	}
	sort.Slice(attributes, func(i, j int) bool {
		return attributes[i].Name < attributes[j].Name
	})

	return &rawschema.Schema_Object{
		Nesting:    protoNesting,
		Attributes: attributes,
	}
}

func schemaDocstringToProto(doc string, format configschema.StringKind) *rawschema.Schema_DocString {
	if doc == "" {
		return nil
	}
	var protoFormat rawschema.Schema_DocString_Format
	switch format {
	case configschema.StringPlain:
		protoFormat = rawschema.Schema_DocString_PLAIN
	case configschema.StringMarkdown:
		protoFormat = rawschema.Schema_DocString_MARKDOWN
	default:
		// We'll ignore strings in unsupported formats, although we should
		// try to keep the above exhaustive if we add new formats in future.
		return nil
	}
	return &rawschema.Schema_DocString{
		Description: doc,
		Format:      protoFormat,
	}
}
