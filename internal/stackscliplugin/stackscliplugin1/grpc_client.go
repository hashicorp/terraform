// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackscliplugin1

import (
	"context"
	"fmt"
	"io"
	"log"

	"github.com/hashicorp/terraform/internal/stackscliplugin"
	"github.com/hashicorp/terraform/internal/stackscliplugin/stackscliproto1"
)

// GRPCStacksCLIClient is the client interface for interacting with stacks-cli-plugin
type GRPCStacksCLIClient struct {
	client  stackscliproto1.CommandServiceClient
	context context.Context
}

// Proof that GRPCStacksCLIClient fulfills the go-plugin interface
var _ stackscliplugin.StacksCLI1 = GRPCStacksCLIClient{}

// Execute sends the client Execute request and waits for the plugin to return
// an exit code response before returning
func (c GRPCStacksCLIClient) Execute(args []string, stdout, stderr io.Writer) int {
	client, err := c.client.Execute(c.context, &stackscliproto1.CommandRequest{
		Args: args,
	})

	if err != nil {
		fmt.Fprint(stderr, err.Error())
		return 1
	}

	for {
		// stackscliplugin streams output as multiple CommandResponse value. Each
		// value will either contain stdout bytes, stderr bytes, or an exit code.
		response, err := client.Recv()
		if err == io.EOF {
			log.Print("[DEBUG] received EOF from stackscliplugin")
			break
		} else if err != nil {
			fmt.Fprintf(stderr, "Failed to receive command response from stackscliplugin: %s", err)
			return 1
		}

		if bytes := response.GetStdout(); len(bytes) > 0 {
			written, err := fmt.Fprint(stdout, string(bytes))
			if err != nil {
				log.Printf("[ERROR] Failed to write stackscliplugin output to stdout: %s", err)
				return 1
			}
			if written != len(bytes) {
				log.Printf("[ERROR] Wrote %d bytes to stdout but expected to write %d", written, len(bytes))
			}
		} else if bytes := response.GetStderr(); len(bytes) > 0 {
			written, err := fmt.Fprint(stderr, string(bytes))
			if err != nil {
				log.Printf("[ERROR] Failed to write stackscliplugin output to stderr: %s", err)
				return 1
			}
			if written != len(bytes) {
				log.Printf("[ERROR] Wrote %d bytes to stdout but expected to write %d", written, len(bytes))
			}
		} else {
			exitCode := response.GetExitCode()
			log.Printf("[TRACE] received exit code: %d", exitCode)
			if exitCode < 0 || exitCode > 255 {
				log.Printf("[ERROR] stackscliplugin returned an invalid error code %d", exitCode)
				return 255
			}
			return int(exitCode)
		}
	}

	// This should indicate a bug in the plugin
	fmt.Fprint(stderr, "stackscliplugin exited without responding with an error code")
	return 1
}
