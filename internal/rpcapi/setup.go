// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package rpcapi

import (
	"context"
	"sync"

	"github.com/hashicorp/terraform/internal/rpcapi/terraform1/setup"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// setupServer is an implementation of the "Setup" service defined in our
// terraform1 package.
//
// This service is here mainly to offer the "Handshake" function, which clients
// must call to negotiate access to any other services. This is really just
// an adapter around a handshake function implemented on [corePlugin].
type setupServer struct {
	setup.UnimplementedSetupServer

	// initOthers is the callback used to perform the capability negotiation
	// step and initialize all of the other API services based on what was
	// negotiated.
	initOthers func(context.Context, *setup.Handshake_Request, *stopper) (*setup.ServerCapabilities, error)

	// stopper is used to track and stop long-running operations when the Stop
	// RPC is called.
	stopper *stopper

	mu sync.Mutex
}

func newSetupServer(initOthers func(context.Context, *setup.Handshake_Request, *stopper) (*setup.ServerCapabilities, error)) setup.SetupServer {
	return &setupServer{
		initOthers: initOthers,
		stopper:    newStopper(),
	}
}

func (s *setupServer) Handshake(ctx context.Context, req *setup.Handshake_Request) (*setup.Handshake_Response, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.initOthers == nil {
		return nil, status.Error(codes.FailedPrecondition, "handshake already completed")
	}

	var serverCaps *setup.ServerCapabilities
	var err error
	{
		ctx, span := tracer.Start(ctx, "initialize RPC services")
		serverCaps, err = s.initOthers(ctx, req, s.stopper)
		span.End()
	}
	s.initOthers = nil // cannot handshake again
	if err != nil {
		return nil, err
	}
	return &setup.Handshake_Response{
		Capabilities: serverCaps,
	}, nil
}

func (s *setupServer) Stop(ctx context.Context, req *setup.Stop_Request) (*setup.Stop_Response, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.stopper.stop()

	return &setup.Stop_Response{}, nil
}
