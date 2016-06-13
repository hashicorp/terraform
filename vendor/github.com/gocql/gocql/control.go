package gocql

import (
	crand "crypto/rand"
	"errors"
	"fmt"
	"golang.org/x/net/context"
	"log"
	"math/rand"
	"net"
	"strconv"
	"sync/atomic"
	"time"
)

var (
	randr *rand.Rand
)

func init() {
	b := make([]byte, 4)
	if _, err := crand.Read(b); err != nil {
		panic(fmt.Sprintf("unable to seed random number generator: %v", err))
	}

	randr = rand.New(rand.NewSource(int64(readInt(b))))
}

// Ensure that the atomic variable is aligned to a 64bit boundary
// so that atomic operations can be applied on 32bit architectures.
type controlConn struct {
	session *Session
	conn    atomic.Value

	retry RetryPolicy

	started int32
	quit    chan struct{}
}

func createControlConn(session *Session) *controlConn {
	control := &controlConn{
		session: session,
		quit:    make(chan struct{}),
		retry:   &SimpleRetryPolicy{NumRetries: 3},
	}

	control.conn.Store((*Conn)(nil))

	return control
}

func (c *controlConn) heartBeat() {
	if !atomic.CompareAndSwapInt32(&c.started, 0, 1) {
		return
	}

	sleepTime := 1 * time.Second

	for {
		select {
		case <-c.quit:
			return
		case <-time.After(sleepTime):
		}

		resp, err := c.writeFrame(&writeOptionsFrame{})
		if err != nil {
			goto reconn
		}

		switch resp.(type) {
		case *supportedFrame:
			// Everything ok
			sleepTime = 5 * time.Second
			continue
		case error:
			goto reconn
		default:
			panic(fmt.Sprintf("gocql: unknown frame in response to options: %T", resp))
		}

	reconn:
		// try to connect a bit faster
		sleepTime = 1 * time.Second
		c.reconnect(true)
		// time.Sleep(5 * time.Second)
		continue
	}
}

func hostInfo(addr string, defaultPort int) (*HostInfo, error) {
	var port int
	host, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		host = addr
		port = defaultPort
	} else {
		port, err = strconv.Atoi(portStr)
		if err != nil {
			return nil, err
		}
	}

	return &HostInfo{peer: host, port: port}, nil
}

func (c *controlConn) shuffleDial(endpoints []string) (conn *Conn, err error) {
	perm := randr.Perm(len(endpoints))
	shuffled := make([]string, len(endpoints))

	for i, endpoint := range endpoints {
		shuffled[perm[i]] = endpoint
	}

	// shuffle endpoints so not all drivers will connect to the same initial
	// node.
	for _, addr := range shuffled {
		if addr == "" {
			return nil, fmt.Errorf("invalid address: %q", addr)
		}

		port := c.session.cfg.Port
		addr = JoinHostPort(addr, port)

		var host *HostInfo
		host, err = hostInfo(addr, port)
		if err != nil {
			return nil, fmt.Errorf("invalid address: %q: %v", addr, err)
		}

		hostInfo, _ := c.session.ring.addHostIfMissing(host)
		conn, err = c.session.connect(addr, c, hostInfo)
		if err == nil {
			return conn, err
		}

		log.Printf("gocql: unable to dial control conn %v: %v\n", addr, err)
	}

	if err != nil {
		return nil, err
	}

	return conn, nil
}

func (c *controlConn) connect(endpoints []string) error {
	if len(endpoints) == 0 {
		return errors.New("control: no endpoints specified")
	}

	conn, err := c.shuffleDial(endpoints)
	if err != nil {
		return fmt.Errorf("control: unable to connect to initial hosts: %v", err)
	}

	if err := c.setupConn(conn); err != nil {
		conn.Close()
		return fmt.Errorf("control: unable to setup connection: %v", err)
	}

	// we could fetch the initial ring here and update initial host data. So that
	// when we return from here we have a ring topology ready to go.

	go c.heartBeat()

	return nil
}

func (c *controlConn) setupConn(conn *Conn) error {
	if err := c.registerEvents(conn); err != nil {
		conn.Close()
		return err
	}

	c.conn.Store(conn)

	host, portstr, err := net.SplitHostPort(conn.conn.RemoteAddr().String())
	if err != nil {
		return err
	}
	port, err := strconv.Atoi(portstr)
	if err != nil {
		return err
	}

	c.session.handleNodeUp(net.ParseIP(host), port, false)

	return nil
}

func (c *controlConn) registerEvents(conn *Conn) error {
	var events []string

	if !c.session.cfg.Events.DisableTopologyEvents {
		events = append(events, "TOPOLOGY_CHANGE")
	}
	if !c.session.cfg.Events.DisableNodeStatusEvents {
		events = append(events, "STATUS_CHANGE")
	}
	if !c.session.cfg.Events.DisableSchemaEvents {
		events = append(events, "SCHEMA_CHANGE")
	}

	if len(events) == 0 {
		return nil
	}

	framer, err := conn.exec(context.Background(),
		&writeRegisterFrame{
			events: events,
		}, nil)
	if err != nil {
		return err
	}

	frame, err := framer.parseFrame()
	if err != nil {
		return err
	} else if _, ok := frame.(*readyFrame); !ok {
		return fmt.Errorf("unexpected frame in response to register: got %T: %v\n", frame, frame)
	}

	return nil
}

func (c *controlConn) reconnect(refreshring bool) {
	// TODO: simplify this function, use session.ring to get hosts instead of the
	// connection pool

	addr := c.addr()
	oldConn := c.conn.Load().(*Conn)
	if oldConn != nil {
		oldConn.Close()
	}

	var newConn *Conn
	if addr != "" {
		// try to connect to the old host
		conn, err := c.session.connect(addr, c, oldConn.host)
		if err != nil {
			// host is dead
			// TODO: this is replicated in a few places
			ip, portStr, _ := net.SplitHostPort(addr)
			port, _ := strconv.Atoi(portStr)
			c.session.handleNodeDown(net.ParseIP(ip), port)
		} else {
			newConn = conn
		}
	}

	// TODO: should have our own roundrobbin for hosts so that we can try each
	// in succession and guantee that we get a different host each time.
	if newConn == nil {
		host := c.session.ring.rrHost()
		if host == nil {
			c.connect(c.session.ring.endpoints)
			return
		}

		var err error
		newConn, err = c.session.connect(host.Peer(), c, host)
		if err != nil {
			// TODO: add log handler for things like this
			return
		}
	}

	if err := c.setupConn(newConn); err != nil {
		newConn.Close()
		log.Printf("gocql: control unable to register events: %v\n", err)
		return
	}

	if refreshring {
		c.session.hostSource.refreshRing()
	}
}

func (c *controlConn) HandleError(conn *Conn, err error, closed bool) {
	if !closed {
		return
	}

	oldConn := c.conn.Load().(*Conn)
	if oldConn != conn {
		return
	}

	c.reconnect(true)
}

func (c *controlConn) writeFrame(w frameWriter) (frame, error) {
	conn := c.conn.Load().(*Conn)
	if conn == nil {
		return nil, errNoControl
	}

	framer, err := conn.exec(context.Background(), w, nil)
	if err != nil {
		return nil, err
	}

	return framer.parseFrame()
}

func (c *controlConn) withConn(fn func(*Conn) *Iter) *Iter {
	const maxConnectAttempts = 5
	connectAttempts := 0

	for i := 0; i < maxConnectAttempts; i++ {
		conn := c.conn.Load().(*Conn)
		if conn == nil {
			if connectAttempts > maxConnectAttempts {
				break
			}

			connectAttempts++

			c.reconnect(false)
			continue
		}

		return fn(conn)
	}

	return &Iter{err: errNoControl}
}

// query will return nil if the connection is closed or nil
func (c *controlConn) query(statement string, values ...interface{}) (iter *Iter) {
	q := c.session.Query(statement, values...).Consistency(One).RoutingKey([]byte{})

	for {
		iter = c.withConn(func(conn *Conn) *Iter {
			return conn.executeQuery(q)
		})

		if gocqlDebug && iter.err != nil {
			log.Printf("control: error executing %q: %v\n", statement, iter.err)
		}

		q.attempts++
		if iter.err == nil || !c.retry.Attempt(q) {
			break
		}
	}

	return
}

func (c *controlConn) fetchHostInfo(addr net.IP, port int) (*HostInfo, error) {
	// TODO(zariel): we should probably move this into host_source or atleast
	// share code with it.
	hostname, _, err := net.SplitHostPort(c.addr())
	if err != nil {
		return nil, fmt.Errorf("unable to fetch host info, invalid conn addr: %q: %v", c.addr(), err)
	}

	isLocal := hostname == addr.String()

	var fn func(*HostInfo) error

	if isLocal {
		fn = func(host *HostInfo) error {
			// TODO(zariel): should we fetch rpc_address from here?
			iter := c.query("SELECT data_center, rack, host_id, tokens, release_version FROM system.local WHERE key='local'")
			iter.Scan(&host.dataCenter, &host.rack, &host.hostId, &host.tokens, &host.version)
			return iter.Close()
		}
	} else {
		fn = func(host *HostInfo) error {
			// TODO(zariel): should we fetch rpc_address from here?
			iter := c.query("SELECT data_center, rack, host_id, tokens, release_version FROM system.peers WHERE peer=?", addr)
			iter.Scan(&host.dataCenter, &host.rack, &host.hostId, &host.tokens, &host.version)
			return iter.Close()
		}
	}

	host := &HostInfo{
		port: port,
	}

	if err := fn(host); err != nil {
		return nil, err
	}
	host.peer = addr.String()

	return host, nil
}

func (c *controlConn) awaitSchemaAgreement() error {
	return c.withConn(func(conn *Conn) *Iter {
		return &Iter{err: conn.awaitSchemaAgreement()}
	}).err
}

func (c *controlConn) addr() string {
	conn := c.conn.Load().(*Conn)
	if conn == nil {
		return ""
	}
	return conn.addr
}

func (c *controlConn) close() {
	if atomic.CompareAndSwapInt32(&c.started, 1, -1) {
		c.quit <- struct{}{}
	}
	conn := c.conn.Load().(*Conn)
	if conn != nil {
		conn.Close()
	}
}

var errNoControl = errors.New("gocql: no control connection available")
