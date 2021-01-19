package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"
	"github.com/mitchellh/cli"
	"google.golang.org/grpc"

	"github.com/hashicorp/terraform/internal/stresstest/internal/stressprovider"
	tfPlugin "github.com/hashicorp/terraform/plugin"
)

// terraformCommand implements the "stresstest terraform" command, which is
// a helper wrapper around the normal Terraform CLI which arranges for the
// "stressful" provider to be available so that we can work with exported
// configuration series.
type terraformCommand struct {
}

var _ cli.Command = (*graphCommand)(nil)

func (c *terraformCommand) Run(args []string) int {
	cmd := exec.Command("terraform", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	attachConfig, close, err := runStressfulProvider()
	defer close()
	if err != nil {
		log.Printf("failed to start the stressful provider: %s", err)
		return 1
	}

	reattachForJSON := map[string]interface{}{
		"terraform.io/stresstest/stressful": map[string]interface{}{
			"Protocol": attachConfig.Protocol,
			"Addr": map[string]interface{}{
				"Network": attachConfig.Addr.Network(),
				"String":  attachConfig.Addr.String(),
			},
			"Pid":  attachConfig.Pid,
			"Test": true,
		},
	}
	reattachJSON, err := json.Marshal(reattachForJSON)
	if err != nil {
		log.Printf("failed to encode provider reattach config: %s", err)
		return 1
	}
	cmd.Env = append(
		os.Environ(),
		"TF_REATTACH_PROVIDERS="+string(reattachJSON),
	)

	err = cmd.Run()
	if err != nil {
		if exitCode := cmd.ProcessState.ExitCode(); exitCode != 0 {
			return exitCode
		}
		log.Printf("failed to run Terraform: %s", err)
		return 128
	}

	return cmd.ProcessState.ExitCode()
}

func (c *terraformCommand) Synopsis() string {
	return "Run Terraform CLI with the fake provider"
}

func (c *terraformCommand) Help() string {
	return strings.TrimSpace(`
Usage: stresstest terraform [terraform arguments...]

...
`)
}

// runStressfulProvider starts an RPC server for the stressful provider in the
// background and returns the details about how to connect to it, along with
// a function that the caller should call once it's finished with the
// stressful provider in order to properly shut it down.
func runStressfulProvider() (config *plugin.ReattachConfig, close func(), err error) {
	ctx, close := context.WithCancel(context.Background())
	provider := stressprovider.New()

	// TODO: Also look for a remote-objects directory in the current directory,
	// and if it's present then preload the provider with the objects described
	// inside, to make sure that we can reproduce second and subsequent
	// steps of a series without detecting all of the objects as having been
	// deleted.
	// Likewise, it would be useful for the "close" callback to also try to
	// serialize the provider's remote objects back into the remote-objects
	// directory, so it'll stay synchronized with any partial updates made
	// while debugging.
	// For now, while just prototyping anyway, we're ignoring this and focusing
	// mainly on interactive-debugging the _first_ step in a series.

	reattachCh := make(chan *plugin.ReattachConfig)
	serveConfig := &plugin.ServeConfig{
		Logger:          hclog.NewNullLogger(), // be quiet so we don't interfere with Terraform's output
		HandshakeConfig: tfPlugin.Handshake,
		VersionedPlugins: map[int]plugin.PluginSet{
			5: {
				"provider": provider.Plugin(),
			},
		},
		GRPCServer: func(opts []grpc.ServerOption) *grpc.Server {
			return grpc.NewServer(opts...)
		},
		Test: &plugin.ServeTestConfig{
			Context:          ctx,
			ReattachConfigCh: reattachCh,
		},
	}

	go func() {
		plugin.Serve(serveConfig)
	}()

	select {
	case config = <-reattachCh:
	case <-time.After(5 * time.Second):
		return nil, close, errors.New("timeout waiting for attach config")
	}

	return config, close, nil
}
