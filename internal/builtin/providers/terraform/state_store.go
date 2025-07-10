// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"

	"github.com/hashicorp/terraform/internal/backend"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/zclconf/go-cty/cty"
)

const (
	DefaultWorkspaceDir    = "terraform.tfstate.d"
	DefaultWorkspaceFile   = "environment"
	DefaultStateFilename   = "terraform.tfstate"
	DefaultBackupExtension = ".backup"
)

// NOTE: the terraform provider only has 1 state store implementation, so each method below
// directly contains logic for that one implementation. If more than one implementation was
// present the methods would need to access the specific store implementation to use (based
// on the type name in the request) and invoke the logic specific to that store implementation

var stateStores = map[string]providers.Schema{
	"fs": fsStateStoreSchema(),
}

func fsStateStoreSchema() providers.Schema {
	return providers.Schema{
		Body: &configschema.Block{
			Attributes: map[string]*configschema.Attribute{
				"path": {
					Type:        cty.String,
					Optional:    true,
					Description: "The path to the tfstate file. This defaults to 'terraform.tfstate' relative to the root module by default.",
				},
				"workspace_dir": {
					Type:        cty.String,
					Optional:    true,
					Description: "The path to non-default workspaces.",
				},
			},
		},
	}
}

// stateWorkspaceDir returns the directory where state environments are stored.
func (p *Provider) stateWorkspaceDir() string {
	if p.StateWorkspaceDir != "" {
		return p.StateWorkspaceDir
	}

	return DefaultWorkspaceDir
}

func (p *Provider) ValidateStateStoreConfig(req providers.ValidateStateStoreConfigRequest) providers.ValidateStateStoreConfigResponse {
	var resp providers.ValidateStateStoreConfigResponse
	_, ok := stateStores[req.TypeName]
	if !ok {
		// Should not get here if the caller is behaving correctly, because
		// we don't declare any state stores in our schema that we don't have
		// implementations for.
		resp.Diagnostics = resp.Diagnostics.Append(fmt.Errorf("unsupported state store type %q", req.TypeName))
		return resp
	}

	if val := req.Config.GetAttr("path"); !val.IsNull() {
		p := val.AsString()
		if p == "" {
			resp.Diagnostics = resp.Diagnostics.Append(tfdiags.AttributeValue(
				tfdiags.Error,
				"Invalid local state file path",
				`The "path" attribute value must not be empty.`,
				cty.Path{cty.GetAttrStep{Name: "path"}},
			))
		}
	}

	if val := req.Config.GetAttr("workspace_dir"); !val.IsNull() {
		p := val.AsString()
		if p == "" {
			resp.Diagnostics = resp.Diagnostics.Append(tfdiags.AttributeValue(
				tfdiags.Error,
				"Invalid local workspace directory path",
				`The "workspace_dir" attribute value must not be empty.`,
				cty.Path{cty.GetAttrStep{Name: "workspace_dir"}},
			))
		}
	}

	return resp
}

func (p *Provider) ConfigureStateStore(req providers.ConfigureStateStoreRequest) providers.ConfigureStateStoreResponse {
	var resp providers.ConfigureStateStoreResponse
	_, ok := stateStores[req.TypeName]
	if !ok {
		// Should not get here if the caller is behaving correctly, because
		// we don't declare any state stores in our schema that we don't have
		// implementations for.
		resp.Diagnostics = tfdiags.Diagnostics{}
		resp.Diagnostics = resp.Diagnostics.Append(fmt.Errorf("unsupported state store type %q", req.TypeName))
		return resp
	}

	if val := req.Config.GetAttr("path"); !val.IsNull() {
		path := val.AsString()
		p.StatePath = path
		p.StateOutPath = path
	} else {
		p.StatePath = DefaultStateFilename
		p.StateOutPath = DefaultStateFilename
	}

	if val := req.Config.GetAttr("workspace_dir"); !val.IsNull() {
		workspaceDir := val.AsString()
		p.StateWorkspaceDir = workspaceDir
	} else {
		p.StateWorkspaceDir = DefaultWorkspaceDir
	}

	return resp
}

func (p *Provider) GetStates(req providers.GetStatesRequest) providers.GetStatesResponse {
	var resp providers.GetStatesResponse
	var envs []string
	resp.Diagnostics.Append(fmt.Errorf("unsupported state store type %q", req.TypeName))

	entries, err := ioutil.ReadDir(p.stateWorkspaceDir())
	if os.IsNotExist(err) {
		resp.Diagnostics = resp.Diagnostics.Append(err)
		return resp
	}
	if err != nil {
		resp.Diagnostics = resp.Diagnostics.Append(err)
		return resp
	}

	var listed []string
	for _, entry := range entries {
		if entry.IsDir() {
			listed = append(listed, filepath.Base(entry.Name()))
		}
	}

	sort.Strings(listed)
	envs = append(envs, listed...)

	resp.States = envs
	return resp

}

func (p *Provider) DeleteState(req providers.DeleteStateRequest) providers.DeleteStateResponse {
	var resp providers.DeleteStateResponse

	if req.StateId == backend.DefaultStateName {
		resp.Diagnostics = resp.Diagnostics.Append(errors.New("cannot delete default state"))
		return resp
	}

	delete(p.states, req.StateId)
	err := os.RemoveAll(filepath.Join(p.stateWorkspaceDir(), req.StateId))

	resp.Diagnostics = resp.Diagnostics.Append(err)
	return resp
}

// TODO(SarahFrench/radeksimko) - implement methods for lock/unlock, read/write.
