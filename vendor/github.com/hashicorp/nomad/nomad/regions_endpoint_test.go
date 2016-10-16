package nomad

import (
	"fmt"
	"testing"

	"github.com/hashicorp/net-rpc-msgpackrpc"
	"github.com/hashicorp/nomad/nomad/structs"
	"github.com/hashicorp/nomad/testutil"
)

func TestRegionList(t *testing.T) {
	// Make the servers
	s1 := testServer(t, func(c *Config) {
		c.Region = "region1"
	})
	defer s1.Shutdown()
	codec := rpcClient(t, s1)

	s2 := testServer(t, func(c *Config) {
		c.Region = "region2"
	})
	defer s2.Shutdown()

	// Join the servers
	s2Addr := fmt.Sprintf("127.0.0.1:%d",
		s2.config.SerfConfig.MemberlistConfig.BindPort)
	if n, err := s1.Join([]string{s2Addr}); err != nil || n != 1 {
		t.Fatalf("Failed joining: %v (%d joined)", err, n)
	}

	// Query the regions list
	testutil.WaitForResult(func() (bool, error) {
		var arg structs.GenericRequest
		var out []string
		if err := msgpackrpc.CallWithCodec(codec, "Region.List", &arg, &out); err != nil {
			t.Fatalf("err: %v", err)
		}
		if len(out) != 2 || out[0] != "region1" || out[1] != "region2" {
			t.Fatalf("unexpected regions: %v", out)
		}
		return true, nil
	}, func(err error) {
		t.Fatalf("err: %v", err)
	})
}
