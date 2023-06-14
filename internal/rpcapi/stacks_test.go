package rpcapi

import (
	"context"
	"testing"

	"github.com/hashicorp/go-slug/sourcebundle"
	"github.com/hashicorp/terraform/internal/rpcapi/terraform1"
	"github.com/hashicorp/terraform/internal/stacks/stackconfig"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestStacksOpenCloseStackConfiguration(t *testing.T) {
	ctx := context.Background()

	handles := newHandleTable()
	stacksServer := newStacksServer(handles)

	// In normal use a client would have previously opened a source bundle
	// using Dependencies.OpenSourceBundle, so we'll simulate the effect
	// of that here.
	var sourcesHnd handle[*sourcebundle.Bundle]
	{
		sources, err := sourcebundle.OpenDir("testdata/sourcebundle")
		if err != nil {
			t.Fatal(err)
		}
		sourcesHnd = handles.NewSourceBundle(sources)
	}

	openResp, err := stacksServer.OpenStackConfiguration(ctx, &terraform1.OpenStackConfiguration_Request{
		SourceBundleHandle: sourcesHnd.ForProtobuf(),
		SourceAddress: &terraform1.SourceAddress{
			Source: "git::https://example.com/foo.git",
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	// A client wouldn't normally be able to interact directly with the
	// stack configuration, but we're doing that here to simulate what would
	// happen in another service that takes configuration handles as input.
	{
		hnd := handle[*stackconfig.Config](openResp.StackConfigHandle)
		cfg := handles.StackConfig(hnd)
		if cfg == nil {
			t.Fatal("returned stack config handle is invalid")
		}
	}

	// A hypothetical attempt to close the underlying source bundle while
	// the stack configuration is active should fail.
	{
		depsServer := newDependenciesServer(handles)

		_, err := depsServer.CloseSourceBundle(ctx, &terraform1.CloseSourceBundle_Request{
			SourceBundleHandle: sourcesHnd.ForProtobuf(),
		})
		if err == nil {
			t.Fatal("successfully closed source bundle while stack config was using it; should have failed to close")
		}
		protoStatus, ok := status.FromError(err)
		if !ok {
			t.Fatal("error is not a protobuf status code")
		}
		if got, want := protoStatus.Code(), codes.InvalidArgument; got != want {
			t.Errorf("wrong error status\ngot:  %s\nwant: %s", got, want)
		}
		if got, want := protoStatus.Message(), "handle is in use by another open handle"; got != want {
			t.Errorf("wrong error message\ngot:  %s\nwant: %s", got, want)
		}
	}

	_, err = stacksServer.CloseStackConfiguration(ctx, &terraform1.CloseStackConfiguration_Request{
		StackConfigHandle: openResp.StackConfigHandle,
	})
	if err != nil {
		t.Fatal(err)
	}

	// Should be able to close the source bundle now too.
	{
		depsServer := newDependenciesServer(handles)

		_, err := depsServer.CloseSourceBundle(ctx, &terraform1.CloseSourceBundle_Request{
			SourceBundleHandle: sourcesHnd.ForProtobuf(),
		})
		if err != nil {
			t.Fatalf("failed to close the source bundle: %s", err.Error())
		}
	}
}
