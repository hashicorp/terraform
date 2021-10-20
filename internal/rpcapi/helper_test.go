package rpcapi

import (
	"testing"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/rpcapi/tfcore1"
	"github.com/hashicorp/terraform/internal/terraform"
)

// inProcessV1Client is an implementation of tfcore1.TerraformClient that just
// directly wraps a tfcore1.TerraformServer, so that we can write most of our
// unit tests as more-or-less direct function calls.
//
// This doesn't test that the rpcplugin mechanisms are working correctly, so
// we should also make sure to have tests verify that we can start up the
// plugin server in a child process and actually send calls to it.
type inProcessV1Client struct {
	server tfcore1.TerraformServer
}

var _ tfcore1.TerraformClient = inProcessV1Client{}

func newV1ClientForTests(t *testing.T, opts *terraform.ContextOpts) tfcore1.TerraformClient {
	t.Helper()

	core, diags := terraform.NewContext(opts)
	if diags.HasErrors() {
		t.Fatalf("failed to instantiate Terraform Core: %s", diags.Err().Error())
	}

	server := newV1PluginServer(core)
	return inProcessV1Client{server}
}

func contextOptsWithTestProvider(factory providers.Factory) *terraform.ContextOpts {
	return &terraform.ContextOpts{
		Meta: &terraform.ContextMeta{
			Env: "default",

			// NOTE: This is just a placeholder to make this contain _something_
			// reasonable, but it's not super realistic because normally
			// this would be an absolute filesystem path in real use.
			OriginalWorkingDir: ".",
		},
		Parallelism: 10,
		Providers: map[addrs.Provider]providers.Factory{
			addrs.MustParseProviderSourceString("hashicorp/test"): factory,
		},
		// All other options left intentionally unset
	}
}
