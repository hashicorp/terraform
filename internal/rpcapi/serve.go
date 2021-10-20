package rpcapi

import (
	"context"

	"go.rpcplugin.org/rpcplugin"
)

// Serve starts the rpcplugin server. It does not return unless server startup
// encounters an error.
func Serve(ctx context.Context) error {
	return rpcplugin.Serve(ctx, &rpcplugin.ServerConfig{
		Handshake: rpcplugin.HandshakeConfig{
			// This is just an arbitrary key/value pair so that the program
			// launching this process can affirm that it's expecting to talk
			// to an rpcplugin plugin, rather than a normal CLI tool.
			CookieKey:   "TERRAFORM_CORE_RPCPLUGIN_COOKIE",
			CookieValue: "36594bbabbaf5783bbbae2284929a2c9",
		},
		ProtoVersions: map[int]rpcplugin.ServerVersion{
			1: version1{},
		},
	})
}
