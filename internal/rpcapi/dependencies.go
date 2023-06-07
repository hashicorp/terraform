package rpcapi

import (
	"context"
	"path/filepath"

	"github.com/hashicorp/go-slug/sourcebundle"
	"github.com/hashicorp/terraform/internal/rpcapi/terraform1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type dependenciesServer struct {
	terraform1.UnimplementedDependenciesServer

	handles *handleTable
}

func newDependenciesServer(handles *handleTable) *dependenciesServer {
	return &dependenciesServer{
		handles: handles,
	}
}

func (s *dependenciesServer) OpenSourceBundle(ctx context.Context, req *terraform1.OpenSourceBundle_Request) (*terraform1.OpenSourceBundle_Response, error) {
	localDir := filepath.Clean(req.LocalPath)
	sources, err := sourcebundle.OpenDir(localDir)
	if err != nil {
		return nil, status.Error(codes.Unknown, err.Error())
	}
	hnd := s.handles.NewSourceBundle(sources)
	return &terraform1.OpenSourceBundle_Response{
		SourceBundleHandle: hnd.ForProtobuf(),
	}, err
}

func (s *dependenciesServer) CloseSourceBundle(ctx context.Context, req *terraform1.CloseSourceBundle_Request) (*terraform1.CloseSourceBundle_Response, error) {
	hnd := handle[*sourcebundle.Bundle](req.SourceBundleHandle)
	ok := s.handles.CloseSourceBundle(hnd)
	if !ok {
		return nil, status.Error(codes.InvalidArgument, "handle does not match an open source bundle")
	}
	return &terraform1.CloseSourceBundle_Response{}, nil
}
