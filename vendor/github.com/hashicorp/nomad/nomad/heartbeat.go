package nomad

import (
	"time"

	"github.com/armon/go-metrics"
	"github.com/hashicorp/consul/lib"
	"github.com/hashicorp/nomad/nomad/structs"
)

// initializeHeartbeatTimers is used when a leader is newly elected to create
// a new map to track heartbeat expiration and to reset all the timers from
// the previously known set of timers.
func (s *Server) initializeHeartbeatTimers() error {
	// Scan all nodes and reset their timer
	snap, err := s.fsm.State().Snapshot()
	if err != nil {
		return err
	}

	// Get an iterator over nodes
	iter, err := snap.Nodes()
	if err != nil {
		return err
	}

	s.heartbeatTimersLock.Lock()
	defer s.heartbeatTimersLock.Unlock()

	// Handle each node
	for {
		raw := iter.Next()
		if raw == nil {
			break
		}
		node := raw.(*structs.Node)
		if node.TerminalStatus() {
			continue
		}
		s.resetHeartbeatTimerLocked(node.ID, s.config.FailoverHeartbeatTTL)
	}
	return nil
}

// resetHeartbeatTimer is used to reset the TTL of a heartbeat.
// This can be used for new heartbeats and existing ones.
func (s *Server) resetHeartbeatTimer(id string) (time.Duration, error) {
	s.heartbeatTimersLock.Lock()
	defer s.heartbeatTimersLock.Unlock()

	// Compute the target TTL value
	n := len(s.heartbeatTimers)
	ttl := lib.RateScaledInterval(s.config.MaxHeartbeatsPerSecond, s.config.MinHeartbeatTTL, n)
	ttl += lib.RandomStagger(ttl)

	// Reset the TTL
	s.resetHeartbeatTimerLocked(id, ttl+s.config.HeartbeatGrace)
	return ttl, nil
}

// resetHeartbeatTimerLocked is used to reset a heartbeat timer
// assuming the heartbeatTimerLock is already held
func (s *Server) resetHeartbeatTimerLocked(id string, ttl time.Duration) {
	// Ensure a timer map exists
	if s.heartbeatTimers == nil {
		s.heartbeatTimers = make(map[string]*time.Timer)
	}

	// Renew the heartbeat timer if it exists
	if timer, ok := s.heartbeatTimers[id]; ok {
		timer.Reset(ttl)
		return
	}

	// Create a new timer to track expiration of this heartbeat
	timer := time.AfterFunc(ttl, func() {
		s.invalidateHeartbeat(id)
	})
	s.heartbeatTimers[id] = timer
}

// invalidateHeartbeat is invoked when a heartbeat TTL is reached and we
// need to invalidate the heartbeat.
func (s *Server) invalidateHeartbeat(id string) {
	defer metrics.MeasureSince([]string{"nomad", "heartbeat", "invalidate"}, time.Now())
	// Clear the heartbeat timer
	s.heartbeatTimersLock.Lock()
	delete(s.heartbeatTimers, id)
	s.heartbeatTimersLock.Unlock()
	s.logger.Printf("[WARN] nomad.heartbeat: node '%s' TTL expired", id)

	// Make a request to update the node status
	req := structs.NodeUpdateStatusRequest{
		NodeID: id,
		Status: structs.NodeStatusDown,
		WriteRequest: structs.WriteRequest{
			Region: s.config.Region,
		},
	}
	var resp structs.NodeUpdateResponse
	if err := s.endpoints.Node.UpdateStatus(&req, &resp); err != nil {
		s.logger.Printf("[ERR] nomad.heartbeat: update status failed: %v", err)
	}
}

// clearHeartbeatTimer is used to clear the heartbeat time for
// a single heartbeat. This is used when a heartbeat is destroyed
// explicitly and no longer needed.
func (s *Server) clearHeartbeatTimer(id string) error {
	s.heartbeatTimersLock.Lock()
	defer s.heartbeatTimersLock.Unlock()

	if timer, ok := s.heartbeatTimers[id]; ok {
		timer.Stop()
		delete(s.heartbeatTimers, id)
	}
	return nil
}

// clearAllHeartbeatTimers is used when a leader is stepping
// down and we no longer need to track any heartbeat timers.
func (s *Server) clearAllHeartbeatTimers() error {
	s.heartbeatTimersLock.Lock()
	defer s.heartbeatTimersLock.Unlock()

	for _, t := range s.heartbeatTimers {
		t.Stop()
	}
	s.heartbeatTimers = nil
	return nil
}

// heartbeatStats is a long running routine used to capture
// the number of active heartbeats being tracked
func (s *Server) heartbeatStats() {
	for {
		select {
		case <-time.After(5 * time.Second):
			s.heartbeatTimersLock.Lock()
			num := len(s.heartbeatTimers)
			s.heartbeatTimersLock.Unlock()
			metrics.SetGauge([]string{"nomad", "heartbeat", "active"}, float32(num))

		case <-s.shutdownCh:
			return
		}
	}
}
