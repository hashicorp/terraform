// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/providers"
	testing_provider "github.com/hashicorp/terraform/internal/providers/testing"
	"github.com/zclconf/go-cty/cty"
)

func TestContext2Apply_ephemeralProviderRef(t *testing.T) {
	m := testModuleInline(t, map[string]string{
		"main.tf": `
ephemeral "ephem_resource" "data" {
}

provider "test" {
  test_string = ephemeral.ephem_resource.data.value
}

resource "test_object" "test" {
}
`,
	})

	ephem := &testing_provider.MockProvider{
		GetProviderSchemaResponse: &providers.GetProviderSchemaResponse{
			EphemeralResourceTypes: map[string]providers.Schema{
				"ephem_resource": {
					Block: &configschema.Block{
						Attributes: map[string]*configschema.Attribute{
							"value": {
								Type:     cty.String,
								Computed: true,
							},
						},
					},
				},
			},
		},
	}

	ephem.OpenEphemeralResourceFn = func(providers.OpenEphemeralResourceRequest) (resp providers.OpenEphemeralResourceResponse) {
		resp.Result = cty.ObjectVal(map[string]cty.Value{
			"value": cty.StringVal("test string"),
		})
		resp.RenewAt = time.Now().Add(11 * time.Millisecond)
		resp.Private = []byte("private data")
		return resp
	}

	// make sure we can wait for renew to be called
	renewed := make(chan bool)
	renewDone := sync.OnceFunc(func() { close(renewed) })

	ephem.RenewEphemeralResourceFn = func(req providers.RenewEphemeralResourceRequest) (resp providers.RenewEphemeralResourceResponse) {
		defer renewDone()
		if string(req.Private) != "private data" {
			resp.Diagnostics = resp.Diagnostics.Append(fmt.Errorf("invalid private data %q", req.Private))
			return resp
		}

		resp.RenewAt = time.Now().Add(10 * time.Millisecond)
		resp.Private = req.Private
		return resp
	}

	p := simpleMockProvider()
	p.ConfigureProviderFn = func(req providers.ConfigureProviderRequest) (resp providers.ConfigureProviderResponse) {
		// wait here for the ephemeral value to be renewed at least once
		<-renewed
		if req.Config.GetAttr("test_string").AsString() != "test string" {
			resp.Diagnostics = resp.Diagnostics.Append(fmt.Errorf("received config did not contain \"test string\", got %#v\n", req.Config))
		}
		return resp
	}

	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			// The providers never actually going to get called here, we should
			// catch the error long before anything happens.
			addrs.NewDefaultProvider("ephem"): testProviderFuncFixed(ephem),
			addrs.NewDefaultProvider("test"):  testProviderFuncFixed(p),
		},
	})

	plan, diags := ctx.Plan(m, nil, DefaultPlanOpts)
	assertNoDiagnostics(t, diags)

	if !ephem.OpenEphemeralResourceCalled {
		t.Error("OpenEphemeralResourceCalled not called")
	}
	if !ephem.RenewEphemeralResourceCalled {
		t.Error("RenewEphemeralResourceCalled not called")
	}
	if !ephem.CloseEphemeralResourceCalled {
		t.Error("CloseEphemeralResourceCalled not called")
	}

	// reset the ephemeral call flags and the gate
	ephem.OpenEphemeralResourceCalled = false
	ephem.RenewEphemeralResourceCalled = false
	ephem.CloseEphemeralResourceCalled = false
	renewed = make(chan bool)
	renewDone = sync.OnceFunc(func() { close(renewed) })

	_, diags = ctx.Apply(plan, m, nil)
	assertNoDiagnostics(t, diags)

	if !ephem.OpenEphemeralResourceCalled {
		t.Error("OpenEphemeralResourceCalled not called")
	}
	if !ephem.RenewEphemeralResourceCalled {
		t.Error("RenewEphemeralResourceCalled not called")
	}
	if !ephem.CloseEphemeralResourceCalled {
		t.Error("CloseEphemeralResourceCalled not called")
	}

	time.Sleep(time.Second)
}
