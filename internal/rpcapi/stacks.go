package rpcapi

import (
	"context"

	"github.com/hashicorp/go-slug/sourcebundle"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/hashicorp/terraform/internal/rpcapi/terraform1"
	"github.com/hashicorp/terraform/internal/stacks/stackconfig"
)

type stacksServer struct {
	terraform1.UnimplementedStacksServer

	handles *handleTable
}

var _ terraform1.StacksServer = (*stacksServer)(nil)

func newStacksServer(handles *handleTable) *stacksServer {
	return &stacksServer{
		handles: handles,
	}
}

func (s *stacksServer) OpenStackConfiguration(ctx context.Context, req *terraform1.OpenStackConfiguration_Request) (*terraform1.OpenStackConfiguration_Response, error) {
	sourcesHnd := handle[*sourcebundle.Bundle](req.SourceBundleHandle)
	sources := s.handles.SourceBundle(sourcesHnd)
	if sources == nil {
		return nil, status.Error(codes.InvalidArgument, "the given source bundle handle is invalid")
	}

	sourceAddr, err := resolveFinalSourceAddr(req.SourceAddress, sources)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid source address: %s", err)
	}

	config, diags := stackconfig.LoadConfigDir(sourceAddr, sources)
	if diags.HasErrors() {
		// For errors in the configuration itself we treat that as a successful
		// result from OpenStackConfiguration but with diagnostics in the
		// response and no source handle.
		return &terraform1.OpenStackConfiguration_Response{
			Diagnostics: diagnosticsToProto(diags),
		}, nil
	}

	configHnd, err := s.handles.NewStackConfig(config, sourcesHnd)
	if err != nil {
		// The only reasonable way we can fail here is if the caller made
		// a concurrent call to Dependencies.CloseSourceBundle after we
		// checked the handle validity above. That'd be a very strange thing
		// to do, but in the event it happens we'll just discard the config
		// we loaded (since its source files on disk might be gone imminently)
		// and return an error.
		return nil, status.Errorf(codes.Unknown, "allocating config handle: %s", err)
	}

	// If we get here then we're guaranteed that the source bundle handle
	// cannot be closed until the config handle is closed -- enforced by
	// [handleTable]'s dependency tracking -- and so we can return the config
	// handle. (The caller is required to ensure that the source bundle files
	// on disk are not modified for as long as the source bundle handle remains
	// open, and its lifetime will necessarily exceed the config handle.)
	return &terraform1.OpenStackConfiguration_Response{
		StackConfigHandle: configHnd.ForProtobuf(),
		Diagnostics:       diagnosticsToProto(diags),
	}, nil
}

func (s *stacksServer) CloseStackConfiguration(ctx context.Context, req *terraform1.CloseStackConfiguration_Request) (*terraform1.CloseStackConfiguration_Response, error) {
	hnd := handle[*stackconfig.Config](req.StackConfigHandle)
	err := s.handles.CloseStackConfig(hnd)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	return &terraform1.CloseStackConfiguration_Response{}, nil
}
