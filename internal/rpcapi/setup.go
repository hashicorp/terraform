// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package rpcapi

import (
	"context"
	"sync"

	"github.com/hashicorp/terraform/internal/rpcapi/terraform1"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// setupServer is an implementation of the "Setup" service defined in our
// terraform1 package.
//
// This service is here only to offer the "Handshake" function, which clients
// must call to negotiate access to any other services. This is really just
// an adapter around a handshake function implemented on [corePlugin].
type setupServer struct {
	terraform1.UnimplementedSetupServer

	// initOthers is the callback used to perform the capability negotiation
	// step and initialize all of the other API services based on what was
	// negotiated.
	initOthers func(context.Context, *terraform1.Handshake_Request) (*terraform1.ServerCapabilities, error)
	mu         sync.Mutex
}

func newSetupServer(initOthers func(context.Context, *terraform1.Handshake_Request) (*terraform1.ServerCapabilities, error)) terraform1.SetupServer {
	return &setupServer{
		initOthers: initOthers,
	}
}

func (s *setupServer) Handshake(ctx context.Context, req *terraform1.Handshake_Request) (*terraform1.Handshake_Response, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.initOthers == nil {
		return nil, status.Error(codes.FailedPrecondition, "handshake already completed")
	}

	var serverCaps *terraform1.ServerCapabilities
	var err error
	{
		ctx, span := tracer.Start(ctx, "initialize RPC services")
		serverCaps, err = s.initOthers(ctx, req)
		span.End()
	}
	s.initOthers = nil // cannot handshake again
	if err != nil {
		return nil, err
	}
	return &terraform1.Handshake_Response{
		Capabilities: serverCaps,
	}, nil
}
