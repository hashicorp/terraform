package api

import (
	"fmt"
	"testing"

	"github.com/hashicorp/nomad/testutil"
)

func TestRegionsList(t *testing.T) {
	c1, s1 := makeClient(t, nil, func(c *testutil.TestServerConfig) {
		c.Region = "regionA"
	})
	defer s1.Stop()

	c2, s2 := makeClient(t, nil, func(c *testutil.TestServerConfig) {
		c.Region = "regionB"
	})
	defer s2.Stop()

	// Join the servers
	if _, err := c2.Agent().Join(s1.SerfAddr); err != nil {
		t.Fatalf("err: %v", err)
	}

	// Regions returned and sorted
	testutil.WaitForResult(func() (bool, error) {
		regions, err := c1.Regions().List()
		if err != nil {
			return false, err
		}
		if n := len(regions); n != 2 {
			return false, fmt.Errorf("expected 2 regions, got: %d", n)
		}
		if regions[0] != "regionA" || regions[1] != "regionB" {
			return false, fmt.Errorf("bad: %#v", regions)
		}
		return true, nil
	}, func(err error) {
		t.Fatalf("err: %v", err)
	})
}
