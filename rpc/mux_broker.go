package rpc

import (
	"encoding/binary"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hashicorp/yamux"
)

// muxBroker is responsible for brokering multiplexed connections by unique ID.
//
// This allows a plugin to request a channel with a specific ID to connect to
// or accept a connection from, and the broker handles the details of
// holding these channels open while they're being negotiated.
type muxBroker struct {
	nextId  uint32
	session *yamux.Session
	streams map[uint32]*muxBrokerPending

	sync.Mutex
}

type muxBrokerPending struct {
	ch     chan net.Conn
	doneCh chan struct{}
}

func newMuxBroker(s *yamux.Session) *muxBroker {
	return &muxBroker{
		session: s,
		streams: make(map[uint32]*muxBrokerPending),
	}
}

// Accept accepts a connection by ID.
//
// This should not be called multiple times with the same ID at one time.
func (m *muxBroker) Accept(id uint32) (net.Conn, error) {
	var c net.Conn
	p := m.getStream(id)
	select {
	case c = <-p.ch:
		close(p.doneCh)
	case <-time.After(5 * time.Second):
		m.Lock()
		defer m.Unlock()
		delete(m.streams, id)

		return nil, fmt.Errorf("timeout waiting for accept")
	}

	// Ack our connection
	if err := binary.Write(c, binary.LittleEndian, id); err != nil {
		c.Close()
		return nil, err
	}

	return c, nil
}

// Close closes the connection and all sub-connections.
func (m *muxBroker) Close() error {
	return m.session.Close()
}

// Dial opens a connection by ID.
func (m *muxBroker) Dial(id uint32) (net.Conn, error) {
	// Open the stream
	stream, err := m.session.OpenStream()
	if err != nil {
		return nil, err
	}

	// Write the stream ID onto the wire.
	if err := binary.Write(stream, binary.LittleEndian, id); err != nil {
		stream.Close()
		return nil, err
	}

	// Read the ack that we connected. Then we're off!
	var ack uint32
	if err := binary.Read(stream, binary.LittleEndian, &ack); err != nil {
		stream.Close()
		return nil, err
	}
	if ack != id {
		stream.Close()
		return nil, fmt.Errorf("bad ack: %d (expected %d)", ack, id)
	}

	return stream, nil
}

// NextId returns a unique ID to use next.
func (m *muxBroker) NextId() uint32 {
	return atomic.AddUint32(&m.nextId, 1)
}

// Run starts the brokering and should be executed in a goroutine, since it
// blocks forever, or until the session closes.
func (m *muxBroker) Run() {
	for {
		stream, err := m.session.AcceptStream()
		if err != nil {
			// Once we receive an error, just exit
			break
		}

		// Read the stream ID from the stream
		var id uint32
		if err := binary.Read(stream, binary.LittleEndian, &id); err != nil {
			stream.Close()
			continue
		}

		// Initialize the waiter
		p := m.getStream(id)
		select {
		case p.ch <- stream:
		default:
		}

		// Wait for a timeout
		go m.timeoutWait(id, p)
	}
}

func (m *muxBroker) getStream(id uint32) *muxBrokerPending {
	m.Lock()
	defer m.Unlock()

	p, ok := m.streams[id]
	if ok {
		return p
	}

	m.streams[id] = &muxBrokerPending{
		ch:     make(chan net.Conn, 1),
		doneCh: make(chan struct{}),
	}
	return m.streams[id]
}

func (m *muxBroker) timeoutWait(id uint32, p *muxBrokerPending) {
	// Wait for the stream to either be picked up and connected, or
	// for a timeout.
	timeout := false
	select {
	case <-p.doneCh:
	case <-time.After(5 * time.Second):
		timeout = true
	}

	m.Lock()
	defer m.Unlock()

	// Delete the stream so no one else can grab it
	delete(m.streams, id)

	// If we timed out, then check if we have a channel in the buffer,
	// and if so, close it.
	if timeout {
		select {
		case s := <-p.ch:
			s.Close()
		}
	}
}
