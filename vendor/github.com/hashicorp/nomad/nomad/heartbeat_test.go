package nomad

import (
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/net-rpc-msgpackrpc"
	"github.com/hashicorp/nomad/nomad/mock"
	"github.com/hashicorp/nomad/nomad/structs"
	"github.com/hashicorp/nomad/testutil"
)

func TestInitializeHeartbeatTimers(t *testing.T) {
	s1 := testServer(t, nil)
	defer s1.Shutdown()
	testutil.WaitForLeader(t, s1.RPC)

	node := mock.Node()
	state := s1.fsm.State()
	err := state.UpsertNode(1, node)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	// Reset the heartbeat timers
	err = s1.initializeHeartbeatTimers()
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	// Check that we have a timer
	_, ok := s1.heartbeatTimers[node.ID]
	if !ok {
		t.Fatalf("missing heartbeat timer")
	}
}

func TestResetHeartbeatTimer(t *testing.T) {
	s1 := testServer(t, nil)
	defer s1.Shutdown()
	testutil.WaitForLeader(t, s1.RPC)

	// Create a new timer
	ttl, err := s1.resetHeartbeatTimer("test")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if ttl < s1.config.MinHeartbeatTTL || ttl > 2*s1.config.MinHeartbeatTTL {
		t.Fatalf("bad: %#v", ttl)
	}

	// Check that we have a timer
	_, ok := s1.heartbeatTimers["test"]
	if !ok {
		t.Fatalf("missing heartbeat timer")
	}
}

func TestResetHeartbeatTimerLocked(t *testing.T) {
	s1 := testServer(t, nil)
	defer s1.Shutdown()
	testutil.WaitForLeader(t, s1.RPC)

	s1.heartbeatTimersLock.Lock()
	s1.resetHeartbeatTimerLocked("foo", 5*time.Millisecond)
	s1.heartbeatTimersLock.Unlock()

	if _, ok := s1.heartbeatTimers["foo"]; !ok {
		t.Fatalf("missing timer")
	}

	time.Sleep(10 * time.Millisecond)

	if _, ok := s1.heartbeatTimers["foo"]; ok {
		t.Fatalf("timer should be gone")
	}
}

func TestResetHeartbeatTimerLocked_Renew(t *testing.T) {
	s1 := testServer(t, nil)
	defer s1.Shutdown()
	testutil.WaitForLeader(t, s1.RPC)

	s1.heartbeatTimersLock.Lock()
	s1.resetHeartbeatTimerLocked("foo", 5*time.Millisecond)
	s1.heartbeatTimersLock.Unlock()

	if _, ok := s1.heartbeatTimers["foo"]; !ok {
		t.Fatalf("missing timer")
	}

	time.Sleep(2 * time.Millisecond)

	// Renew the heartbeat
	s1.heartbeatTimersLock.Lock()
	s1.resetHeartbeatTimerLocked("foo", 5*time.Millisecond)
	s1.heartbeatTimersLock.Unlock()
	renew := time.Now()

	// Watch for invalidation
	for time.Now().Sub(renew) < 20*time.Millisecond {
		s1.heartbeatTimersLock.Lock()
		_, ok := s1.heartbeatTimers["foo"]
		s1.heartbeatTimersLock.Unlock()
		if !ok {
			end := time.Now()
			if diff := end.Sub(renew); diff < 5*time.Millisecond {
				t.Fatalf("early invalidate %v", diff)
			}
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("should have expired")
}

func TestInvalidateHeartbeat(t *testing.T) {
	s1 := testServer(t, nil)
	defer s1.Shutdown()
	testutil.WaitForLeader(t, s1.RPC)

	// Create a node
	node := mock.Node()
	state := s1.fsm.State()
	err := state.UpsertNode(1, node)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	// This should cause a status update
	s1.invalidateHeartbeat(node.ID)

	// Check it is updated
	out, err := state.NodeByID(node.ID)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !out.TerminalStatus() {
		t.Fatalf("should update node: %#v", out)
	}
}

func TestClearHeartbeatTimer(t *testing.T) {
	s1 := testServer(t, nil)
	defer s1.Shutdown()
	testutil.WaitForLeader(t, s1.RPC)

	s1.heartbeatTimersLock.Lock()
	s1.resetHeartbeatTimerLocked("foo", 5*time.Millisecond)
	s1.heartbeatTimersLock.Unlock()

	err := s1.clearHeartbeatTimer("foo")
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	if _, ok := s1.heartbeatTimers["foo"]; ok {
		t.Fatalf("timer should be gone")
	}
}

func TestClearAllHeartbeatTimers(t *testing.T) {
	s1 := testServer(t, nil)
	defer s1.Shutdown()
	testutil.WaitForLeader(t, s1.RPC)

	s1.heartbeatTimersLock.Lock()
	s1.resetHeartbeatTimerLocked("foo", 10*time.Millisecond)
	s1.resetHeartbeatTimerLocked("bar", 10*time.Millisecond)
	s1.resetHeartbeatTimerLocked("baz", 10*time.Millisecond)
	s1.heartbeatTimersLock.Unlock()

	err := s1.clearAllHeartbeatTimers()
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	if len(s1.heartbeatTimers) != 0 {
		t.Fatalf("timers should be gone")
	}
}

func TestServer_HeartbeatTTL_Failover(t *testing.T) {
	s1 := testServer(t, nil)
	defer s1.Shutdown()

	s2 := testServer(t, func(c *Config) {
		c.DevDisableBootstrap = true
	})
	defer s2.Shutdown()

	s3 := testServer(t, func(c *Config) {
		c.DevDisableBootstrap = true
	})
	defer s3.Shutdown()
	servers := []*Server{s1, s2, s3}
	testJoin(t, s1, s2, s3)

	testutil.WaitForResult(func() (bool, error) {
		peers, _ := s1.raftPeers.Peers()
		return len(peers) == 3, nil
	}, func(err error) {
		t.Fatalf("should have 3 peers")
	})

	// Find the leader
	var leader *Server
	for _, s := range servers {
		// Check that s.heartbeatTimers is empty
		if len(s.heartbeatTimers) != 0 {
			t.Fatalf("should have no heartbeatTimers")
		}
		// Find the leader too
		if s.IsLeader() {
			leader = s
		}
	}
	if leader == nil {
		t.Fatalf("Should have a leader")
	}
	codec := rpcClient(t, leader)

	// Create the register request
	node := mock.Node()
	req := &structs.NodeRegisterRequest{
		Node:         node,
		WriteRequest: structs.WriteRequest{Region: "global"},
	}

	// Fetch the response
	var resp structs.GenericResponse
	if err := msgpackrpc.CallWithCodec(codec, "Node.Register", req, &resp); err != nil {
		t.Fatalf("err: %v", err)
	}

	// Check that heartbeatTimers has the heartbeat ID
	if _, ok := leader.heartbeatTimers[node.ID]; !ok {
		t.Fatalf("missing heartbeat timer")
	}

	// Shutdown the leader!
	leader.Shutdown()

	// heartbeatTimers should be cleared on leader shutdown
	if len(leader.heartbeatTimers) != 0 {
		t.Fatalf("heartbeat timers should be empty on the shutdown leader")
	}

	// Find the new leader
	testutil.WaitForResult(func() (bool, error) {
		leader = nil
		for _, s := range servers {
			if s.IsLeader() {
				leader = s
			}
		}
		if leader == nil {
			return false, fmt.Errorf("Should have a new leader")
		}

		// Ensure heartbeat timer is restored
		if _, ok := leader.heartbeatTimers[node.ID]; !ok {
			return false, fmt.Errorf("missing heartbeat timer")
		}

		return true, nil
	}, func(err error) {
		t.Fatalf("err: %s", err)
	})
}
