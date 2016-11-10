package nomad

import (
	"sync/atomic"

	"github.com/hashicorp/serf/serf"
)

const (
	// StatusReap is used to update the status of a node if we
	// are handling a EventMemberReap
	StatusReap = serf.MemberStatus(-1)
)

// serfEventHandler is used to handle events from the serf cluster
func (s *Server) serfEventHandler() {
	for {
		select {
		case e := <-s.eventCh:
			switch e.EventType() {
			case serf.EventMemberJoin:
				s.nodeJoin(e.(serf.MemberEvent))
				s.localMemberEvent(e.(serf.MemberEvent))
			case serf.EventMemberLeave, serf.EventMemberFailed:
				s.nodeFailed(e.(serf.MemberEvent))
				s.localMemberEvent(e.(serf.MemberEvent))
			case serf.EventMemberUpdate, serf.EventMemberReap,
				serf.EventUser, serf.EventQuery: // Ignore
			default:
				s.logger.Printf("[WARN] nomad: unhandled serf event: %#v", e)
			}

		case <-s.shutdownCh:
			return
		}
	}
}

// nodeJoin is used to handle join events on the serf cluster
func (s *Server) nodeJoin(me serf.MemberEvent) {
	for _, m := range me.Members {
		ok, parts := isNomadServer(m)
		if !ok {
			s.logger.Printf("[WARN] nomad: non-server in gossip pool: %s", m.Name)
			continue
		}
		s.logger.Printf("[INFO] nomad: adding server %s", parts)

		// Check if this server is known
		found := false
		s.peerLock.Lock()
		existing := s.peers[parts.Region]
		for idx, e := range existing {
			if e.Name == parts.Name {
				existing[idx] = parts
				found = true
				break
			}
		}

		// Add ot the list if not known
		if !found {
			s.peers[parts.Region] = append(existing, parts)
		}

		// Check if a local peer
		if parts.Region == s.config.Region {
			s.localPeers[parts.Addr.String()] = parts
		}
		s.peerLock.Unlock()

		// If we still expecting to bootstrap, may need to handle this
		if atomic.LoadInt32(&s.config.BootstrapExpect) != 0 {
			s.maybeBootstrap()
		}
	}
}

// maybeBootsrap is used to handle bootstrapping when a new server joins
func (s *Server) maybeBootstrap() {
	var index uint64
	var err error
	if s.raftStore != nil {
		index, err = s.raftStore.LastIndex()
	} else if s.raftInmem != nil {
		index, err = s.raftInmem.LastIndex()
	} else {
		panic("neither raftInmem or raftStore is initialized")
	}
	if err != nil {
		s.logger.Printf("[ERR] nomad: failed to read last raft index: %v", err)
		return
	}

	// Bootstrap can only be done if there are no committed logs,
	// remove our expectations of bootstrapping
	if index != 0 {
		atomic.StoreInt32(&s.config.BootstrapExpect, 0)
		return
	}

	// Scan for all the known servers
	members := s.serf.Members()
	addrs := make([]string, 0)
	for _, member := range members {
		valid, p := isNomadServer(member)
		if !valid {
			continue
		}
		if p.Region != s.config.Region {
			continue
		}
		if p.Expect != 0 && p.Expect != int(atomic.LoadInt32(&s.config.BootstrapExpect)) {
			s.logger.Printf("[ERR] nomad: peer %v has a conflicting expect value. All nodes should expect the same number.", member)
			return
		}
		if p.Bootstrap {
			s.logger.Printf("[ERR] nomad: peer %v has bootstrap mode. Expect disabled.", member)
			return
		}
		addrs = append(addrs, p.Addr.String())
	}

	// Skip if we haven't met the minimum expect count
	if len(addrs) < int(atomic.LoadInt32(&s.config.BootstrapExpect)) {
		return
	}

	// Update the peer set
	s.logger.Printf("[INFO] nomad: Attempting bootstrap with nodes: %v", addrs)
	if err := s.raft.SetPeers(addrs).Error(); err != nil {
		s.logger.Printf("[ERR] nomad: failed to bootstrap peers: %v", err)
	}

	// Bootstrapping complete, don't enter this again
	atomic.StoreInt32(&s.config.BootstrapExpect, 0)
}

// nodeFailed is used to handle fail events on the serf cluster
func (s *Server) nodeFailed(me serf.MemberEvent) {
	for _, m := range me.Members {
		ok, parts := isNomadServer(m)
		if !ok {
			continue
		}
		s.logger.Printf("[INFO] nomad: removing server %s", parts)

		// Remove the server if known
		s.peerLock.Lock()
		existing := s.peers[parts.Region]
		n := len(existing)
		for i := 0; i < n; i++ {
			if existing[i].Name == parts.Name {
				existing[i], existing[n-1] = existing[n-1], nil
				existing = existing[:n-1]
				n--
				break
			}
		}

		// Trim the list there are no known servers in a region
		if n == 0 {
			delete(s.peers, parts.Region)
		} else {
			s.peers[parts.Region] = existing
		}

		// Check if local peer
		if parts.Region == s.config.Region {
			delete(s.localPeers, parts.Addr.String())
		}
		s.peerLock.Unlock()
	}
}

// localMemberEvent is used to reconcile Serf events with the
// consistent store if we are the current leader.
func (s *Server) localMemberEvent(me serf.MemberEvent) {
	// Do nothing if we are not the leader
	if !s.IsLeader() {
		return
	}

	// Check if this is a reap event
	isReap := me.EventType() == serf.EventMemberReap

	// Queue the members for reconciliation
	for _, m := range me.Members {
		// Change the status if this is a reap event
		if isReap {
			m.Status = StatusReap
		}
		select {
		case s.reconcileCh <- m:
		default:
		}
	}
}
