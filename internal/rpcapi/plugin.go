// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package rpcapi

import (
	"context"
	"fmt"

	"github.com/hashicorp/go-plugin"
	"github.com/hashicorp/terraform-svchost/disco"
	"google.golang.org/grpc"

	"github.com/hashicorp/terraform/internal/rpcapi/dynrpcserver"
	"github.com/hashicorp/terraform/internal/rpcapi/terraform1"
)

type corePlugin struct {
	plugin.Plugin

	experimentsAllowed bool
}

func (p *corePlugin) GRPCClient(ctx context.Context, broker *plugin.GRPCBroker, c *grpc.ClientConn) (interface{}, error) {
	// This codebase only provides a server implementation of this plugin.
	// Clients must live elsewhere.
	return nil, fmt.Errorf("there is no client implementation in this codebase")
}

func (p *corePlugin) GRPCServer(broker *plugin.GRPCBroker, s *grpc.Server) error {
	generalOpts := &serviceOpts{
		experimentsAllowed: p.experimentsAllowed,
	}
	registerGRPCServices(s, generalOpts)
	return nil
}

func registerGRPCServices(s *grpc.Server, opts *serviceOpts) {
	// We initially only register the setup server, because the registration
	// of other services can vary depending on the capabilities negotiated
	// during handshake.
	setup := newSetupServer(serverHandshake(s, opts))
	terraform1.RegisterSetupServer(s, setup)
}

func serverHandshake(s *grpc.Server, opts *serviceOpts) func(context.Context, *terraform1.ClientCapabilities) (*terraform1.ServerCapabilities, error) {
	dependencies := dynrpcserver.NewDependenciesStub()
	terraform1.RegisterDependenciesServer(s, dependencies)
	stacks := dynrpcserver.NewStacksStub()
	terraform1.RegisterStacksServer(s, stacks)
	packages := dynrpcserver.NewPackagesStub()
	terraform1.RegisterPackagesServer(s, packages)

	return func(ctx context.Context, clientCaps *terraform1.ClientCapabilities) (*terraform1.ServerCapabilities, error) {
		// All of our servers will share a common handles table so that objects
		// can be passed from one service to another.
		handles := newHandleTable()

		// NOTE: This is intentionally not the same disco that "package main"
		// instantiates for Terraform CLI, because the RPC API is
		// architecturally independent from CLI despite being launched through
		// it, and so it is not subject to any ambient CLI configuration files
		// that might be in scope. If we later discover requirements for
		// callers to customize the service discovery settings, consider
		// adding new fields to terraform1.ClientCapabilities (even though
		// this isn't strictly a "capability") so that the RPC caller has
		// full control without needing to also tinker with the current user's
		// CLI configuration.
		services := disco.New()

		// If handshaking is successful (which it currently always is, because
		// we don't have any special capabilities to negotiate yet) then we
		// will initialize all of the other services so the client can begin
		// doing real work. In future the details of what we register here
		// might vary based on the negotiated capabilities.
		dependencies.ActivateRPCServer(newDependenciesServer(handles, services))
		stacks.ActivateRPCServer(newStacksServer(handles, opts))
		packages.ActivateRPCServer(newPackagesServer(services))

		// If the client requested any extra capabililties that we're going
		// to honor then we should announce them in this result.
		return &terraform1.ServerCapabilities{}, nil
	}
}

// serviceOpts are options that could potentially apply to all of our
// individual RPC services.
//
// This could potentially be embedded inside a service-specific options
// structure, if needed.
type serviceOpts struct {
	experimentsAllowed bool
}
