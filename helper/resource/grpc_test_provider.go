package resource

import (
	"context"
	"net"
	"time"

	"github.com/hashicorp/terraform/helper/plugin"
	tfplugin "github.com/hashicorp/terraform/plugin"
	"github.com/hashicorp/terraform/plugin/proto"
	"github.com/hashicorp/terraform/providers"
	"github.com/hashicorp/terraform/terraform"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
)

// GRPCTestProvider takes a legacy ResourceProvider, wraps it in the new GRPC
// shim and starts it in a grpc server using an inmem connection. It returns a
// GRPCClient for this new server to test the shimmed resource provider.
func GRPCTestProvider(rp terraform.ResourceProvider) providers.Interface {
	listener := bufconn.Listen(256 * 1024)
	grpcServer := grpc.NewServer()

	p := plugin.NewGRPCProviderServerShim(rp)
	proto.RegisterProviderServer(grpcServer, p)

	go grpcServer.Serve(listener)

	conn, err := grpc.Dial("", grpc.WithDialer(func(string, time.Duration) (net.Conn, error) {
		return listener.Dial()
	}), grpc.WithInsecure())
	if err != nil {
		panic(err)
	}

	var pp tfplugin.GRPCProviderPlugin
	client, _ := pp.GRPCClient(context.Background(), nil, conn)

	grpcClient := client.(*tfplugin.GRPCProvider)
	grpcClient.TestListener = listener

	return grpcClient
}
