package rpcapi

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform/internal/rpcapi/tfcore1"
)

func TestServerOpenCloseConfig(t *testing.T) {
	ctx := context.Background()
	configDir := t.TempDir()

	// We're not actually going to use any providers here, so we can
	// provide a nil factory without any problems.
	client := newV1ClientForTests(t, configDir, coreOptsWithTestProvider(nil))

	resp, err := client.OpenConfigCwd(ctx, &tfcore1.OpenConfigCwd_Request{})
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Diagnostics) > 0 {
		t.Fatalf("unexpected diagnostics\n%s", cmp.Diff(nil, resp.Diagnostics))
	}
	if resp.ConfigId == 0 {
		t.Fatal("not assigned a configuration id")
	}
	t.Logf("configuration id is %d", resp.ConfigId)

	_, err = client.CloseConfig(ctx, &tfcore1.CloseConfig_Request{
		ConfigId: resp.ConfigId,
	})
	if err != nil {
		t.Fatal(err)
	}
}
