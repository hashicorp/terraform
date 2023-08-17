package rpcapi

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/go-slug/sourceaddrs"
	"github.com/hashicorp/go-slug/sourcebundle"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/depsfile"
	"github.com/hashicorp/terraform/internal/getproviders"
	"github.com/hashicorp/terraform/internal/rpcapi/terraform1"
	"google.golang.org/protobuf/testing/protocmp"
)

func TestDependenciesOpenCloseSourceBundle(t *testing.T) {
	ctx := context.Background()

	handles := newHandleTable()
	depsServer := newDependenciesServer(handles)

	openResp, err := depsServer.OpenSourceBundle(ctx, &terraform1.OpenSourceBundle_Request{
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

	_, err = depsServer.CloseSourceBundle(ctx, &terraform1.CloseSourceBundle_Request{
		SourceBundleHandle: openResp.SourceBundleHandle,
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestDependencyLocks(t *testing.T) {
	ctx := context.Background()

	handles := newHandleTable()
	depsServer := newDependenciesServer(handles)

	openSourcesResp, err := depsServer.OpenSourceBundle(ctx, &terraform1.OpenSourceBundle_Request{
		LocalPath: "testdata/sourcebundle",
	})
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		depsServer.CloseSourceBundle(ctx, &terraform1.CloseSourceBundle_Request{
			SourceBundleHandle: openSourcesResp.SourceBundleHandle,
		})
	}()

	openLocksResp, err := depsServer.OpenDependencyLockFile(ctx, &terraform1.OpenDependencyLockFile_Request{
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

	getProvidersResp, err := depsServer.GetLockedProviderDependencies(ctx, &terraform1.GetLockedProviderDependencies_Request{
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

	_, err = depsServer.CloseDependencyLocks(ctx, &terraform1.CloseDependencyLocks_Request{
		DependencyLocksHandle: openLocksResp.DependencyLocksHandle,
	})
	if err != nil {
		t.Fatal(err)
	}
}
