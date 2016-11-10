// Package rpcproxy provides a proxy interface to Nomad Servers.  The
// RPCProxy periodically shuffles which server a Nomad Client communicates
// with in order to redistribute load across Nomad Servers.  Nomad Servers
// that fail an RPC request are automatically cycled to the end of the list
// until the server list is reshuffled.
//
// The rpcproxy package does not provide any external API guarantees and
// should be called only by `hashicorp/nomad`.
package rpcproxy

import (
	"fmt"
	"log"
	"math/rand"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hashicorp/consul/lib"
	"github.com/hashicorp/nomad/nomad/structs"
)

const (
	// clientRPCJitterFraction determines the amount of jitter added to
	// clientRPCMinReuseDuration before a connection is expired and a new
	// connection is established in order to rebalance load across Nomad
	// servers.  The cluster-wide number of connections per second from
	// rebalancing is applied after this jitter to ensure the CPU impact
	// is always finite.  See newRebalanceConnsPerSecPerServer's comment
	// for additional commentary.
	//
	// For example, in a 10K Nomad cluster with 5x servers, this default
	// averages out to ~13 new connections from rebalancing per server
	// per second.
	clientRPCJitterFraction = 2

	// clientRPCMinReuseDuration controls the minimum amount of time RPC
	// queries are sent over an established connection to a single server
	clientRPCMinReuseDuration = 600 * time.Second

	// Limit the number of new connections a server receives per second
	// for connection rebalancing.  This limit caps the load caused by
	// continual rebalancing efforts when a cluster is in equilibrium.  A
	// lower value comes at the cost of increased recovery time after a
	// partition.  This parameter begins to take effect when there are
	// more than ~48K clients querying 5x servers or at lower server
	// counts when there is a partition.
	//
	// For example, in a 100K Nomad cluster with 5x servers, it will take
	// ~5min for all servers to rebalance their connections.  If 99,995
	// agents are in the minority talking to only one server, it will
	// take ~26min for all servers to rebalance.  A 10K cluster in the
	// same scenario will take ~2.6min to rebalance.
	newRebalanceConnsPerSecPerServer = 64

	// rpcAPIMismatchLogRate determines the rate at which log entries are
	// emitted when the client and server's API versions are mismatched.
	rpcAPIMismatchLogRate = 3 * time.Hour
)

// NomadConfigInfo is an interface wrapper around this Nomad Agent's
// configuration to prevents a cyclic import dependency.
type NomadConfigInfo interface {
	Datacenter() string
	RPCMajorVersion() int
	RPCMinorVersion() int
	Region() string
}

// Pinger is an interface wrapping client.ConnPool to prevent a
// cyclic import dependency
type Pinger interface {
	PingNomadServer(region string, apiMajorVersion int, s *ServerEndpoint) (bool, error)
}

// serverList is an array of Nomad Servers.  The first server in the list is
// the active server.
//
// NOTE(sean@): We are explicitly relying on the fact that serverList will be
// copied onto the stack by atomic.Value.  Please keep this structure light.
type serverList struct {
	L []*ServerEndpoint
}

// RPCProxy is the manager type responsible for returning and managing Nomad
// addresses.
type RPCProxy struct {
	// activatedList manages the list of Nomad Servers that are eligible
	// to be queried by the Client agent.
	activatedList     atomic.Value
	activatedListLock sync.Mutex

	// primaryServers is a list of servers found in the last heartbeat.
	// primaryServers are periodically reshuffled.  Covered by
	// serverListLock.
	primaryServers serverList

	// backupServers is a list of fallback servers.  These servers are
	// appended to the RPCProxy's serverList, but are never shuffled with
	// the list of servers discovered via the Nomad heartbeat.  Covered
	// by serverListLock.
	backupServers serverList

	// serverListLock covers both backupServers and primaryServers.  If
	// it is necessary to hold serverListLock and listLock, obtain an
	// exclusive lock on serverListLock before listLock.
	serverListLock sync.RWMutex

	leaderAddr string
	numNodes   int

	// rebalanceTimer controls the duration of the rebalance interval
	rebalanceTimer *time.Timer

	// shutdownCh is a copy of the channel in nomad.Client
	shutdownCh chan struct{}

	logger *log.Logger

	configInfo NomadConfigInfo

	// rpcAPIMismatchThrottle regulates the rate at which warning
	// messages are emitted in the event of an API mismatch between the
	// clients and servers.
	rpcAPIMismatchThrottle map[string]time.Time

	// connPoolPinger is used to test the health of a server in the
	// connection pool.  Pinger is an interface that wraps
	// client.ConnPool.
	connPoolPinger Pinger
}

// NewRPCProxy is the only way to safely create a new RPCProxy.
func NewRPCProxy(logger *log.Logger, shutdownCh chan struct{}, configInfo NomadConfigInfo, connPoolPinger Pinger) *RPCProxy {
	p := &RPCProxy{
		logger:         logger,
		configInfo:     configInfo,     // can't pass *nomad.Client: import cycle
		connPoolPinger: connPoolPinger, // can't pass *nomad.ConnPool: import cycle
		rebalanceTimer: time.NewTimer(clientRPCMinReuseDuration),
		shutdownCh:     shutdownCh,
	}

	l := serverList{}
	l.L = make([]*ServerEndpoint, 0)
	p.saveServerList(l)
	return p
}

// activateEndpoint adds an endpoint to the RPCProxy's active serverList.
// Returns true if the server was added, returns false if the server already
// existed in the RPCProxy's serverList.
func (p *RPCProxy) activateEndpoint(s *ServerEndpoint) bool {
	l := p.getServerList()

	// Check if this server is known
	found := false
	for idx, existing := range l.L {
		if existing.Name == s.Name {
			newServers := make([]*ServerEndpoint, len(l.L))
			copy(newServers, l.L)

			// Overwrite the existing server details in order to
			// possibly update metadata (e.g. server version)
			newServers[idx] = s

			l.L = newServers
			found = true
			break
		}
	}

	// Add to the list if not known
	if !found {
		newServers := make([]*ServerEndpoint, len(l.L), len(l.L)+1)
		copy(newServers, l.L)
		newServers = append(newServers, s)
		l.L = newServers
	}

	p.saveServerList(l)

	return !found
}

// SetBackupServers sets a list of Nomad Servers to be used in the event that
// the Nomad Agent lost contact with the list of Nomad Servers provided via
// the Nomad Agent's heartbeat.  If available, the backup servers are
// populated via Consul.
func (p *RPCProxy) SetBackupServers(addrs []string) error {
	l := make([]*ServerEndpoint, 0, len(addrs))
	for _, s := range addrs {
		s, err := NewServerEndpoint(s)
		if err != nil {
			p.logger.Printf("[WARN] client.rpcproxy: unable to create backup server %+q: %v", s, err)
			return fmt.Errorf("unable to create new backup server from %+q: %v", s, err)
		}
		l = append(l, s)
	}

	p.serverListLock.Lock()
	p.backupServers.L = l
	p.serverListLock.Unlock()

	p.activatedListLock.Lock()
	defer p.activatedListLock.Unlock()
	for _, s := range l {
		p.activateEndpoint(s)
	}

	return nil
}

// AddPrimaryServer takes the RPC address of a Nomad server, creates a new
// endpoint, and adds it to both the primaryServers list and the active
// serverList used in the RPC Proxy.  If the endpoint is not known by the
// RPCProxy, appends the endpoint to the list.  The new endpoint will begin
// seeing use after the rebalance timer fires (or enough servers fail
// organically).  Any values in the primary server list are overridden by the
// next successful heartbeat.
func (p *RPCProxy) AddPrimaryServer(rpcAddr string) *ServerEndpoint {
	s, err := NewServerEndpoint(rpcAddr)
	if err != nil {
		p.logger.Printf("[WARN] client.rpcproxy: unable to create new primary server from endpoint %+q: %v", rpcAddr, err)
		return nil
	}

	k := s.Key()
	p.serverListLock.Lock()
	if serverExists := p.primaryServers.serverExistByKey(k); serverExists {
		p.serverListLock.Unlock()
		return s
	}
	p.primaryServers.L = append(p.primaryServers.L, s)
	p.serverListLock.Unlock()

	p.activatedListLock.Lock()
	p.activateEndpoint(s)
	p.activatedListLock.Unlock()

	return s
}

// cycleServers returns a new list of servers that has dequeued the first
// server and enqueued it at the end of the list.  cycleServers assumes the
// caller is holding the listLock.  cycleServer does not test or ping
// the next server inline.  cycleServer may be called when the environment
// has just entered an unhealthy situation and blocking on a server test is
// less desirable than just returning the next server in the firing line.  If
// the next server fails, it will fail fast enough and cycleServer will be
// called again.
func (l *serverList) cycleServer() (servers []*ServerEndpoint) {
	numServers := len(l.L)
	if numServers < 2 {
		return servers // No action required
	}

	newServers := make([]*ServerEndpoint, 0, numServers)
	newServers = append(newServers, l.L[1:]...)
	newServers = append(newServers, l.L[0])

	return newServers
}

// serverExistByKey performs a search to see if a server exists in the
// serverList.  Assumes the caller is holding at least a read lock.
func (l *serverList) serverExistByKey(targetKey *EndpointKey) bool {
	var found bool
	for _, server := range l.L {
		if targetKey.Equal(server.Key()) {
			found = true
		}
	}
	return found
}

// removeServerByKey performs an inline removal of the first matching server
func (l *serverList) removeServerByKey(targetKey *EndpointKey) {
	for i, s := range l.L {
		if targetKey.Equal(s.Key()) {
			copy(l.L[i:], l.L[i+1:])
			l.L[len(l.L)-1] = nil
			l.L = l.L[:len(l.L)-1]
			return
		}
	}
}

// shuffleServers shuffles the server list in place
func (l *serverList) shuffleServers() {
	for i := len(l.L) - 1; i > 0; i-- {
		j := rand.Int31n(int32(i + 1))
		l.L[i], l.L[j] = l.L[j], l.L[i]
	}
}

// String returns a string representation of serverList
func (l *serverList) String() string {
	if len(l.L) == 0 {
		return fmt.Sprintf("empty server list")
	}

	serverStrs := make([]string, 0, len(l.L))
	for _, server := range l.L {
		serverStrs = append(serverStrs, server.String())
	}

	return fmt.Sprintf("[%s]", strings.Join(serverStrs, ", "))
}

// FindServer takes out an internal "read lock" and searches through the list
// of servers to find a "healthy" server.  If the server is actually
// unhealthy, we rely on heartbeats to detect this and remove the node from
// the server list.  If the server at the front of the list has failed or
// fails during an RPC call, it is rotated to the end of the list.  If there
// are no servers available, return nil.
func (p *RPCProxy) FindServer() *ServerEndpoint {
	l := p.getServerList()
	numServers := len(l.L)
	if numServers == 0 {
		p.logger.Printf("[WARN] client.rpcproxy: No servers available")
		return nil
	}

	// Return whatever is at the front of the list because it is
	// assumed to be the oldest in the server list (unless -
	// hypothetically - the server list was rotated right after a
	// server was added).
	return l.L[0]
}

// getServerList is a convenience method which hides the locking semantics
// of atomic.Value from the caller.
func (p *RPCProxy) getServerList() serverList {
	return p.activatedList.Load().(serverList)
}

// saveServerList is a convenience method which hides the locking semantics
// of atomic.Value from the caller.
func (p *RPCProxy) saveServerList(l serverList) {
	p.activatedList.Store(l)
}

// LeaderAddr returns the current leader address.  If an empty string, then
// the Nomad Server for this Nomad Agent is in the minority or the Nomad
// Servers are in the middle of an election.
func (p *RPCProxy) LeaderAddr() string {
	p.activatedListLock.Lock()
	defer p.activatedListLock.Unlock()
	return p.leaderAddr
}

// NotifyFailedServer marks the passed in server as "failed" by rotating it
// to the end of the server list.
func (p *RPCProxy) NotifyFailedServer(s *ServerEndpoint) {
	l := p.getServerList()

	// If the server being failed is not the first server on the list,
	// this is a noop.  If, however, the server is failed and first on
	// the list, acquire the lock, retest, and take the penalty of moving
	// the server to the end of the list.

	// Only rotate the server list when there is more than one server
	if len(l.L) > 1 && l.L[0] == s {
		// Grab a lock, retest, and take the hit of cycling the first
		// server to the end.
		p.activatedListLock.Lock()
		defer p.activatedListLock.Unlock()
		l = p.getServerList()

		if len(l.L) > 1 && l.L[0] == s {
			l.L = l.cycleServer()
			p.saveServerList(l)
		}
	}
}

// NumNodes returns the estimated number of nodes according to the last Nomad
// Heartbeat.
func (p *RPCProxy) NumNodes() int {
	return p.numNodes
}

// NumServers takes out an internal "read lock" and returns the number of
// servers.  numServers includes both healthy and unhealthy servers.
func (p *RPCProxy) NumServers() int {
	l := p.getServerList()
	return len(l.L)
}

// RebalanceServers shuffles the list of servers on this agent.  The server
// at the front of the list is selected for the next RPC.  RPC calls that
// fail for a particular server are rotated to the end of the list.  This
// method reshuffles the list periodically in order to redistribute work
// across all known Nomad servers (i.e. guarantee that the order of servers
// in the server list is not positively correlated with the age of a server
// in the Nomad cluster).  Periodically shuffling the server list prevents
// long-lived clients from fixating on long-lived servers.
//
// Unhealthy servers are removed from the server list during the next client
// heartbeat.  Before the newly shuffled server list is saved, the new remote
// endpoint is tested to ensure its responsive.
func (p *RPCProxy) RebalanceServers() {
	var serverListLocked bool
	p.serverListLock.Lock()
	serverListLocked = true
	defer func() {
		if serverListLocked {
			p.serverListLock.Unlock()
		}
	}()

	// Early abort if there is nothing to shuffle
	if (len(p.primaryServers.L) + len(p.backupServers.L)) < 2 {
		return
	}

	// Shuffle server lists independently
	p.primaryServers.shuffleServers()
	p.backupServers.shuffleServers()

	// Create a new merged serverList
	type targetServer struct {
		server *ServerEndpoint
		// 'p' == Primary Server
		// 's' == Secondary/Backup Server
		// 'b' == Both
		state byte
	}
	mergedList := make(map[EndpointKey]*targetServer, len(p.primaryServers.L)+len(p.backupServers.L))
	for _, s := range p.primaryServers.L {
		mergedList[*s.Key()] = &targetServer{server: s, state: 'p'}
	}
	for _, s := range p.backupServers.L {
		k := s.Key()
		_, found := mergedList[*k]
		if found {
			mergedList[*k].state = 'b'
		} else {
			mergedList[*k] = &targetServer{server: s, state: 's'}
		}
	}

	l := &serverList{L: make([]*ServerEndpoint, 0, len(mergedList))}
	for _, s := range p.primaryServers.L {
		l.L = append(l.L, s)
	}
	for _, v := range mergedList {
		if v.state != 's' {
			continue
		}
		l.L = append(l.L, v.server)
	}

	// Release the lock before we begin transition to operations on the
	// network timescale and attempt to ping servers.  A copy of the
	// servers has been made at this point.
	p.serverListLock.Unlock()
	serverListLocked = false

	// Iterate through the shuffled server list to find an assumed
	// healthy server.  NOTE: Do not iterate on the list directly because
	// this loop mutates the server list in-place.
	var foundHealthyServer bool
	for i := 0; i < len(l.L); i++ {
		// Always test the first server.  Failed servers are cycled
		// and eventually removed from the list when Nomad heartbeats
		// detect the failed node.
		selectedServer := l.L[0]

		ok, err := p.connPoolPinger.PingNomadServer(p.configInfo.Region(), p.configInfo.RPCMajorVersion(), selectedServer)
		if ok {
			foundHealthyServer = true
			break
		}
		p.logger.Printf(`[DEBUG] client.rpcproxy: pinging server "%s" failed: %s`, selectedServer.String(), err)

		l.cycleServer()
	}

	// If no healthy servers were found, sleep and wait for the admin to
	// join this node to a server and begin receiving heartbeats with an
	// updated list of Nomad servers.  Or Consul will begin advertising a
	// new server in the nomad service (Nomad server service).
	if !foundHealthyServer {
		p.logger.Printf("[DEBUG] client.rpcproxy: No healthy servers during rebalance, aborting")
		return
	}

	// Verify that all servers are present.  Reconcile will save the
	// final serverList.
	if p.reconcileServerList(l) {
		p.logger.Printf("[TRACE] client.rpcproxy: Rebalanced %d servers, next active server is %s", len(l.L), l.L[0].String())
	} else {
		// reconcileServerList failed because Nomad removed the
		// server that was at the front of the list that had
		// successfully been Ping'ed.  Between the Ping and
		// reconcile, a Nomad heartbeat removed the node.
		//
		// Instead of doing any heroics, "freeze in place" and
		// continue to use the existing connection until the next
		// rebalance occurs.
	}

	return
}

// reconcileServerList returns true when the first server in serverList
// (l) exists in the receiver's serverList (p).  If true, the merged
// serverList (l) is stored as the receiver's serverList (p).  Returns
// false if the first server in p does not exist in the passed in list (l)
// (i.e. was removed by Nomad during a PingNomadServer() call.  Newly added
// servers are appended to the list and other missing servers are removed
// from the list.
func (p *RPCProxy) reconcileServerList(l *serverList) bool {
	p.activatedListLock.Lock()
	defer p.activatedListLock.Unlock()

	// newServerList is a serverList that has been kept up-to-date with
	// join and leave events.
	newServerList := p.getServerList()

	// If a Nomad heartbeat removed all nodes, or there is no selected
	// server (zero nodes in serverList), abort early.
	if len(newServerList.L) == 0 || len(l.L) == 0 {
		return false
	}

	type targetServer struct {
		server *ServerEndpoint

		//   'b' == both
		//   'o' == original
		//   'n' == new
		state byte
	}
	mergedList := make(map[EndpointKey]*targetServer, len(l.L))
	for _, s := range l.L {
		mergedList[*s.Key()] = &targetServer{server: s, state: 'o'}
	}
	for _, s := range newServerList.L {
		k := s.Key()
		_, found := mergedList[*k]
		if found {
			mergedList[*k].state = 'b'
		} else {
			mergedList[*k] = &targetServer{server: s, state: 'n'}
		}
	}

	// Ensure the selected server has not been removed by a heartbeat
	selectedServerKey := l.L[0].Key()
	if v, found := mergedList[*selectedServerKey]; found && v.state == 'o' {
		return false
	}

	// Append any new servers and remove any old servers
	for k, v := range mergedList {
		switch v.state {
		case 'b':
			// Do nothing, server exists in both
		case 'o':
			// Server has been removed
			l.removeServerByKey(&k)
		case 'n':
			// Server added
			l.L = append(l.L, v.server)
		default:
			panic("unknown merge list state")
		}
	}

	p.saveServerList(*l)
	return true
}

// RemoveServer takes out an internal write lock and removes a server from
// the activated server list.
func (p *RPCProxy) RemoveServer(s *ServerEndpoint) {
	// Lock hierarchy protocol dictates serverListLock is acquired first.
	p.serverListLock.Lock()
	defer p.serverListLock.Unlock()

	p.activatedListLock.Lock()
	defer p.activatedListLock.Unlock()
	l := p.getServerList()

	k := s.Key()
	l.removeServerByKey(k)
	p.saveServerList(l)

	p.primaryServers.removeServerByKey(k)
	p.backupServers.removeServerByKey(k)
}

// refreshServerRebalanceTimer is only called once p.rebalanceTimer expires.
func (p *RPCProxy) refreshServerRebalanceTimer() time.Duration {
	l := p.getServerList()
	numServers := len(l.L)
	// Limit this connection's life based on the size (and health) of the
	// cluster.  Never rebalance a connection more frequently than
	// connReuseLowWatermarkDuration, and make sure we never exceed
	// clusterWideRebalanceConnsPerSec operations/s across numLANMembers.
	clusterWideRebalanceConnsPerSec := float64(numServers * newRebalanceConnsPerSecPerServer)
	connReuseLowWatermarkDuration := clientRPCMinReuseDuration + lib.RandomStagger(clientRPCMinReuseDuration/clientRPCJitterFraction)
	numLANMembers := p.numNodes
	connRebalanceTimeout := lib.RateScaledInterval(clusterWideRebalanceConnsPerSec, connReuseLowWatermarkDuration, numLANMembers)

	p.rebalanceTimer.Reset(connRebalanceTimeout)
	return connRebalanceTimeout
}

// ResetRebalanceTimer resets the rebalance timer.  This method exists for
// testing and should not be used directly.
func (p *RPCProxy) ResetRebalanceTimer() {
	p.activatedListLock.Lock()
	defer p.activatedListLock.Unlock()
	p.rebalanceTimer.Reset(clientRPCMinReuseDuration)
}

// ServerRPCAddrs returns one RPC Address per server
func (p *RPCProxy) ServerRPCAddrs() []string {
	l := p.getServerList()
	serverAddrs := make([]string, 0, len(l.L))
	for _, s := range l.L {
		serverAddrs = append(serverAddrs, s.Addr.String())
	}
	return serverAddrs
}

// Run is used to start and manage the task of automatically shuffling and
// rebalancing the list of Nomad servers.  This maintenance only happens
// periodically based on the expiration of the timer.  Failed servers are
// automatically cycled to the end of the list.  New servers are appended to
// the list.  The order of the server list must be shuffled periodically to
// distribute load across all known and available Nomad servers.
func (p *RPCProxy) Run() {
	for {
		select {
		case <-p.rebalanceTimer.C:
			p.RebalanceServers()

			p.refreshServerRebalanceTimer()
		case <-p.shutdownCh:
			p.logger.Printf("[INFO] client.rpcproxy: shutting down")
			return
		}
	}
}

// RefreshServerLists is called when the Client receives an update from a
// Nomad Server.  The response from Nomad Client Heartbeats contain a list of
// Nomad Servers that the Nomad Client should use for RPC requests.
// RefreshServerLists does not rebalance its serverLists (that is handled
// elsewhere via a periodic timer).  New Nomad Servers learned via the
// heartbeat are appended to the RPCProxy's activated serverList.  Servers
// that are no longer present in the Heartbeat are removed immediately from
// all server lists.  Nomad Servers speaking a newer major or minor API
// version are filtered from the serverList.
func (p *RPCProxy) RefreshServerLists(servers []*structs.NodeServerInfo, numNodes int32, leaderRPCAddr string) error {
	// Merge all servers found in the response.  Servers in the response
	// with newer API versions are filtered from the list.  If the list
	// is missing an address found in the RPCProxy's server list, remove
	// it from the RPCProxy.

	p.serverListLock.Lock()
	defer p.serverListLock.Unlock()

	// Clear the backup server list when a heartbeat contains at least
	// one server.
	if len(servers) > 0 && len(p.backupServers.L) > 0 {
		p.backupServers.L = make([]*ServerEndpoint, 0, len(servers))
	}

	// 1) Create a map to reconcile the difference between
	// p.primaryServers and servers.
	type targetServer struct {
		server *ServerEndpoint

		//   'b' == both
		//   'o' == original
		//   'n' == new
		state byte
	}
	mergedPrimaryMap := make(map[EndpointKey]*targetServer, len(p.primaryServers.L)+len(servers))
	numOldServers := 0
	for _, s := range p.primaryServers.L {
		mergedPrimaryMap[*s.Key()] = &targetServer{server: s, state: 'o'}
		numOldServers++
	}
	numBothServers := 0
	var newServers bool
	for _, s := range servers {
		// Filter out servers using a newer API version.  Prevent
		// spamming the logs every heartbeat.
		//
		// TODO(sean@): Move the logging throttle logic into a
		// dedicated logging package so RPCProxy does not have to
		// perform this accounting.
		if int32(p.configInfo.RPCMajorVersion()) < s.RPCMajorVersion ||
			(int32(p.configInfo.RPCMajorVersion()) == s.RPCMajorVersion &&
				int32(p.configInfo.RPCMinorVersion()) < s.RPCMinorVersion) {
			now := time.Now()
			t, ok := p.rpcAPIMismatchThrottle[s.RPCAdvertiseAddr]
			if ok && t.After(now) {
				continue
			}

			p.logger.Printf("[WARN] client.rpcproxy: API mismatch between client version (v%d.%d) and server version (v%d.%d), ignoring server %+q", p.configInfo.RPCMajorVersion(), p.configInfo.RPCMinorVersion(), s.RPCMajorVersion, s.RPCMinorVersion, s.RPCAdvertiseAddr)
			p.rpcAPIMismatchThrottle[s.RPCAdvertiseAddr] = now.Add(rpcAPIMismatchLogRate)
			continue
		}

		server, err := NewServerEndpoint(s.RPCAdvertiseAddr)
		if err != nil {
			p.logger.Printf("[WARN] client.rpcproxy: Unable to create a server from %+q: %v", s.RPCAdvertiseAddr, err)
			continue
		}

		// Nomad servers in different datacenters are automatically
		// added to the backup server list.
		if s.Datacenter != p.configInfo.Datacenter() {
			p.backupServers.L = append(p.backupServers.L, server)
			continue
		}

		k := server.Key()
		_, found := mergedPrimaryMap[*k]
		if found {
			mergedPrimaryMap[*k].state = 'b'
			numBothServers++
		} else {
			mergedPrimaryMap[*k] = &targetServer{server: server, state: 'n'}
			newServers = true
		}
	}

	// Short-circuit acquiring listLock if nothing changed
	if !newServers && numOldServers == numBothServers {
		return nil
	}

	p.activatedListLock.Lock()
	defer p.activatedListLock.Unlock()
	newServerCfg := p.getServerList()
	for k, v := range mergedPrimaryMap {
		switch v.state {
		case 'b':
			// Do nothing, server exists in both
		case 'o':
			// Server has been removed

			// TODO(sean@): Teach Nomad servers how to remove
			// themselves from their heartbeat in order to
			// gracefully drain their clients over the next
			// cluster's max rebalanceTimer duration.  Without
			// this enhancement, if a server being shutdown and
			// it is the first in serverList, the client will
			// fail its next RPC connection.
			p.primaryServers.removeServerByKey(&k)
			newServerCfg.removeServerByKey(&k)
		case 'n':
			// Server added.  Append it to both lists
			// immediately.  The server should only go into
			// active use in the event of a failure or after a
			// rebalance occurs.
			p.primaryServers.L = append(p.primaryServers.L, v.server)
			newServerCfg.L = append(newServerCfg.L, v.server)
		default:
			panic("unknown merge list state")
		}
	}

	p.numNodes = int(numNodes)
	p.leaderAddr = leaderRPCAddr
	p.saveServerList(newServerCfg)

	return nil
}
