package nomad

import (
	"net"
	"net/rpc"
	"testing"
	"time"

	"github.com/hashicorp/nomad/testutil"
)

// rpcClient is a test helper method to return a ClientCodec to use to make rpc
// calls to the passed server.
func rpcClient(t *testing.T, s *Server) rpc.ClientCodec {
	addr := s.config.RPCAddr
	conn, err := net.DialTimeout("tcp", addr.String(), time.Second)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	// Write the Consul RPC byte to set the mode
	conn.Write([]byte{byte(rpcNomad)})
	return NewClientCodec(conn)
}

func TestRPC_forwardLeader(t *testing.T) {
	s1 := testServer(t, nil)
	defer s1.Shutdown()
	s2 := testServer(t, func(c *Config) {
		c.DevDisableBootstrap = true
	})
	defer s2.Shutdown()
	testJoin(t, s1, s2)
	testutil.WaitForLeader(t, s1.RPC)
	testutil.WaitForLeader(t, s2.RPC)

	isLeader, remote := s1.getLeader()
	if !isLeader && remote == nil {
		t.Fatalf("missing leader")
	}

	if remote != nil {
		var out struct{}
		err := s1.forwardLeader(remote, "Status.Ping", struct{}{}, &out)
		if err != nil {
			t.Fatalf("err: %v", err)
		}
	}

	isLeader, remote = s2.getLeader()
	if !isLeader && remote == nil {
		t.Fatalf("missing leader")
	}

	if remote != nil {
		var out struct{}
		err := s2.forwardLeader(remote, "Status.Ping", struct{}{}, &out)
		if err != nil {
			t.Fatalf("err: %v", err)
		}
	}
}

func TestRPC_forwardRegion(t *testing.T) {
	s1 := testServer(t, nil)
	defer s1.Shutdown()
	s2 := testServer(t, func(c *Config) {
		c.Region = "region2"
	})
	defer s2.Shutdown()
	testJoin(t, s1, s2)
	testutil.WaitForLeader(t, s1.RPC)
	testutil.WaitForLeader(t, s2.RPC)

	var out struct{}
	err := s1.forwardRegion("region2", "Status.Ping", struct{}{}, &out)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	err = s2.forwardRegion("global", "Status.Ping", struct{}{}, &out)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
}
