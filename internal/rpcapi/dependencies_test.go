package rpcapi

import (
	"context"
	"testing"

	"github.com/hashicorp/go-slug/sourceaddrs"
	"github.com/hashicorp/go-slug/sourcebundle"
	"github.com/hashicorp/terraform/internal/rpcapi/terraform1"
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
