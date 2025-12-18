// Copyright IBM Corp. 2014, 2025
// SPDX-License-Identifier: BUSL-1.1

package e2etest

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"
	"github.com/hashicorp/terraform/internal/e2e"
	"github.com/hashicorp/terraform/internal/grpcwrap"
	tfplugin5 "github.com/hashicorp/terraform/internal/plugin"
	tfplugin "github.com/hashicorp/terraform/internal/plugin6"
	simple "github.com/hashicorp/terraform/internal/provider-simple-v6"
	proto5 "github.com/hashicorp/terraform/internal/tfplugin5"
	proto "github.com/hashicorp/terraform/internal/tfplugin6"
)

func TestUnmanagedQuery(t *testing.T) {
	if !canRunGoBuild {
		// We're running in a separate-build-then-run context, so we can't
		// currently execute this test which depends on being able to build
		// new executable at runtime.
		//
		// (See the comment on canRunGoBuild's declaration for more information.)
		t.Skip("can't run without building a new provider executable")
	}

	t.Parallel()

	tests := []struct {
		name            string
		protocolVersion int
	}{
		{
			name:            "proto6",
			protocolVersion: 6,
		},
		{
			name:            "proto5",
			protocolVersion: 5,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			os.Setenv(e2e.TestExperimentFlag, "true")
			terraformBin := e2e.GoBuild("github.com/hashicorp/terraform", "terraform")

			fixturePath := filepath.Join("testdata", "query-provider")
			tf := e2e.NewBinary(t, terraformBin, fixturePath)

			reattachCh := make(chan *plugin.ReattachConfig)
			closeCh := make(chan struct{})

			var provider interface {
				ListResourceCalled() bool
			}
			var versionedPlugins map[int]plugin.PluginSet

			// Configure provider and plugins based on protocol version
			if tc.protocolVersion == 6 {
				provider6 := &providerServer{
					ProviderServer: grpcwrap.Provider6(simple.Provider()),
				}
				provider = provider6
				versionedPlugins = map[int]plugin.PluginSet{
					6: {
						"provider": &tfplugin.GRPCProviderPlugin{
							GRPCProvider: func() proto.ProviderServer {
								return provider6
							},
						},
					},
				}
			} else {
				provider5 := &providerServer5{
					ProviderServer: grpcwrap.Provider(simple.Provider()),
				}
				provider = provider5
				versionedPlugins = map[int]plugin.PluginSet{
					5: {
						"provider": &tfplugin5.GRPCProviderPlugin{
							GRPCProvider: func() proto5.ProviderServer {
								return provider5
							},
						},
					},
				}
			}

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			go plugin.Serve(&plugin.ServeConfig{
				Logger: hclog.New(&hclog.LoggerOptions{
					Name:   "plugintest",
					Level:  hclog.Trace,
					Output: io.Discard,
				}),
				Test: &plugin.ServeTestConfig{
					Context:          ctx,
					ReattachConfigCh: reattachCh,
					CloseCh:          closeCh,
				},
				GRPCServer:       plugin.DefaultGRPCServer,
				VersionedPlugins: versionedPlugins,
			})
			config := <-reattachCh
			if config == nil {
				t.Fatalf("no reattach config received")
			}
			reattachStr, err := json.Marshal(map[string]reattachConfig{
				"hashicorp/test": {
					Protocol:        string(config.Protocol),
					ProtocolVersion: tc.protocolVersion,
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

			//// INIT
			stdout, stderr, err := tf.Run("init")
			if err != nil {
				t.Fatalf("unexpected init error: %s\nstderr:\n%s", err, stderr)
			}

			// Make sure we didn't download the binary
			if strings.Contains(stdout, "Installing hashicorp/test v") {
				t.Errorf("test provider download message is present in init output:\n%s", stdout)
			}
			if tf.FileExists(filepath.Join(".terraform", "plugins", "registry.terraform.io", "hashicorp", "test")) {
				t.Errorf("test provider binary found in .terraform dir")
			}

			//// QUERY
			stdout, stderr, err = tf.Run("query")
			if err != nil {
				t.Fatalf("unexpected query error: %s\nstderr:\n%s", err, stderr)
			}

			if !provider.ListResourceCalled() {
				t.Error("ListResource not called on un-managed provider")
			}

			// The output should contain the expected resource data. (using regex so that the number of whitespace characters doesn't matter)
			regex := regexp.MustCompile(`(?m)^list\.simple_resource\.test\s+id=static_id\s+static_display_name$`)
			if !regex.MatchString(stdout) {
				t.Errorf("expected resource data not found in output:\n%s", stdout)
			}

			cancel()
			<-closeCh
		})
	}
}
