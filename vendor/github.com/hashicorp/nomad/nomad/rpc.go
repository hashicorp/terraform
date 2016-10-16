package nomad

import (
	"crypto/tls"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/rpc"
	"strings"
	"time"

	"github.com/armon/go-metrics"
	"github.com/hashicorp/consul/lib"
	"github.com/hashicorp/net-rpc-msgpackrpc"
	"github.com/hashicorp/nomad/nomad/state"
	"github.com/hashicorp/nomad/nomad/structs"
	"github.com/hashicorp/nomad/nomad/watch"
	"github.com/hashicorp/raft"
	"github.com/hashicorp/yamux"
)

type RPCType byte

const (
	rpcNomad     RPCType = 0x01
	rpcRaft              = 0x02
	rpcMultiplex         = 0x03
	rpcTLS               = 0x04
)

const (
	// maxQueryTime is used to bound the limit of a blocking query
	maxQueryTime = 300 * time.Second

	// defaultQueryTime is the amount of time we block waiting for a change
	// if no time is specified. Previously we would wait the maxQueryTime.
	defaultQueryTime = 300 * time.Second

	// jitterFraction is a the limit to the amount of jitter we apply
	// to a user specified MaxQueryTime. We divide the specified time by
	// the fraction. So 16 == 6.25% limit of jitter. This jitter is also
	// applied to RPCHoldTimeout.
	jitterFraction = 16

	// Warn if the Raft command is larger than this.
	// If it's over 1MB something is probably being abusive.
	raftWarnSize = 1024 * 1024

	// enqueueLimit caps how long we will wait to enqueue
	// a new Raft command. Something is probably wrong if this
	// value is ever reached. However, it prevents us from blocking
	// the requesting goroutine forever.
	enqueueLimit = 30 * time.Second
)

// NewClientCodec returns a new rpc.ClientCodec to be used to make RPC calls to
// the Nomad Server.
func NewClientCodec(conn io.ReadWriteCloser) rpc.ClientCodec {
	return msgpackrpc.NewCodecFromHandle(true, true, conn, structs.HashiMsgpackHandle)
}

// NewServerCodec returns a new rpc.ServerCodec to be used by the Nomad Server
// to handle rpcs.
func NewServerCodec(conn io.ReadWriteCloser) rpc.ServerCodec {
	return msgpackrpc.NewCodecFromHandle(true, true, conn, structs.HashiMsgpackHandle)
}

// listen is used to listen for incoming RPC connections
func (s *Server) listen() {
	for {
		// Accept a connection
		conn, err := s.rpcListener.Accept()
		if err != nil {
			if s.shutdown {
				return
			}
			s.logger.Printf("[ERR] nomad.rpc: failed to accept RPC conn: %v", err)
			continue
		}

		go s.handleConn(conn, false)
		metrics.IncrCounter([]string{"nomad", "rpc", "accept_conn"}, 1)
	}
}

// handleConn is used to determine if this is a Raft or
// Nomad type RPC connection and invoke the correct handler
func (s *Server) handleConn(conn net.Conn, isTLS bool) {
	// Read a single byte
	buf := make([]byte, 1)
	if _, err := conn.Read(buf); err != nil {
		if err != io.EOF {
			s.logger.Printf("[ERR] nomad.rpc: failed to read byte: %v", err)
		}
		conn.Close()
		return
	}

	// Enforce TLS if VerifyIncoming is set
	if s.config.RequireTLS && !isTLS && RPCType(buf[0]) != rpcTLS {
		s.logger.Printf("[WARN] nomad.rpc: Non-TLS connection attempted with RequireTLS set")
		conn.Close()
		return
	}

	// Switch on the byte
	switch RPCType(buf[0]) {
	case rpcNomad:
		s.handleNomadConn(conn)

	case rpcRaft:
		metrics.IncrCounter([]string{"nomad", "rpc", "raft_handoff"}, 1)
		s.raftLayer.Handoff(conn)

	case rpcMultiplex:
		s.handleMultiplex(conn)

	case rpcTLS:
		if s.rpcTLS == nil {
			s.logger.Printf("[WARN] nomad.rpc: TLS connection attempted, server not configured for TLS")
			conn.Close()
			return
		}
		conn = tls.Server(conn, s.rpcTLS)
		s.handleConn(conn, true)

	default:
		s.logger.Printf("[ERR] nomad.rpc: unrecognized RPC byte: %v", buf[0])
		conn.Close()
		return
	}
}

// handleMultiplex is used to multiplex a single incoming connection
// using the Yamux multiplexer
func (s *Server) handleMultiplex(conn net.Conn) {
	defer conn.Close()
	conf := yamux.DefaultConfig()
	conf.LogOutput = s.config.LogOutput
	server, _ := yamux.Server(conn, conf)
	for {
		sub, err := server.Accept()
		if err != nil {
			if err != io.EOF {
				s.logger.Printf("[ERR] nomad.rpc: multiplex conn accept failed: %v", err)
			}
			return
		}
		go s.handleNomadConn(sub)
	}
}

// handleNomadConn is used to service a single Nomad RPC connection
func (s *Server) handleNomadConn(conn net.Conn) {
	defer conn.Close()
	rpcCodec := NewServerCodec(conn)
	for {
		select {
		case <-s.shutdownCh:
			return
		default:
		}

		if err := s.rpcServer.ServeRequest(rpcCodec); err != nil {
			if err != io.EOF && !strings.Contains(err.Error(), "closed") {
				s.logger.Printf("[ERR] nomad.rpc: RPC error: %v (%v)", err, conn)
				metrics.IncrCounter([]string{"nomad", "rpc", "request_error"}, 1)
			}
			return
		}
		metrics.IncrCounter([]string{"nomad", "rpc", "request"}, 1)
	}
}

// forward is used to forward to a remote region or to forward to the local leader
// Returns a bool of if forwarding was performed, as well as any error
func (s *Server) forward(method string, info structs.RPCInfo, args interface{}, reply interface{}) (bool, error) {
	var firstCheck time.Time

	region := info.RequestRegion()
	if region == "" {
		return true, fmt.Errorf("missing target RPC")
	}

	// Handle region forwarding
	if region != s.config.Region {
		err := s.forwardRegion(region, method, args, reply)
		return true, err
	}

	// Check if we can allow a stale read
	if info.IsRead() && info.AllowStaleRead() {
		return false, nil
	}

CHECK_LEADER:
	// Find the leader
	isLeader, remoteServer := s.getLeader()

	// Handle the case we are the leader
	if isLeader {
		return false, nil
	}

	// Handle the case of a known leader
	if remoteServer != nil {
		err := s.forwardLeader(remoteServer, method, args, reply)
		return true, err
	}

	// Gate the request until there is a leader
	if firstCheck.IsZero() {
		firstCheck = time.Now()
	}
	if time.Now().Sub(firstCheck) < s.config.RPCHoldTimeout {
		jitter := lib.RandomStagger(s.config.RPCHoldTimeout / jitterFraction)
		select {
		case <-time.After(jitter):
			goto CHECK_LEADER
		case <-s.shutdownCh:
		}
	}

	// No leader found and hold time exceeded
	return true, structs.ErrNoLeader
}

// getLeader returns if the current node is the leader, and if not
// then it returns the leader which is potentially nil if the cluster
// has not yet elected a leader.
func (s *Server) getLeader() (bool, *serverParts) {
	// Check if we are the leader
	if s.IsLeader() {
		return true, nil
	}

	// Get the leader
	leader := s.raft.Leader()
	if leader == "" {
		return false, nil
	}

	// Lookup the server
	s.peerLock.RLock()
	server := s.localPeers[leader]
	s.peerLock.RUnlock()

	// Server could be nil
	return false, server
}

// forwardLeader is used to forward an RPC call to the leader, or fail if no leader
func (s *Server) forwardLeader(server *serverParts, method string, args interface{}, reply interface{}) error {
	// Handle a missing server
	if server == nil {
		return structs.ErrNoLeader
	}
	return s.connPool.RPC(s.config.Region, server.Addr, server.MajorVersion, method, args, reply)
}

// forwardRegion is used to forward an RPC call to a remote region, or fail if no servers
func (s *Server) forwardRegion(region, method string, args interface{}, reply interface{}) error {
	// Bail if we can't find any servers
	s.peerLock.RLock()
	servers := s.peers[region]
	if len(servers) == 0 {
		s.peerLock.RUnlock()
		s.logger.Printf("[WARN] nomad.rpc: RPC request for region '%s', no path found",
			region)
		return structs.ErrNoRegionPath
	}

	// Select a random addr
	offset := rand.Intn(len(servers))
	server := servers[offset]
	s.peerLock.RUnlock()

	// Forward to remote Nomad
	metrics.IncrCounter([]string{"nomad", "rpc", "cross-region", region}, 1)
	return s.connPool.RPC(region, server.Addr, server.MajorVersion, method, args, reply)
}

// raftApplyFuture is used to encode a message, run it through raft, and return the Raft future.
func (s *Server) raftApplyFuture(t structs.MessageType, msg interface{}) (raft.ApplyFuture, error) {
	buf, err := structs.Encode(t, msg)
	if err != nil {
		return nil, fmt.Errorf("Failed to encode request: %v", err)
	}

	// Warn if the command is very large
	if n := len(buf); n > raftWarnSize {
		s.logger.Printf("[WARN] nomad: Attempting to apply large raft entry (type %d) (%d bytes)", t, n)
	}

	future := s.raft.Apply(buf, enqueueLimit)
	return future, nil
}

// raftApply is used to encode a message, run it through raft, and return
// the FSM response along with any errors
func (s *Server) raftApply(t structs.MessageType, msg interface{}) (interface{}, uint64, error) {
	future, err := s.raftApplyFuture(t, msg)
	if err != nil {
		return nil, 0, err
	}
	if err := future.Error(); err != nil {
		return nil, 0, err
	}
	return future.Response(), future.Index(), nil
}

// setQueryMeta is used to populate the QueryMeta data for an RPC call
func (s *Server) setQueryMeta(m *structs.QueryMeta) {
	if s.IsLeader() {
		m.LastContact = 0
		m.KnownLeader = true
	} else {
		m.LastContact = time.Now().Sub(s.raft.LastContact())
		m.KnownLeader = (s.raft.Leader() != "")
	}
}

// blockingOptions is used to parameterize blockingRPC
type blockingOptions struct {
	queryOpts *structs.QueryOptions
	queryMeta *structs.QueryMeta
	watch     watch.Items
	run       func() error
}

// blockingRPC is used for queries that need to wait for a
// minimum index. This is used to block and wait for changes.
func (s *Server) blockingRPC(opts *blockingOptions) error {
	var timeout *time.Timer
	var notifyCh chan struct{}
	var state *state.StateStore

	// Fast path non-blocking
	if opts.queryOpts.MinQueryIndex == 0 {
		goto RUN_QUERY
	}

	// Restrict the max query time, and ensure there is always one
	if opts.queryOpts.MaxQueryTime > maxQueryTime {
		opts.queryOpts.MaxQueryTime = maxQueryTime
	} else if opts.queryOpts.MaxQueryTime <= 0 {
		opts.queryOpts.MaxQueryTime = defaultQueryTime
	}

	// Apply a small amount of jitter to the request
	opts.queryOpts.MaxQueryTime += lib.RandomStagger(opts.queryOpts.MaxQueryTime / jitterFraction)

	// Setup a query timeout
	timeout = time.NewTimer(opts.queryOpts.MaxQueryTime)

	// Setup the notify channel
	notifyCh = make(chan struct{}, 1)

	// Ensure we tear down any watchers on return
	state = s.fsm.State()
	defer func() {
		timeout.Stop()
		state.StopWatch(opts.watch, notifyCh)
	}()

REGISTER_NOTIFY:
	// Register the notification channel. This may be done
	// multiple times if we have not reached the target wait index.
	state.Watch(opts.watch, notifyCh)

RUN_QUERY:
	// Update the query meta data
	s.setQueryMeta(opts.queryMeta)

	// Run the query function
	metrics.IncrCounter([]string{"nomad", "rpc", "query"}, 1)
	err := opts.run()

	// Check for minimum query time
	if err == nil && opts.queryOpts.MinQueryIndex > 0 && opts.queryMeta.Index <= opts.queryOpts.MinQueryIndex {
		select {
		case <-notifyCh:
			goto REGISTER_NOTIFY
		case <-timeout.C:
		}
	}
	return err
}
