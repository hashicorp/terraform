package rpc

import (
	"net"
	"net/rpc"
	"testing"

	"github.com/hashicorp/terraform/terraform"
)

func testConn(t *testing.T) (net.Conn, net.Conn) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	var serverConn net.Conn
	doneCh := make(chan struct{})
	go func() {
		defer close(doneCh)
		defer l.Close()
		var err error
		serverConn, err = l.Accept()
		if err != nil {
			t.Fatalf("err: %s", err)
		}
	}()

	clientConn, err := net.Dial("tcp", l.Addr().String())
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	<-doneCh

	return clientConn, serverConn
}

func testClientServer(t *testing.T) (*rpc.Client, *rpc.Server) {
	clientConn, serverConn := testConn(t)

	server := rpc.NewServer()
	go server.ServeConn(serverConn)

	client := rpc.NewClient(clientConn)

	return client, server
}

func testNewClientServer(t *testing.T) (*Client, *Server) {
	clientConn, serverConn := testConn(t)

	server := &Server{
		ProviderFunc: testProviderFixed(new(terraform.MockResourceProvider)),
		ProvisionerFunc: testProvisionerFixed(
			new(terraform.MockResourceProvisioner)),
	}
	go server.ServeConn(serverConn)

	client, err := NewClient(clientConn)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	return client, server
}

func testProviderFixed(p terraform.ResourceProvider) ProviderFunc {
	return func() terraform.ResourceProvider {
		return p
	}
}

func testProvisionerFixed(p terraform.ResourceProvisioner) ProvisionerFunc {
	return func() terraform.ResourceProvisioner {
		return p
	}
}
