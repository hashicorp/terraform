// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package states

import (
	"strings"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/internal/backend"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/states/statemgr"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/zclconf/go-cty/cty"
)

func TestGetBackendInstanceReturnsConfigureDiagnostics(t *testing.T) {
	t.Parallel()

	const wantErr = "configure failed"

	_, err := getBackendInstance("test", &configs.Backend{
		Config: hcl.EmptyBody(),
	}, func() backend.Backend {
		return &backendWithConfigureError{
			configureDiags: tfdiags.Diagnostics{}.Append(
				tfdiags.Sourceless(tfdiags.Error, "Configure failed", wantErr),
			),
		}
	})
	if err == nil {
		t.Fatal("expected configure error, got nil")
	}
	if !strings.Contains(err.Error(), wantErr) {
		t.Fatalf("expected error containing %q, got %q", wantErr, err)
	}
}

type backendWithConfigureError struct {
	configureDiags tfdiags.Diagnostics
}

func (b *backendWithConfigureError) ConfigSchema() *configschema.Block {
	return &configschema.Block{}
}

func (b *backendWithConfigureError) PrepareConfig(config cty.Value) (cty.Value, tfdiags.Diagnostics) {
	return config, nil
}

func (b *backendWithConfigureError) Configure(cty.Value) tfdiags.Diagnostics {
	return b.configureDiags
}

func (b *backendWithConfigureError) StateMgr(string) (statemgr.Full, tfdiags.Diagnostics) {
	return nil, nil
}

func (b *backendWithConfigureError) DeleteWorkspace(string, bool) tfdiags.Diagnostics {
	return nil
}

func (b *backendWithConfigureError) Workspaces() ([]string, tfdiags.Diagnostics) {
	return nil, nil
}
