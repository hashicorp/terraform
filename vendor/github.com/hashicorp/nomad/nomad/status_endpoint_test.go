package nomad

import (
	"testing"

	"github.com/hashicorp/net-rpc-msgpackrpc"
	"github.com/hashicorp/nomad/nomad/structs"
	"github.com/hashicorp/nomad/testutil"
)

func TestStatusVersion(t *testing.T) {
	s1 := testServer(t, nil)
	defer s1.Shutdown()
	codec := rpcClient(t, s1)

	arg := &structs.GenericRequest{
		QueryOptions: structs.QueryOptions{
			Region:     "global",
			AllowStale: true,
		},
	}
	var out structs.VersionResponse
	if err := msgpackrpc.CallWithCodec(codec, "Status.Version", arg, &out); err != nil {
		t.Fatalf("err: %v", err)
	}

	if out.Build == "" {
		t.Fatalf("bad: %#v", out)
	}
	if out.Versions[structs.ProtocolVersion] != ProtocolVersionMax {
		t.Fatalf("bad: %#v", out)
	}
	if out.Versions[structs.APIMajorVersion] != structs.ApiMajorVersion {
		t.Fatalf("bad: %#v", out)
	}
	if out.Versions[structs.APIMinorVersion] != structs.ApiMinorVersion {
		t.Fatalf("bad: %#v", out)
	}
}

func TestStatusPing(t *testing.T) {
	s1 := testServer(t, nil)
	defer s1.Shutdown()
	codec := rpcClient(t, s1)

	arg := struct{}{}
	var out struct{}
	if err := msgpackrpc.CallWithCodec(codec, "Status.Ping", arg, &out); err != nil {
		t.Fatalf("err: %v", err)
	}
}

func TestStatusLeader(t *testing.T) {
	s1 := testServer(t, nil)
	defer s1.Shutdown()
	codec := rpcClient(t, s1)
	testutil.WaitForLeader(t, s1.RPC)

	arg := &structs.GenericRequest{
		QueryOptions: structs.QueryOptions{
			Region:     "global",
			AllowStale: true,
		},
	}
	var leader string
	if err := msgpackrpc.CallWithCodec(codec, "Status.Leader", arg, &leader); err != nil {
		t.Fatalf("err: %v", err)
	}
	if leader == "" {
		t.Fatalf("unexpected leader: %v", leader)
	}
}

func TestStatusPeers(t *testing.T) {
	s1 := testServer(t, nil)
	defer s1.Shutdown()
	codec := rpcClient(t, s1)

	arg := &structs.GenericRequest{
		QueryOptions: structs.QueryOptions{
			Region:     "global",
			AllowStale: true,
		},
	}
	var peers []string
	if err := msgpackrpc.CallWithCodec(codec, "Status.Peers", arg, &peers); err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(peers) != 1 {
		t.Fatalf("no peers: %v", peers)
	}
}
