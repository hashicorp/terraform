package gocql

import (
	"log"
	"net"
	"sync"
	"time"
)

type eventDeouncer struct {
	name   string
	timer  *time.Timer
	mu     sync.Mutex
	events []frame

	callback func([]frame)
	quit     chan struct{}
}

func newEventDeouncer(name string, eventHandler func([]frame)) *eventDeouncer {
	e := &eventDeouncer{
		name:     name,
		quit:     make(chan struct{}),
		timer:    time.NewTimer(eventDebounceTime),
		callback: eventHandler,
	}
	e.timer.Stop()
	go e.flusher()

	return e
}

func (e *eventDeouncer) stop() {
	e.quit <- struct{}{} // sync with flusher
	close(e.quit)
}

func (e *eventDeouncer) flusher() {
	for {
		select {
		case <-e.timer.C:
			e.mu.Lock()
			e.flush()
			e.mu.Unlock()
		case <-e.quit:
			return
		}
	}
}

const (
	eventBufferSize   = 1000
	eventDebounceTime = 1 * time.Second
)

// flush must be called with mu locked
func (e *eventDeouncer) flush() {
	if len(e.events) == 0 {
		return
	}

	// if the flush interval is faster than the callback then we will end up calling
	// the callback multiple times, probably a bad idea. In this case we could drop
	// frames?
	go e.callback(e.events)
	e.events = make([]frame, 0, eventBufferSize)
}

func (e *eventDeouncer) debounce(frame frame) {
	e.mu.Lock()
	e.timer.Reset(eventDebounceTime)

	// TODO: probably need a warning to track if this threshold is too low
	if len(e.events) < eventBufferSize {
		e.events = append(e.events, frame)
	} else {
		log.Printf("%s: buffer full, dropping event frame: %s", e.name, frame)
	}

	e.mu.Unlock()
}

func (s *Session) handleEvent(framer *framer) {
	// TODO(zariel): need to debounce events frames, and possible also events
	defer framerPool.Put(framer)

	frame, err := framer.parseFrame()
	if err != nil {
		// TODO: logger
		log.Printf("gocql: unable to parse event frame: %v\n", err)
		return
	}

	if gocqlDebug {
		log.Printf("gocql: handling frame: %v\n", frame)
	}

	// TODO: handle medatadata events
	switch f := frame.(type) {
	case *schemaChangeKeyspace, *schemaChangeFunction, *schemaChangeTable:
		s.schemaEvents.debounce(frame)
	case *topologyChangeEventFrame, *statusChangeEventFrame:
		s.nodeEvents.debounce(frame)
	default:
		log.Printf("gocql: invalid event frame (%T): %v\n", f, f)
	}
}

func (s *Session) handleSchemaEvent(frames []frame) {
	if s.schemaDescriber == nil {
		return
	}
	for _, frame := range frames {
		switch f := frame.(type) {
		case *schemaChangeKeyspace:
			s.schemaDescriber.clearSchema(f.keyspace)
		case *schemaChangeTable:
			s.schemaDescriber.clearSchema(f.keyspace)
		}
	}
}

func (s *Session) handleNodeEvent(frames []frame) {
	type nodeEvent struct {
		change string
		host   net.IP
		port   int
	}

	events := make(map[string]*nodeEvent)

	for _, frame := range frames {
		// TODO: can we be sure the order of events in the buffer is correct?
		switch f := frame.(type) {
		case *topologyChangeEventFrame:
			event, ok := events[f.host.String()]
			if !ok {
				event = &nodeEvent{change: f.change, host: f.host, port: f.port}
				events[f.host.String()] = event
			}
			event.change = f.change

		case *statusChangeEventFrame:
			event, ok := events[f.host.String()]
			if !ok {
				event = &nodeEvent{change: f.change, host: f.host, port: f.port}
				events[f.host.String()] = event
			}
			event.change = f.change
		}
	}

	for _, f := range events {
		if gocqlDebug {
			log.Printf("gocql: dispatching event: %+v\n", f)
		}

		switch f.change {
		case "NEW_NODE":
			s.handleNewNode(f.host, f.port, true)
		case "REMOVED_NODE":
			s.handleRemovedNode(f.host, f.port)
		case "MOVED_NODE":
		// java-driver handles this, not mentioned in the spec
		// TODO(zariel): refresh token map
		case "UP":
			s.handleNodeUp(f.host, f.port, true)
		case "DOWN":
			s.handleNodeDown(f.host, f.port)
		}
	}
}

func (s *Session) handleNewNode(host net.IP, port int, waitForBinary bool) {
	// TODO(zariel): need to be able to filter discovered nodes

	var hostInfo *HostInfo
	if s.control != nil && !s.cfg.IgnorePeerAddr {
		var err error
		hostInfo, err = s.control.fetchHostInfo(host, port)
		if err != nil {
			log.Printf("gocql: events: unable to fetch host info for %v: %v\n", host, err)
			return
		}

	} else {
		hostInfo = &HostInfo{peer: host.String(), port: port, state: NodeUp}
	}

	addr := host.String()
	if s.cfg.IgnorePeerAddr && hostInfo.Peer() != addr {
		hostInfo.setPeer(addr)
	}

	if s.cfg.HostFilter != nil {
		if !s.cfg.HostFilter.Accept(hostInfo) {
			return
		}
	} else if !s.cfg.Discovery.matchFilter(hostInfo) {
		// TODO: remove this when the host selection policy is more sophisticated
		return
	}

	if t := hostInfo.Version().nodeUpDelay(); t > 0 && waitForBinary {
		time.Sleep(t)
	}

	// should this handle token moving?
	if existing, ok := s.ring.addHostIfMissing(hostInfo); ok {
		existing.update(hostInfo)
		hostInfo = existing
	}

	s.pool.addHost(hostInfo)
	s.policy.AddHost(hostInfo)
	hostInfo.setState(NodeUp)

	if s.control != nil && !s.cfg.IgnorePeerAddr {
		s.hostSource.refreshRing()
	}
}

func (s *Session) handleRemovedNode(ip net.IP, port int) {
	// we remove all nodes but only add ones which pass the filter
	addr := ip.String()

	host := s.ring.getHost(addr)
	if host == nil {
		host = &HostInfo{peer: addr}
	}

	if s.cfg.HostFilter != nil && !s.cfg.HostFilter.Accept(host) {
		return
	}

	host.setState(NodeDown)
	s.policy.RemoveHost(addr)
	s.pool.removeHost(addr)
	s.ring.removeHost(addr)

	if !s.cfg.IgnorePeerAddr {
		s.hostSource.refreshRing()
	}
}

func (s *Session) handleNodeUp(ip net.IP, port int, waitForBinary bool) {
	if gocqlDebug {
		log.Printf("gocql: Session.handleNodeUp: %s:%d\n", ip.String(), port)
	}
	addr := ip.String()
	host := s.ring.getHost(addr)
	if host != nil {
		if s.cfg.IgnorePeerAddr && host.Peer() != addr {
			host.setPeer(addr)
		}

		if s.cfg.HostFilter != nil {
			if !s.cfg.HostFilter.Accept(host) {
				return
			}
		} else if !s.cfg.Discovery.matchFilter(host) {
			// TODO: remove this when the host selection policy is more sophisticated
			return
		}

		if t := host.Version().nodeUpDelay(); t > 0 && waitForBinary {
			time.Sleep(t)
		}

		host.setPort(port)
		s.pool.hostUp(host)
		s.policy.HostUp(host)
		host.setState(NodeUp)
		return
	}

	s.handleNewNode(ip, port, waitForBinary)
}

func (s *Session) handleNodeDown(ip net.IP, port int) {
	if gocqlDebug {
		log.Printf("gocql: Session.handleNodeDown: %s:%d\n", ip.String(), port)
	}
	addr := ip.String()
	host := s.ring.getHost(addr)
	if host == nil {
		host = &HostInfo{peer: addr}
	}

	if s.cfg.HostFilter != nil && !s.cfg.HostFilter.Accept(host) {
		return
	}

	host.setState(NodeDown)
	s.policy.HostDown(addr)
	s.pool.hostDown(addr)
}
