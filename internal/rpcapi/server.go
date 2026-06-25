// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package rpcapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/hashicorp/go-plugin"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
)

// ServePlugin attempts to complete the go-plugin protocol handshake, and then
// if successful starts the plugin server and blocks until the given context
// is cancelled.
//
// Returns [ErrNotPluginClient] if this program doesn't seem to be running as
// the child of a plugin client, which is detected based on a magic environment
// variable that the client ought to have set.
func ServePlugin(ctx context.Context, opts ServerOpts) error {
	// go-plugin has its own check for the environment variable magic cookie
	// but it returns a generic error message. We'll pre-check it out here
	// instead so we can return a more specific error message.
	if os.Getenv(handshake.MagicCookieKey) != handshake.MagicCookieValue {
		return ErrNotPluginClient
	}

	var debugging bool
	var serveTestCfg *plugin.ServeTestConfig
	debugCh := make(chan *plugin.ReattachConfig)
	debugCloseCh := make(chan struct{})

	// This could also be a flag, but env variable is easier to setup for now
	if os.Getenv("TF_RPCAPI_DEBUG") != "" {
		debugCtx, cancel := context.WithCancel(context.Background())
		signalCh := make(chan os.Signal, 1)

		signal.Notify(signalCh, []os.Signal{os.Interrupt}...)

		defer func() {
			signal.Stop(signalCh)
			cancel()
		}()

		go func() {
			select {
			case <-signalCh:
				cancel()
			case <-ctx.Done():
			}
		}()

		debugging = true
		serveTestCfg = &plugin.ServeTestConfig{
			Context:          debugCtx,
			ReattachConfigCh: debugCh,
			CloseCh:          debugCloseCh,
		}
	}

	srvCfg := &plugin.ServeConfig{
		HandshakeConfig: handshake,
		VersionedPlugins: map[int]plugin.PluginSet{
			1: {
				"tfcore": &corePlugin{
					debugging:          debugging,
					experimentsAllowed: opts.ExperimentsAllowed,
				},
			},
		},
		GRPCServer: func(opts []grpc.ServerOption) *grpc.Server {
			fullOpts := []grpc.ServerOption{
				grpc.StatsHandler(otelgrpc.NewServerHandler()),
			}
			fullOpts = append(fullOpts, opts...)
			server := grpc.NewServer(fullOpts...)
			// We'll also monitor the given context for cancellation
			// and terminate the server gracefully if we get cancelled.
			go func() {
				<-ctx.Done()
				server.GracefulStop()
				// The above will block until all of the pending RPCs have
				// finished.
				os.Exit(0)
			}()
			return server
		},
		Test: serveTestCfg,
	}

	if !debugging {
		plugin.Serve(srvCfg)
		return nil
	}

	go plugin.Serve(srvCfg)

	var pluginReattachConfig *plugin.ReattachConfig
	select {
	case pluginReattachConfig = <-debugCh:
	case <-time.After(10 * time.Second):
		return errors.New("timeout waiting on reattach configuration")
	}

	if pluginReattachConfig == nil {
		return errors.New("nil reattach configuration received")
	}
	type reattachConfigAddr struct {
		Network string
		String  string
	}

	type reattachConfig struct {
		Protocol        string
		ProtocolVersion int
		Pid             int
		Test            bool
		Addr            reattachConfigAddr
	}

	reattachBytes, err := json.Marshal(reattachConfig{
		Protocol:        string(pluginReattachConfig.Protocol),
		ProtocolVersion: pluginReattachConfig.ProtocolVersion,
		Pid:             pluginReattachConfig.Pid,
		Test:            pluginReattachConfig.Test,
		Addr: reattachConfigAddr{
			Network: pluginReattachConfig.Addr.Network(),
			String:  pluginReattachConfig.Addr.String(),
		},
	})

	if err != nil {
		return fmt.Errorf("Error building reattach string: %w", err)
	}

	fmt.Printf("\t%s='%s'\n", "TF_RPCAPI_REATTACH", strings.ReplaceAll(string(reattachBytes), `'`, `'"'"'`))
	<-debugCloseCh

	return nil
}

var ErrNotPluginClient = errors.New("caller is not a plugin client")

type ServerOpts struct {
	ExperimentsAllowed bool
}

// handshake is the HandshakeConfig used to begin negotiation between client
// and server.
var handshake = plugin.HandshakeConfig{
	// The ProtocolVersion is the version that must match between TF core
	// and TF plugins.
	ProtocolVersion: 1,

	// The magic cookie values should NEVER be changed.
	MagicCookieKey:   "TERRAFORM_RPCAPI_COOKIE",
	MagicCookieValue: "fba0991c9bcd453982f0d88e2da95940",
}
