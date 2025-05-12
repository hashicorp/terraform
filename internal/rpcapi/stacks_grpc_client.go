// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package rpcapi

import (
	"context"
	"fmt"
	"io"
	"log"

	"github.com/hashicorp/go-plugin"
	"github.com/hashicorp/terraform-svchost/disco"
	"github.com/hashicorp/terraform/internal/pluginshared"

	"github.com/hashicorp/terraform/internal/rpcapi/dynrpcserver"
	"github.com/hashicorp/terraform/internal/rpcapi/terraform1/dependencies"
	"github.com/hashicorp/terraform/internal/rpcapi/terraform1/packages"
	"github.com/hashicorp/terraform/internal/rpcapi/terraform1/stacks"
	"github.com/hashicorp/terraform/internal/stacksplugin/stacksproto1"
	"google.golang.org/grpc"
)

// GRPCStacksClient is the client interface for interacting with terraform-stacksplugin
type GRPCStacksClient struct {
	Client   stacksproto1.CommandServiceClient
	Broker   *plugin.GRPCBroker
	Services *disco.Disco
	Context  context.Context
}

// Proof that GRPCStacksClient fulfills the go-plugin interface
var _ pluginshared.CustomPluginClient = GRPCStacksClient{}

type brokerIDs struct {
	packagesBrokerID     uint32
	dependenciesBrokerID uint32
	stacksBrokerID       uint32
}

// registerBrokers starts the GRPC servers for the dependencies, packages, and stacks
// services and returns the broker IDs for each.
func (c GRPCStacksClient) registerBrokers(stdout, stderr io.Writer) brokerIDs {
	handles := newHandleTable()

	dependenciesServer := dynrpcserver.NewDependenciesStub()
	packagesServer := dynrpcserver.NewPackagesStub()
	stacksServer := dynrpcserver.NewStacksStub()

	// Create channels to signal when each service is ready
	dependenciesReady := make(chan struct{})
	packagesReady := make(chan struct{})
	stacksReady := make(chan struct{})

	dependenciesServerFunc := func(opts []grpc.ServerOption) *grpc.Server {
		s := grpc.NewServer(opts...)
		dependencies.RegisterDependenciesServer(s, dependenciesServer)
		dependenciesServer.ActivateRPCServer(newDependenciesServer(handles, c.Services))
		close(dependenciesReady) // Signal that this service is ready

		return s
	}

	dependenciesBrokerID := c.Broker.NextId()
	go c.Broker.AcceptAndServe(dependenciesBrokerID, dependenciesServerFunc)

	packagesServerFunc := func(opts []grpc.ServerOption) *grpc.Server {
		s := grpc.NewServer(opts...)
		packages.RegisterPackagesServer(s, packagesServer)
		packagesServer.ActivateRPCServer(newPackagesServer(c.Services))
		close(packagesReady) // Signal that this service is ready

		return s
	}

	packagesBrokerID := c.Broker.NextId()
	go c.Broker.AcceptAndServe(packagesBrokerID, packagesServerFunc)

	stacksServerFunc := func(opts []grpc.ServerOption) *grpc.Server {
		s := grpc.NewServer(opts...)
		stacks.RegisterStacksServer(s, stacksServer)
		stacksServer.ActivateRPCServer(newStacksServer(
			newStopper(), handles, c.Services, &serviceOpts{experimentsAllowed: true}))

		close(stacksReady) // Signal that this service is ready
		return s
	}

	stacksBrokerID := c.Broker.NextId()
	go c.Broker.AcceptAndServe(stacksBrokerID, stacksServerFunc)

	// Wait for all services to be ready
	<-dependenciesReady
	<-packagesReady
	<-stacksReady

	return brokerIDs{
		dependenciesBrokerID: dependenciesBrokerID,
		packagesBrokerID:     packagesBrokerID,
		stacksBrokerID:       stacksBrokerID,
	}
}

func logServerInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	log.Printf("[TRACE] Received request: %s", info.FullMethod)
	resp, err := handler(ctx, req)
	if err != nil {
		log.Printf("[ERROR] Handler error for %s: %v", info.FullMethod, err)
	}
	return resp, err
}

func logStreamInterceptor(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	log.Printf("[TRACE] Started streaming: %s", info.FullMethod)
	err := handler(srv, ss)
	if err != nil {
		log.Printf("[ERROR] Stream handler error for %s: %v", info.FullMethod, err)
	}
	return err
}

// Execute sends the client Execute request and waits for the plugin to return
// an exit code response before returning
func (c GRPCStacksClient) executeWithBrokers(brokerIDs brokerIDs, args []string, stdout, stderr io.Writer) int {
	client, err := c.Client.Execute(c.Context, &stacksproto1.CommandRequest{
		DependenciesServer: brokerIDs.dependenciesBrokerID,
		PackagesServer:     brokerIDs.packagesBrokerID,
		StacksServer:       brokerIDs.stacksBrokerID,
		Args:               args,
	})

	if err != nil {
		fmt.Fprint(stderr, err.Error())
		return 1
	}

	for {
		// stacksplugin streams output as multiple CommandResponse value. Each
		// value will either contain stdout bytes, stderr bytes, or an exit code.
		response, err := client.Recv()
		if err == io.EOF {
			log.Print("[DEBUG] received EOF from stacksplugin")
			break
		} else if err != nil {
			fmt.Fprintf(stderr, "Failed to receive command response from stacksplugin: %s", err)
			return 1
		}

		if bytes := response.GetStdout(); len(bytes) > 0 {
			written, err := fmt.Fprint(stdout, string(bytes))
			if err != nil {
				log.Printf("[ERROR] Failed to write stacksplugin output to stdout: %s", err)
				return 1
			}
			if written != len(bytes) {
				log.Printf("[ERROR] Wrote %d bytes to stdout but expected to write %d", written, len(bytes))
			}
		} else if bytes := response.GetStderr(); len(bytes) > 0 {
			written, err := fmt.Fprint(stderr, string(bytes))
			if err != nil {
				log.Printf("[ERROR] Failed to write stacksplugin output to stderr: %s", err)
				return 1
			}
			if written != len(bytes) {
				log.Printf("[ERROR] Wrote %d bytes to stdout but expected to write %d", written, len(bytes))
			}
		} else {
			exitCode := response.GetExitCode()
			log.Printf("[TRACE] received exit code: %d", exitCode)
			if exitCode < 0 || exitCode > 255 {
				log.Printf("[ERROR] stacksplugin returned an invalid error code %d", exitCode)
				return 255
			}
			return int(exitCode)
		}
	}

	// This should indicate a bug in the plugin
	fmt.Fprint(stderr, "stacksplugin exited without responding with an error code")
	return 1
}

// Execute sends the client Execute request and waits for the plugin to return
// an exit code response before returning
func (c GRPCStacksClient) Execute(args []string, stdout, stderr io.Writer) int {
	brokerIDs := c.registerBrokers(stdout, stderr)
	return c.executeWithBrokers(brokerIDs, args, stdout, stderr)
}
