// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

// Package dynrpcserver deals with an annoying detail of the rpcapi
// implementation: we need to complete the Setup.Handshake call before we can
// instantiate the remaining services (since their behavior might vary
// depending on negotiated capabilities) but the Go gRPC implementation doesn't
// allow registration of a new service after the gRPC server is already running.
//
// To deal with that we generate forwarding wrappers that initially just
// return errors and then, once a real implementation is provided, just forward
// all requests to the real service.
package dynrpcserver
