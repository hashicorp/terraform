// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package rpcapi

import (
	"context"
	"io"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/go-slug/sourceaddrs"
	"github.com/hashicorp/go-slug/sourcebundle"
	"github.com/hashicorp/terraform-svchost/disco"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/depsfile"
	"github.com/hashicorp/terraform/internal/getproviders"
	"github.com/hashicorp/terraform/internal/rpcapi/terraform1"
	"github.com/hashicorp/terraform/internal/rpcapi/terraform1/dependencies"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/testing/protocmp"

	_ "github.com/hashicorp/terraform/internal/logging"
)

func TestDependenciesOpenCloseSourceBundle(t *testing.T) {
	ctx := context.Background()

	handles := newHandleTable()
	depsServer := newDependenciesServer(handles, disco.New())

	openResp, err := depsServer.OpenSourceBundle(ctx, &dependencies.OpenSourceBundle_Request{
		LocalPath: "testdata/sourcebundle",
	})
	if err != nil {
		t.Fatal(err)
	}

	// A client wouldn't normally be able to interact directly with the
	// source bundle, but we're doing that here to simulate what would
	// happen in another service that takes source bundle handles as input.
	// (This nested scope encapsulates some internal stuff that a normal client
	// would not have access to.)
	{
		hnd := handle[*sourcebundle.Bundle](openResp.SourceBundleHandle)
		sources := handles.SourceBundle(hnd)
		if sources == nil {
			t.Fatal("returned source bundle handle is invalid")
		}

		_, err = sources.LocalPathForSource(
			// The following is one of the source addresses known to the
			// source bundle we requested above.
			sourceaddrs.MustParseSource("git::https://example.com/foo.git").(sourceaddrs.FinalSource),
		)
		if err != nil {
			t.Fatalf("source bundle doesn't have the package we were expecting: %s", err)
		}
	}

	_, err = depsServer.CloseSourceBundle(ctx, &dependencies.CloseSourceBundle_Request{
		SourceBundleHandle: openResp.SourceBundleHandle,
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestDependencyLocks(t *testing.T) {
	ctx := context.Background()

	handles := newHandleTable()
	depsServer := newDependenciesServer(handles, disco.New())

	openSourcesResp, err := depsServer.OpenSourceBundle(ctx, &dependencies.OpenSourceBundle_Request{
		LocalPath: "testdata/sourcebundle",
	})
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		depsServer.CloseSourceBundle(ctx, &dependencies.CloseSourceBundle_Request{
			SourceBundleHandle: openSourcesResp.SourceBundleHandle,
		})
	}()

	openLocksResp, err := depsServer.OpenDependencyLockFile(ctx, &dependencies.OpenDependencyLockFile_Request{
		SourceBundleHandle: openSourcesResp.SourceBundleHandle,
		SourceAddress: &terraform1.SourceAddress{
			Source: "git::https://example.com/foo.git//.terraform.lock.hcl",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(openLocksResp.Diagnostics) != 0 {
		t.Error("OpenDependencyLockFile returned unexpected diagnostics")
	}

	// A client wouldn't normally be able to interact directly with the
	// locks object, but we're doing that here to simulate what would
	// happen in another service that takes dependency lock handles as input.
	// (This nested scope encapsulates some internal stuff that a normal client
	// would not have access to.)
	{
		hnd := handle[*depsfile.Locks](openLocksResp.DependencyLocksHandle)
		locks := handles.DependencyLocks(hnd)
		if locks == nil {
			t.Fatal("returned dependency locks handle is invalid")
		}

		wantProvider := addrs.MustParseProviderSourceString("example.com/foo/bar")
		got := locks.AllProviders()
		want := map[addrs.Provider]*depsfile.ProviderLock{
			wantProvider: depsfile.NewProviderLock(
				wantProvider, getproviders.MustParseVersion("1.2.3"),
				nil,
				[]getproviders.Hash{
					"zh:abcdabcdabcdabcdabcdabcdabcdabcdabcdabcdabcdabcdabcdabcdabcdabcd",
				},
			),
		}
		if diff := cmp.Diff(want, got, cmp.AllowUnexported(depsfile.ProviderLock{})); diff != "" {
			t.Errorf("wrong locked providers\n%s", diff)
		}
	}

	getProvidersResp, err := depsServer.GetLockedProviderDependencies(ctx, &dependencies.GetLockedProviderDependencies_Request{
		DependencyLocksHandle: openLocksResp.DependencyLocksHandle,
	})
	if err != nil {
		t.Fatal(err)
	}
	wantProviderLocks := []*terraform1.ProviderPackage{
		{
			SourceAddr: "example.com/foo/bar",
			Version:    "1.2.3",
			Hashes: []string{
				"zh:abcdabcdabcdabcdabcdabcdabcdabcdabcdabcdabcdabcdabcdabcdabcdabcd",
			},
		},
	}
	if diff := cmp.Diff(wantProviderLocks, getProvidersResp.SelectedProviders, protocmp.Transform()); diff != "" {
		t.Errorf("wrong GetLockedProviderDependencies result\n%s", diff)
	}

	_, err = depsServer.CloseDependencyLocks(ctx, &dependencies.CloseDependencyLocks_Request{
		DependencyLocksHandle: openLocksResp.DependencyLocksHandle,
	})
	if err != nil {
		t.Fatal(err)
	}

	// We should now be able to create a new locks handle referring to the
	// same providers as the one we just closed. This simulates a caller
	// propagating its provider locks between separate instances of rpcapi.
	newLocksResp, err := depsServer.CreateDependencyLocks(ctx, &dependencies.CreateDependencyLocks_Request{
		ProviderSelections: getProvidersResp.SelectedProviders,
	})
	if err != nil {
		t.Fatal(err)
	}

	getProvidersResp, err = depsServer.GetLockedProviderDependencies(ctx, &dependencies.GetLockedProviderDependencies_Request{
		DependencyLocksHandle: newLocksResp.DependencyLocksHandle,
	})
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(wantProviderLocks, getProvidersResp.SelectedProviders, protocmp.Transform()); diff != "" {
		t.Errorf("wrong GetLockedProviderDependencies result\n%s", diff)
	}
}

func TestDependenciesProviderCache(t *testing.T) {
	ctx := context.Background()

	handles := newHandleTable()
	depsServer := newDependenciesServer(handles, disco.New())

	// This test involves a streaming RPC operation, so we'll need help from
	// a real in-memory gRPC connection to exercise it concisely so that
	// we can work with the client API rather than the server API.
	grpcClient, close := grpcClientForTesting(ctx, t, func(srv *grpc.Server) {
		dependencies.RegisterDependenciesServer(srv, depsServer)
	})
	defer close()
	depsClient := dependencies.NewDependenciesClient(grpcClient)

	openSourcesResp, err := depsClient.OpenSourceBundle(ctx, &dependencies.OpenSourceBundle_Request{
		LocalPath: "testdata/sourcebundle",
	})
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_, err := depsClient.CloseSourceBundle(ctx, &dependencies.CloseSourceBundle_Request{
			SourceBundleHandle: openSourcesResp.SourceBundleHandle,
		})
		if err != nil {
			t.Error(err)
		}
	}()

	openLocksResp, err := depsClient.OpenDependencyLockFile(ctx, &dependencies.OpenDependencyLockFile_Request{
		SourceBundleHandle: openSourcesResp.SourceBundleHandle,
		SourceAddress: &terraform1.SourceAddress{
			Source: "git::https://example.com/foo.git//.terraform.lock.hcl",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(openLocksResp.Diagnostics) != 0 {
		t.Error("OpenDependencyLockFile returned unexpected diagnostics")
	}

	tmpDir := t.TempDir()
	cacheDir := filepath.Join(tmpDir, "pc")

	evts, err := depsClient.BuildProviderPluginCache(ctx, &dependencies.BuildProviderPluginCache_Request{
		DependencyLocksHandle: openLocksResp.DependencyLocksHandle,
		CacheDir:              cacheDir,

		// We force a local provider mirror and fake platform here just to keep
		// this test self-contained. This wraps the provider installer which
		// already has its own tests for the different installation methods,
		// so we don't need to be exhaustive about them all here.
		// (A real client of this API would typically just specify the "direct"
		// installation method, which retrieves packages from their origin
		// registries.)
		InstallationMethods: []*dependencies.BuildProviderPluginCache_Request_InstallMethod{
			{
				Source: &dependencies.BuildProviderPluginCache_Request_InstallMethod_LocalMirrorDir{
					LocalMirrorDir: "testdata/provider-fs-mirror",
				},
			},
		},
		OverridePlatform: "os_arch",
	})
	if err != nil {
		t.Fatal(err)
	}

	seenFakeProvider := false
	for {
		msg, err := evts.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err) // not expecting any errors
		}

		// TODO: We're not comprehensively testing all of the events right now
		// since we're primarily interested in whether the provider gets
		// installed at all, but once clients start depending on the events
		// for UI purposes we ought to add more coverage here for the other
		// event types.
		switch evt := msg.Event.(type) {
		case *dependencies.BuildProviderPluginCache_Event_Diagnostic:
			t.Errorf("unexpected diagnostic:\n\n%s\n\n%s", evt.Diagnostic.Summary, evt.Diagnostic.Detail)
		case *dependencies.BuildProviderPluginCache_Event_FetchComplete_:
			if evt.FetchComplete.ProviderVersion.SourceAddr == "example.com/foo/bar" {
				seenFakeProvider = true
				if got, want := evt.FetchComplete.ProviderVersion.Version, "1.2.3"; got != want {
					t.Errorf("wrong provider version\ngot:  %s\nwant: %s", got, want)
				}
			}
		}
		t.Logf("installation event: %s", msg.String())
	}

	if !seenFakeProvider {
		t.Error("no 'fetch complete' event for example.com/foo/bar")
	}

	openCacheResp, err := depsClient.OpenProviderPluginCache(ctx, &dependencies.OpenProviderPluginCache_Request{
		CacheDir:         cacheDir,
		OverridePlatform: "os_arch",
	})
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_, err := depsClient.CloseProviderPluginCache(ctx, &dependencies.CloseProviderPluginCache_Request{
			ProviderCacheHandle: openCacheResp.ProviderCacheHandle,
		})
		if err != nil {
			t.Error(err)
		}
	}()
	pkgsResp, err := depsClient.GetCachedProviders(ctx, &dependencies.GetCachedProviders_Request{
		ProviderCacheHandle: openCacheResp.ProviderCacheHandle,
	})
	if err != nil {
		t.Fatal(err)
	}

	got := pkgsResp.AvailableProviders
	want := []*terraform1.ProviderPackage{
		{
			SourceAddr: "example.com/foo/bar",
			Version:    "1.2.3",
			Hashes: []string{
				// This hash is of the fake package directory we installed
				// from, under testdata/provider-fs-mirror .
				"h1:cAp58lPuOAaPN9ZDdFHx9FxVK2NU0UeObQs2/zld9Lc=",
			},
		},
	}
	if diff := cmp.Diff(want, got, protocmp.Transform()); diff != "" {
		t.Errorf("wrong providers in cache reported after building\n%s", diff)
	}
}

func TestDependenciesProviderSchema(t *testing.T) {
	ctx := context.Background()

	handles := newHandleTable()
	depsServer := newDependenciesServer(handles, disco.New())

	providersResp, err := depsServer.GetBuiltInProviders(ctx, &dependencies.GetBuiltInProviders_Request{})
	if err != nil {
		t.Fatal(err)
	}
	{
		got := providersResp.AvailableProviders
		want := []*terraform1.ProviderPackage{
			{
				SourceAddr: "terraform.io/builtin/terraform",
			},
		}
		if diff := cmp.Diff(want, got, protocmp.Transform()); diff != "" {
			t.Errorf("wrong built-in providers\n%s", diff)
		}
	}

	schemaResp, err := depsServer.GetProviderSchema(ctx, &dependencies.GetProviderSchema_Request{
		ProviderAddr: "terraform.io/builtin/terraform",
	})
	if err != nil {
		t.Fatal(err)
	}
	{
		got := schemaResp.Schema
		want := &dependencies.ProviderSchema{
			ProviderConfig: &dependencies.Schema{
				Block: &dependencies.Schema_Block{
					// This provider has no configuration arguments
				},
			},
			DataResourceTypes: map[string]*dependencies.Schema{
				"terraform_remote_state": &dependencies.Schema{
					Block: &dependencies.Schema_Block{
						Attributes: []*dependencies.Schema_Attribute{
							{
								Name:     "backend",
								Type:     []byte(`"string"`),
								Required: true,
								Description: &dependencies.Schema_DocString{
									Description: "The remote backend to use, e.g. `remote` or `http`.",
									Format:      dependencies.Schema_DocString_MARKDOWN,
								},
							},
							{
								Name:     "config",
								Type:     []byte(`"dynamic"`),
								Optional: true,
								Description: &dependencies.Schema_DocString{
									Description: "The configuration of the remote backend. Although this is optional, most backends require some configuration.\n\nThe object can use any arguments that would be valid in the equivalent `terraform { backend \"<TYPE>\" { ... } }` block.",
									Format:      dependencies.Schema_DocString_MARKDOWN,
								},
							},
							{
								Name:     "defaults",
								Type:     []byte(`"dynamic"`),
								Optional: true,
								Description: &dependencies.Schema_DocString{
									Description: "Default values for outputs, in case the state file is empty or lacks a required output.",
									Format:      dependencies.Schema_DocString_MARKDOWN,
								},
							},
							{
								Name:     "outputs",
								Type:     []byte(`"dynamic"`),
								Computed: true,
								Description: &dependencies.Schema_DocString{
									Description: "An object containing every root-level output in the remote state.",
									Format:      dependencies.Schema_DocString_MARKDOWN,
								},
							},
							{
								Name:     "workspace",
								Type:     []byte(`"string"`),
								Optional: true,
								Description: &dependencies.Schema_DocString{
									Description: "The Terraform workspace to use, if the backend supports workspaces.",
									Format:      dependencies.Schema_DocString_MARKDOWN,
								},
							},
						},
					},
				},
			},
			ManagedResourceTypes: map[string]*dependencies.Schema{
				"terraform_data": &dependencies.Schema{
					Block: &dependencies.Schema_Block{
						Attributes: []*dependencies.Schema_Attribute{
							{
								Name:     "id",
								Type:     []byte(`"string"`),
								Computed: true,
							},
							{
								Name:     "input",
								Type:     []byte(`"dynamic"`),
								Optional: true,
							},
							{
								Name:     "output",
								Type:     []byte(`"dynamic"`),
								Computed: true,
							},
							{
								Name:     "triggers_replace",
								Type:     []byte(`"dynamic"`),
								Optional: true,
							},
						},
					},
				},
			},
		}
		if diff := cmp.Diff(want, got, protocmp.Transform()); diff != "" {
			// NOTE: This is testing against the schema of a real provider
			// that can evolve independently of rpcapi. If that provider's
			// schema changes in future then it's expected that this test
			// will fail, and it's okay to change "want" to match as long as
			// it's a correct description of that provider's updated schema.
			//
			// If this turns out to be a big maintenence burden then we could
			// consider some way to include a mock provider, but that would
			// add another possible kind of provider into the mix and we'd
			// rather avoid that complexity if possible.
			t.Errorf("unexpected schema for the built-in 'terraform' provider\n%s", diff)
		}
	}

}
