// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package e2etest

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"
	"github.com/hashicorp/terraform/internal/e2e"
	"github.com/hashicorp/terraform/internal/grpcwrap"
	tfplugin6 "github.com/hashicorp/terraform/internal/plugin6"
	simple6 "github.com/hashicorp/terraform/internal/provider-simple-v6"
	proto6 "github.com/hashicorp/terraform/internal/tfplugin6"
)

func TestWorkspace_stateStore_new(t *testing.T) {
	t.Parallel()

	fixturePath := filepath.Join("testdata", "workspace")
	tf := e2e.NewBinary(t, terraformBin, fixturePath)

	reattachCh := make(chan *plugin.ReattachConfig)
	closeCh := make(chan struct{})
	provider := &providerServer{
		ProviderServer: grpcwrap.Provider6(simple6.Provider()),
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go plugin.Serve(&plugin.ServeConfig{
		Logger: hclog.New(&hclog.LoggerOptions{
			Name:   "plugintest",
			Level:  hclog.Trace,
			Output: ioutil.Discard,
		}),
		Test: &plugin.ServeTestConfig{
			Context:          ctx,
			ReattachConfigCh: reattachCh,
			CloseCh:          closeCh,
		},
		GRPCServer: plugin.DefaultGRPCServer,
		VersionedPlugins: map[int]plugin.PluginSet{
			6: {
				"provider": &tfplugin6.GRPCProviderPlugin{
					GRPCProvider: func() proto6.ProviderServer {
						return provider
					},
				},
			},
		},
	})
	config := <-reattachCh
	if config == nil {
		t.Fatalf("no reattach config received")
	}
	reattachStr, err := json.Marshal(map[string]reattachConfig{
		"hashicorp/test": {
			Protocol:        string(config.Protocol),
			ProtocolVersion: 6,
			Pid:             config.Pid,
			Test:            true,
			Addr: reattachConfigAddr{
				Network: config.Addr.Network(),
				String:  config.Addr.String(),
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	tf.AddEnv("TF_REATTACH_PROVIDERS=" + string(reattachStr))

	// Run command
	workspaceName := "my-workspace"
	stdout, stderr, err := tf.Run("workspace", "new", workspaceName)
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}

	if stderr != "" {
		t.Errorf("unexpected stderr output:\n%s", stderr)
	}

	wantString := fmt.Sprintf("Created and switched to workspace %q!", workspaceName)
	if !strings.Contains(stdout, wantString) {
		t.Errorf("output does not contain the expected string %q:\n%s", wantString, stdout)
	}
}
