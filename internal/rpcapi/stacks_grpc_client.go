// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package rpcapi

import (
	"context"
	"fmt"
	"io"
	"log"
	"sync"

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
	Client     stacksproto1.CommandServiceClient
	Broker     *plugin.GRPCBroker
	Services   *disco.Disco
	Context    context.Context
	ShutdownCh <-chan struct{}
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

	var serverWG sync.WaitGroup
	// wait for all 3 servers to start
	serverWG.Add(3)

	dependenciesServerFunc := func(opts []grpc.ServerOption) *grpc.Server {
		s := grpc.NewServer(opts...)
		dependencies.RegisterDependenciesServer(s, dependenciesServer)
		dependenciesServer.ActivateRPCServer(newDependenciesServer(handles, c.Services))

		serverWG.Done()
		return s
	}

	dependenciesBrokerID := c.Broker.NextId()
	go c.Broker.AcceptAndServe(dependenciesBrokerID, dependenciesServerFunc)

	packagesServerFunc := func(opts []grpc.ServerOption) *grpc.Server {
		s := grpc.NewServer(opts...)
		packages.RegisterPackagesServer(s, packagesServer)
		packagesServer.ActivateRPCServer(newPackagesServer(c.Services))

		serverWG.Done()
		return s
	}

	packagesBrokerID := c.Broker.NextId()
	go c.Broker.AcceptAndServe(packagesBrokerID, packagesServerFunc)

	stacksServerFunc := func(opts []grpc.ServerOption) *grpc.Server {
		s := grpc.NewServer(opts...)
		stacks.RegisterStacksServer(s, stacksServer)
		stacksServer.ActivateRPCServer(newStacksServer(
			newStopper(), handles, c.Services, &serviceOpts{experimentsAllowed: true}))

		serverWG.Done()
		return s
	}

	stacksBrokerID := c.Broker.NextId()
	go c.Broker.AcceptAndServe(stacksBrokerID, stacksServerFunc)

	// block till all 3 servers have signaled readiness
	serverWG.Wait()

	return brokerIDs{
		dependenciesBrokerID: dependenciesBrokerID,
		packagesBrokerID:     packagesBrokerID,
		stacksBrokerID:       stacksBrokerID,
	}
}

// Execute sends the client Execute request and waits for the plugin to return
// an exit code response before returning
func (c GRPCStacksClient) executeWithBrokers(brokerIDs brokerIDs, args []string, stdout, stderr io.Writer) int {
	ctx, cancel := context.WithCancel(c.Context)

	// Monitor for interrupt and cancel the context if received
	go func() {
		sig := <-c.ShutdownCh
		fmt.Print("\n\nOperation Interrupted, any remote operations started will continue\n\n")
		log.Printf("[INFO] Received signal: %s, cancelling operation", sig)
		cancel()
	}()

	client, err := c.Client.Execute(ctx, &stacksproto1.CommandRequest{
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
