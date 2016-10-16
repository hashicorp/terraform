package api

import (
	"reflect"
	"sort"
	"testing"

	"github.com/hashicorp/nomad/testutil"
)

func TestAgent_Self(t *testing.T) {
	c, s := makeClient(t, nil, nil)
	defer s.Stop()

	// Get a handle on the Agent endpoints
	a := c.Agent()

	// Query the endpoint
	res, err := a.Self()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	// Check that we got a valid response
	if name, ok := res["member"]["Name"]; !ok || name == "" {
		t.Fatalf("bad member name in response: %#v", res)
	}

	// Local cache was populated
	if a.nodeName == "" || a.datacenter == "" || a.region == "" {
		t.Fatalf("cache should be populated, got: %#v", a)
	}
}

func TestAgent_NodeName(t *testing.T) {
	c, s := makeClient(t, nil, nil)
	defer s.Stop()
	a := c.Agent()

	// Query the agent for the node name
	res, err := a.NodeName()
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if res == "" {
		t.Fatalf("expected node name, got nothing")
	}
}

func TestAgent_Datacenter(t *testing.T) {
	c, s := makeClient(t, nil, nil)
	defer s.Stop()
	a := c.Agent()

	// Query the agent for the datacenter
	dc, err := a.Datacenter()
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if dc != "dc1" {
		t.Fatalf("expected dc1, got: %q", dc)
	}
}

func TestAgent_Join(t *testing.T) {
	c1, s1 := makeClient(t, nil, nil)
	defer s1.Stop()
	a1 := c1.Agent()

	_, s2 := makeClient(t, nil, func(c *testutil.TestServerConfig) {
		c.Server.BootstrapExpect = 0
	})
	defer s2.Stop()

	// Attempting to join a non-existent host returns error
	n, err := a1.Join("nope")
	if err == nil {
		t.Fatalf("expected error, got nothing")
	}
	if n != 0 {
		t.Fatalf("expected 0 nodes, got: %d", n)
	}

	// Returns correctly if join succeeds
	n, err = a1.Join(s2.SerfAddr)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if n != 1 {
		t.Fatalf("expected 1 node, got: %d", n)
	}
}

func TestAgent_Members(t *testing.T) {
	c, s := makeClient(t, nil, nil)
	defer s.Stop()
	a := c.Agent()

	// Query nomad for all the known members
	mem, err := a.Members()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	// Check that we got the expected result
	if n := len(mem); n != 1 {
		t.Fatalf("expected 1 member, got: %d", n)
	}
	if m := mem[0]; m.Name == "" || m.Addr == "" || m.Port == 0 {
		t.Fatalf("bad member: %#v", m)
	}
}

func TestAgent_ForceLeave(t *testing.T) {
	c, s := makeClient(t, nil, nil)
	defer s.Stop()
	a := c.Agent()

	// Force-leave on a non-existent node does not error
	if err := a.ForceLeave("nope"); err != nil {
		t.Fatalf("err: %s", err)
	}

	// TODO: test force-leave on an existing node
}

func (a *AgentMember) String() string {
	return "{Name: " + a.Name + " Region: " + a.Tags["region"] + " DC: " + a.Tags["dc"] + "}"
}

func TestAgents_Sort(t *testing.T) {
	var sortTests = []struct {
		in  []*AgentMember
		out []*AgentMember
	}{
		{
			[]*AgentMember{
				&AgentMember{Name: "nomad-2.vac.us-east",
					Tags: map[string]string{"region": "us-east", "dc": "us-east-1c"}},
				&AgentMember{Name: "nomad-1.global",
					Tags: map[string]string{"region": "global", "dc": "dc1"}},
				&AgentMember{Name: "nomad-1.vac.us-east",
					Tags: map[string]string{"region": "us-east", "dc": "us-east-1c"}},
			},
			[]*AgentMember{
				&AgentMember{Name: "nomad-1.global",
					Tags: map[string]string{"region": "global", "dc": "dc1"}},
				&AgentMember{Name: "nomad-1.vac.us-east",
					Tags: map[string]string{"region": "us-east", "dc": "us-east-1c"}},
				&AgentMember{Name: "nomad-2.vac.us-east",
					Tags: map[string]string{"region": "us-east", "dc": "us-east-1c"}},
			},
		},
		{
			[]*AgentMember{
				&AgentMember{Name: "nomad-02.tam.us-east",
					Tags: map[string]string{"region": "us-east", "dc": "tampa"}},
				&AgentMember{Name: "nomad-02.pal.us-west",
					Tags: map[string]string{"region": "us-west", "dc": "palo_alto"}},
				&AgentMember{Name: "nomad-01.pal.us-west",
					Tags: map[string]string{"region": "us-west", "dc": "palo_alto"}},
				&AgentMember{Name: "nomad-01.tam.us-east",
					Tags: map[string]string{"region": "us-east", "dc": "tampa"}},
			},
			[]*AgentMember{
				&AgentMember{Name: "nomad-01.tam.us-east",
					Tags: map[string]string{"region": "us-east", "dc": "tampa"}},
				&AgentMember{Name: "nomad-02.tam.us-east",
					Tags: map[string]string{"region": "us-east", "dc": "tampa"}},
				&AgentMember{Name: "nomad-01.pal.us-west",
					Tags: map[string]string{"region": "us-west", "dc": "palo_alto"}},
				&AgentMember{Name: "nomad-02.pal.us-west",
					Tags: map[string]string{"region": "us-west", "dc": "palo_alto"}},
			},
		},
		{
			[]*AgentMember{
				&AgentMember{Name: "nomad-02.tam.us-east",
					Tags: map[string]string{"region": "us-east", "dc": "tampa"}},
				&AgentMember{Name: "nomad-02.ams.europe",
					Tags: map[string]string{"region": "europe", "dc": "amsterdam"}},
				&AgentMember{Name: "nomad-01.tam.us-east",
					Tags: map[string]string{"region": "us-east", "dc": "tampa"}},
				&AgentMember{Name: "nomad-01.ams.europe",
					Tags: map[string]string{"region": "europe", "dc": "amsterdam"}},
			},
			[]*AgentMember{
				&AgentMember{Name: "nomad-01.ams.europe",
					Tags: map[string]string{"region": "europe", "dc": "amsterdam"}},
				&AgentMember{Name: "nomad-02.ams.europe",
					Tags: map[string]string{"region": "europe", "dc": "amsterdam"}},
				&AgentMember{Name: "nomad-01.tam.us-east",
					Tags: map[string]string{"region": "us-east", "dc": "tampa"}},
				&AgentMember{Name: "nomad-02.tam.us-east",
					Tags: map[string]string{"region": "us-east", "dc": "tampa"}},
			},
		},
		{
			[]*AgentMember{
				&AgentMember{Name: "nomad-02.ber.europe",
					Tags: map[string]string{"region": "europe", "dc": "berlin"}},
				&AgentMember{Name: "nomad-02.ams.europe",
					Tags: map[string]string{"region": "europe", "dc": "amsterdam"}},
				&AgentMember{Name: "nomad-01.ams.europe",
					Tags: map[string]string{"region": "europe", "dc": "amsterdam"}},
				&AgentMember{Name: "nomad-01.ber.europe",
					Tags: map[string]string{"region": "europe", "dc": "berlin"}},
			},
			[]*AgentMember{
				&AgentMember{Name: "nomad-01.ams.europe",
					Tags: map[string]string{"region": "europe", "dc": "amsterdam"}},
				&AgentMember{Name: "nomad-02.ams.europe",
					Tags: map[string]string{"region": "europe", "dc": "amsterdam"}},
				&AgentMember{Name: "nomad-01.ber.europe",
					Tags: map[string]string{"region": "europe", "dc": "berlin"}},
				&AgentMember{Name: "nomad-02.ber.europe",
					Tags: map[string]string{"region": "europe", "dc": "berlin"}},
			},
		},
		{
			[]*AgentMember{
				&AgentMember{Name: "nomad-1.global"},
				&AgentMember{Name: "nomad-3.global"},
				&AgentMember{Name: "nomad-2.global"},
			},
			[]*AgentMember{
				&AgentMember{Name: "nomad-1.global"},
				&AgentMember{Name: "nomad-2.global"},
				&AgentMember{Name: "nomad-3.global"},
			},
		},
	}
	for _, tt := range sortTests {
		sort.Sort(AgentMembersNameSort(tt.in))
		if !reflect.DeepEqual(tt.in, tt.out) {
			t.Errorf("\necpected: %s\nget     : %s", tt.in, tt.out)
		}
	}
}
