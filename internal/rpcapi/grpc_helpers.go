// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package rpcapi

import (
	"sync"

	"google.golang.org/protobuf/proto"
)

// This interface should match the interfaces that grpc-gen-go tends to
// generate for a the server side of RPC function which produces streaming
// results with a particular message type.
type grpcServerStreamingSender[Message proto.Message] interface {
	Send(Message) error
}

// syncStreamingRPCSender is a wrapper around a generated gprc.ServerStream
// wrapper that makes Send calls concurrency-safe by holding a mutex throughout
// each call to the underlying Send.
//
// Instantiate this using [newSyncStreamingRPCSender] so you can rely on
// type inference to avoid writing out the type parameters explicitly.
// Consider declaring a type alias with specific Server and Message types if
// you need to name an instantiation of this generic type, so you'll only have
// to write the long-winded instantiation expression once and can use a more
// intuitive name elsewhere.
type syncStreamingRPCSender[
	Server grpcServerStreamingSender[Message],
	Message proto.Message,
] struct {
	wrapped Server
	mu      sync.Mutex
}

// newSyncStreamingRPCSender wraps an interface value implementing an interface
// generated for the server side of a streaming RPC response and makes its
// Send method concurrency-safe, by holding a mutex throughout the call to
// the underlying Send.
func newSyncStreamingRPCSender[
	Server grpcServerStreamingSender[Message],
	Message proto.Message,
](wrapped Server) *syncStreamingRPCSender[Server, Message] {
	return &syncStreamingRPCSender[Server, Message]{
		wrapped: wrapped,
	}
}

// Send holds a mutex while calling Send on the wrapped server, and then
// returns its error value.
func (s *syncStreamingRPCSender[Server, Message]) Send(msg Message) error {
	s.mu.Lock()
	err := s.wrapped.Send(msg)
	s.mu.Unlock()
	return err
}
