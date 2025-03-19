// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stacksplugin1

import (
	"context"
	"fmt"
	"io"
	"log"

	"github.com/hashicorp/go-plugin"
	"github.com/hashicorp/terraform/internal/stacksplugin"
	"github.com/hashicorp/terraform/internal/stacksplugin/dynrpcserver"
	"github.com/hashicorp/terraform/internal/stacksplugin/stacksproto1"
	dep "github.com/hashicorp/terraform/internal/stacksplugin/stacksproto1/dependencies"
	pack "github.com/hashicorp/terraform/internal/stacksplugin/stacksproto1/packages"
	stack "github.com/hashicorp/terraform/internal/stacksplugin/stacksproto1/stacks"
	"google.golang.org/grpc"
)

// GRPCCloudClient is the client interface for interacting with terraform-cloudplugin
type GRPCStacksClient struct {
	client  stacksproto1.CommandServiceClient
	broker  *plugin.GRPCBroker
	context context.Context
}

// Proof that GRPCStacksClient fulfills the go-plugin interface
var _ stacksplugin.Stacks1 = GRPCStacksClient{}

// Execute sends the client Execute request and waits for the plugin to return
// an exit code response before returning
func (c GRPCStacksClient) Execute(args []string, stdout, stderr io.Writer) int {
	dependenciesServer := dynrpcserver.NewDependenciesStub()
	packagesServer := dynrpcserver.NewPackagesStub()
	stacksServer := dynrpcserver.NewStacksStub()

	var s *grpc.Server
	dependenciesServerFunc := func(opts []grpc.ServerOption) *grpc.Server {
		s = grpc.NewServer(opts...)
		dep.RegisterDependenciesServer(s, dependenciesServer)

		return s
	}

	dependenciesBrokerID := c.broker.NextId()
	go c.broker.AcceptAndServe(dependenciesBrokerID, dependenciesServerFunc)

	packagesServerFunc := func(opts []grpc.ServerOption) *grpc.Server {
		s = grpc.NewServer(opts...)
		pack.RegisterPackagesServer(s, packagesServer)

		return s
	}

	packagesBrokerID := c.broker.NextId()
	go c.broker.AcceptAndServe(packagesBrokerID, packagesServerFunc)

	stacksServerFunc := func(opts []grpc.ServerOption) *grpc.Server {
		s = grpc.NewServer(opts...)
		stack.RegisterStacksServer(s, stacksServer)

		return s
	}

	stacksBrokerID := c.broker.NextId()
	go c.broker.AcceptAndServe(stacksBrokerID, stacksServerFunc)

	client, err := c.client.Execute(c.context, &stacksproto1.CommandRequest{
		DependenciesServer: dependenciesBrokerID,
		PackagesServer:     packagesBrokerID,
		StacksServer:       stacksBrokerID,
		Args:               args,
	})

	if err != nil {
		fmt.Fprint(stderr, err.Error())
		return 1
	}

	for {
		// cloudplugin streams output as multiple CommandResponse value. Each
		// value will either contain stdout bytes, stderr bytes, or an exit code.
		response, err := client.Recv()
		if err == io.EOF {
			log.Print("[DEBUG] received EOF from cloudplugin")
			break
		} else if err != nil {
			fmt.Fprintf(stderr, "Failed to receive command response from cloudplugin: %s", err)
			return 1
		}

		if bytes := response.GetStdout(); len(bytes) > 0 {
			written, err := fmt.Fprint(stdout, string(bytes))
			if err != nil {
				log.Printf("[ERROR] Failed to write cloudplugin output to stdout: %s", err)
				return 1
			}
			if written != len(bytes) {
				log.Printf("[ERROR] Wrote %d bytes to stdout but expected to write %d", written, len(bytes))
			}
		} else if bytes := response.GetStderr(); len(bytes) > 0 {
			written, err := fmt.Fprint(stderr, string(bytes))
			if err != nil {
				log.Printf("[ERROR] Failed to write cloudplugin output to stderr: %s", err)
				return 1
			}
			if written != len(bytes) {
				log.Printf("[ERROR] Wrote %d bytes to stdout but expected to write %d", written, len(bytes))
			}
		} else {
			exitCode := response.GetExitCode()
			log.Printf("[TRACE] received exit code: %d", exitCode)
			if exitCode < 0 || exitCode > 255 {
				log.Printf("[ERROR] cloudplugin returned an invalid error code %d", exitCode)
				return 255
			}
			return int(exitCode)
		}
	}

	// This should indicate a bug in the plugin
	fmt.Fprint(stderr, "cloudplugin exited without responding with an error code")
	return 1
}
