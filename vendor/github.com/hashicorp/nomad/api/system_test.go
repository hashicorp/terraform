package api

import (
	"testing"
)

func TestSystem_GarbageCollect(t *testing.T) {
	c, s := makeClient(t, nil, nil)
	defer s.Stop()
	e := c.System()
	if err := e.GarbageCollect(); err != nil {
		t.Fatal(err)
	}
}
