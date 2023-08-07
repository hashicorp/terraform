// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package cloudplugin1

import (
	"context"
	"fmt"
	"io"
	"log"

	"github.com/hashicorp/terraform/internal/cloudplugin"
	"github.com/hashicorp/terraform/internal/cloudplugin/cloudproto1"
)

// GRPCCloudClient is the client interface for interacting with terraform-cloudplugin
type GRPCCloudClient struct {
	client  cloudproto1.CommandServiceClient
	context context.Context
}

// Proof that GRPCCloudClient fulfills the go-plugin interface
var _ cloudplugin.Cloud1 = GRPCCloudClient{}

// Execute sends the client Execute request and waits for the plugin to return
// an exit code response before returning
func (c GRPCCloudClient) Execute(args []string, stdout, stderr io.Writer) int {
	client, err := c.client.Execute(c.context, &cloudproto1.CommandRequest{
		Args: args,
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
			_, err := fmt.Fprint(stdout, string(bytes))
			if err != nil {
				log.Printf("[ERROR] Failed to write cloudplugin output to stdout: %s", err)
				return 1
			}
		} else if bytes := response.GetStderr(); len(bytes) > 0 {
			fmt.Fprint(stderr, string(bytes))
			if err != nil {
				log.Printf("[ERROR] Failed to write cloudplugin output to stderr: %s", err)
				return 1
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
