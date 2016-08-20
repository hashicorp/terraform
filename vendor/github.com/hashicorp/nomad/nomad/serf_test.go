package nomad

import (
	"fmt"
	"os"
	"path"
	"testing"

	"github.com/hashicorp/nomad/testutil"
)

func TestNomad_JoinPeer(t *testing.T) {
	s1 := testServer(t, nil)
	defer s1.Shutdown()
	s2 := testServer(t, func(c *Config) {
		c.Region = "region2"
	})
	defer s2.Shutdown()
	testJoin(t, s1, s2)

	testutil.WaitForResult(func() (bool, error) {
		if members := s1.Members(); len(members) != 2 {
			return false, fmt.Errorf("bad: %#v", members)
		}
		if members := s2.Members(); len(members) != 2 {
			return false, fmt.Errorf("bad: %#v", members)
		}
		return true, nil
	}, func(err error) {
		t.Fatalf("err: %v", err)
	})

	testutil.WaitForResult(func() (bool, error) {
		if len(s1.peers) != 2 {
			return false, fmt.Errorf("bad: %#v", s1.peers)
		}
		if len(s2.peers) != 2 {
			return false, fmt.Errorf("bad: %#v", s2.peers)
		}
		if len(s1.localPeers) != 1 {
			return false, fmt.Errorf("bad: %#v", s1.localPeers)
		}
		if len(s2.localPeers) != 1 {
			return false, fmt.Errorf("bad: %#v", s2.localPeers)
		}
		return true, nil
	}, func(err error) {
		t.Fatalf("err: %v", err)
	})
}

func TestNomad_RemovePeer(t *testing.T) {
	s1 := testServer(t, nil)
	defer s1.Shutdown()
	s2 := testServer(t, func(c *Config) {
		c.Region = "region2"
	})
	defer s2.Shutdown()
	testJoin(t, s1, s2)

	testutil.WaitForResult(func() (bool, error) {
		if members := s1.Members(); len(members) != 2 {
			return false, fmt.Errorf("bad: %#v", members)
		}
		if members := s2.Members(); len(members) != 2 {
			return false, fmt.Errorf("bad: %#v", members)
		}
		return true, nil
	}, func(err error) {
		t.Fatalf("err: %v", err)
	})

	// Leave immediately
	s2.Leave()
	s2.Shutdown()

	testutil.WaitForResult(func() (bool, error) {
		if len(s1.peers) != 1 {
			return false, fmt.Errorf("bad: %#v", s1.peers)
		}
		if len(s2.peers) != 1 {
			return false, fmt.Errorf("bad: %#v", s2.peers)
		}
		return true, nil
	}, func(err error) {
		t.Fatalf("err: %v", err)
	})
}

func TestNomad_BootstrapExpect(t *testing.T) {
	dir := tmpDir(t)
	defer os.RemoveAll(dir)

	s1 := testServer(t, func(c *Config) {
		c.BootstrapExpect = 2
		c.DevMode = false
		c.DataDir = path.Join(dir, "node1")
	})
	defer s1.Shutdown()
	s2 := testServer(t, func(c *Config) {
		c.BootstrapExpect = 2
		c.DevMode = false
		c.DataDir = path.Join(dir, "node2")
	})
	defer s2.Shutdown()
	testJoin(t, s1, s2)

	testutil.WaitForResult(func() (bool, error) {
		peers, err := s1.numOtherPeers()
		if err != nil {
			return false, err
		}
		if peers != 1 {
			return false, fmt.Errorf("bad: %#v", peers)
		}
		peers, err = s2.numOtherPeers()
		if err != nil {
			return false, err
		}
		if peers != 1 {
			return false, fmt.Errorf("bad: %#v", peers)
		}
		if len(s1.localPeers) != 2 {
			return false, fmt.Errorf("bad: %#v", s1.localPeers)
		}
		if len(s2.localPeers) != 2 {
			return false, fmt.Errorf("bad: %#v", s2.localPeers)
		}
		return true, nil
	}, func(err error) {
		t.Fatalf("err: %v", err)
	})
}

func TestNomad_BadExpect(t *testing.T) {
	s1 := testServer(t, func(c *Config) {
		c.BootstrapExpect = 2
		c.DevDisableBootstrap = true
	})
	defer s1.Shutdown()
	s2 := testServer(t, func(c *Config) {
		c.BootstrapExpect = 3
		c.DevDisableBootstrap = true
	})
	defer s2.Shutdown()
	servers := []*Server{s1, s2}
	testJoin(t, s1, s2)

	// Serf members should update
	testutil.WaitForResult(func() (bool, error) {
		for _, s := range servers {
			members := s.Members()
			if len(members) != 2 {
				return false, fmt.Errorf("%d", len(members))
			}
		}
		return true, nil
	}, func(err error) {
		t.Fatalf("should have 2 peers: %v", err)
	})

	// should still have no peers (because s2 is in expect=2 mode)
	testutil.WaitForResult(func() (bool, error) {
		for _, s := range servers {
			p, _ := s.raftPeers.Peers()
			if len(p) != 0 {
				return false, fmt.Errorf("%d", len(p))
			}
		}
		return true, nil
	}, func(err error) {
		t.Fatalf("should have 0 peers: %v", err)
	})
}
