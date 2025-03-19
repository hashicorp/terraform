// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stacksplugin1

import (
	"context"
	"errors"
	"fmt"
	"net/rpc"

	"github.com/hashicorp/go-plugin"
	svchost "github.com/hashicorp/terraform-svchost"
	"github.com/hashicorp/terraform-svchost/auth"
	"github.com/hashicorp/terraform-svchost/disco"
	"github.com/hashicorp/terraform/internal/command/cliconfig"
	pluginDiscovery "github.com/hashicorp/terraform/internal/plugin/discovery"
	"github.com/hashicorp/terraform/internal/stacksplugin/dynrpcserver"
	"github.com/hashicorp/terraform/internal/stacksplugin/stacksproto1"
	"github.com/hashicorp/terraform/internal/stacksplugin/stacksproto1/dependencies"
	"github.com/hashicorp/terraform/internal/stacksplugin/stacksproto1/packages"
	"github.com/hashicorp/terraform/internal/stacksplugin/stacksproto1/setup"
	"github.com/hashicorp/terraform/internal/stacksplugin/stacksproto1/stacks"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// GRPCCloudPlugin is the go-plugin implementation, but only the client
// implementation exists in this package.
type GRPCStacksPlugin struct {
	plugin.GRPCPlugin
	Metadata metadata.MD
}

// Server always returns an error; we're only implementing the GRPCPlugin
// interface, not the Plugin interface.
func (p *GRPCStacksPlugin) Server(*plugin.MuxBroker) (interface{}, error) {
	return nil, errors.New("cloudplugin only implements gRPC clients")
}

// Client always returns an error; we're only implementing the GRPCPlugin
// interface, not the Plugin interface.
func (p *GRPCStacksPlugin) Client(*plugin.MuxBroker, *rpc.Client) (interface{}, error) {
	return nil, errors.New("cloudplugin only implements gRPC clients")
}

// GRPCClient returns a new GRPC client for interacting with the cloud plugin server.
func (p *GRPCStacksPlugin) GRPCClient(ctx context.Context, broker *plugin.GRPCBroker, c *grpc.ClientConn) (interface{}, error) {
	ctx = metadata.NewOutgoingContext(ctx, p.Metadata)
	return &GRPCStacksClient{
		client:  stacksproto1.NewCommandServiceClient(c),
		context: ctx,
	}, nil
}

// GRPCServer always returns an error; we're only implementing the client
// interface, not the server.
func (p *GRPCStacksPlugin) GRPCServer(broker *plugin.GRPCBroker, s *grpc.Server) error {

	return nil
}

func SetupStacksPluginServices() (dependenciesServer, stacksServer, packagesServer) {
	handles := newHandleTable()
	dependenciesServer := newDependenciesServer(handles)
	packagesServer := newPackagesServer(handles)
	stacksServer := newStacksServer(handles)
	return dependenciesServer, stacksServer, packagesServer
}

func registerGRPCServices(s *grpc.Server, opts *serviceOpts) {
	// We initially only register the setup server, because the registration
	// of other services can vary depending on the capabilities negotiated
	// during handshake.
	server := newSetupServer(serverHandshake(s, opts))
	setup.RegisterSetupServer(s, server)
}

func serverHandshake(s *grpc.Server, opts *serviceOpts) func(context.Context, *setup.Handshake_Request, *stopper) (*setup.ServerCapabilities, error) {
	dependenciesStub := dynrpcserver.NewDependenciesStub()
	dependencies.RegisterDependenciesServer(s, dependenciesStub)
	stacksStub := dynrpcserver.NewStacksStub()
	stacks.RegisterStacksServer(s, stacksStub)
	packagesStub := dynrpcserver.NewPackagesStub()
	packages.RegisterPackagesServer(s, packagesStub)

	return func(ctx context.Context, request *setup.Handshake_Request, stopper *stopper) (*setup.ServerCapabilities, error) {
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
		services, err := newServiceDisco(request.GetConfig())
		if err != nil {
			return &setup.ServerCapabilities{}, err
		}

		// If handshaking is successful (which it currently always is, because
		// we don't have any special capabilities to negotiate yet) then we
		// will initialize all of the other services so the client can begin
		// doing real work. In future the details of what we register here
		// might vary based on the negotiated capabilities.
		dependenciesStub.ActivateRPCServer(newDependenciesServer(handles, services))
		stacksStub.ActivateRPCServer(newStacksServer(stopper, handles))
		packagesStub.ActivateRPCServer(newPackagesServer(services))

		// If the client requested any extra capabililties that we're going
		// to honor then we should announce them in this result.
		return &setup.ServerCapabilities{}, nil
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

func newServiceDisco(config *setup.Config) (*disco.Disco, error) {
	// First, we'll try and load any credentials that might have been available
	// to the UI. It's perfectly fine if there are none so any errors we find
	// are from malformed credentials rather than missing ones.

	file, diags := cliconfig.LoadConfig()
	if diags.HasErrors() {
		return nil, fmt.Errorf("problem loading CLI configuration: %w", diags.ErrWithWarnings())
	}

	helperPlugins := pluginDiscovery.FindPlugins("credentials", cliconfig.GlobalPluginDirs())
	src, err := file.CredentialsSource(helperPlugins)
	if err != nil {
		return nil, fmt.Errorf("problem creating credentials source: %w", err)
	}
	services := disco.NewWithCredentialsSource(src)

	// Second, we'll side-load any credentials that might have been passed in.

	credSrc := services.CredentialsSource()
	if config != nil {
		for host, cred := range config.GetCredentials() {
			if err := credSrc.StoreForHost(svchost.Hostname(host), auth.HostCredentialsToken(cred.Token)); err != nil {
				return nil, fmt.Errorf("problem storing credential for host %s with: %w", host, err)
			}
		}
		services.SetCredentialsSource(credSrc)
	}

	return services, nil
}
