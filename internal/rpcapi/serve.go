package rpcapi

import (
	"context"

	"github.com/apparentlymart/go-ctxenv/ctxenv"
	"github.com/hashicorp/terraform/internal/terraform"
	"go.rpcplugin.org/rpcplugin"
)

var handshakeConfig = rpcplugin.HandshakeConfig{
	// This is just an arbitrary key/value pair so that the program
	// launching this process can affirm that it's expecting to talk
	// to an rpcplugin plugin, rather than a normal CLI tool.
	CookieKey:   "TERRAFORM_CORE_RPCPLUGIN_COOKIE",
	CookieValue: "36594bbabbaf5783bbbae2284929a2c9",
}

// Serve starts the rpcplugin server. It does not return unless server startup
// encounters an error.
func Serve(ctx context.Context, opts ServeOpts) error {
	return rpcplugin.Serve(ctx, &rpcplugin.ServerConfig{
		Handshake: handshakeConfig,
		ProtoVersions: map[int]rpcplugin.ServerVersion{
			1: version1{
				getCoreOpts: opts.GetCoreOpts,
			},
		},
	})
}

type ServeOpts struct {
	// GetCoreOpts is a function that the server will call whenever it's
	// about to construct a new Terraform Core instance, in order to get
	// the options to pass to terraform.NewContext.
	GetCoreOpts func() *terraform.ContextOpts
}

// RunningAsPlugin checks the process environment to see if it contains the
// environment variable we use as a heuristic for the launching process's
// intent to actually start an RPC plugin, rather than a normal command.
//
// If RunningAsPlugin returns false, the caller should typically return a
// helpful error message explaining that this isn't a normal command.
//
// RunningAsPlugin consults the real process environment by default, but
// callers can override that by using the ctxenv package to write custom
// variables into the given Context, which is intended primarily for
// testing purposes rather than main code use.
func RunningAsPlugin(ctx context.Context) bool {
	got := ctxenv.Getenv(ctx, handshakeConfig.CookieKey)
	want := handshakeConfig.CookieValue
	return got == want
}
