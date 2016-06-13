// Copyright (c) 2012 The gocql Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
//This file will be the future home for more policies
package gocql

import (
	"fmt"
	"log"
	"sync"
	"sync/atomic"

	"github.com/hailocab/go-hostpool"
)

// cowHostList implements a copy on write host list, its equivilent type is []*HostInfo
type cowHostList struct {
	list atomic.Value
	mu   sync.Mutex
}

func (c *cowHostList) String() string {
	return fmt.Sprintf("%+v", c.get())
}

func (c *cowHostList) get() []*HostInfo {
	// TODO(zariel): should we replace this with []*HostInfo?
	l, ok := c.list.Load().(*[]*HostInfo)
	if !ok {
		return nil
	}
	return *l
}

func (c *cowHostList) set(list []*HostInfo) {
	c.mu.Lock()
	c.list.Store(&list)
	c.mu.Unlock()
}

// add will add a host if it not already in the list
func (c *cowHostList) add(host *HostInfo) bool {
	c.mu.Lock()
	l := c.get()

	if n := len(l); n == 0 {
		l = []*HostInfo{host}
	} else {
		newL := make([]*HostInfo, n+1)
		for i := 0; i < n; i++ {
			if host.Equal(l[i]) {
				c.mu.Unlock()
				return false
			}
			newL[i] = l[i]
		}
		newL[n] = host
		l = newL
	}

	c.list.Store(&l)
	c.mu.Unlock()
	return true
}

func (c *cowHostList) update(host *HostInfo) {
	c.mu.Lock()
	l := c.get()

	if len(l) == 0 {
		c.mu.Unlock()
		return
	}

	found := false
	newL := make([]*HostInfo, len(l))
	for i := range l {
		if host.Equal(l[i]) {
			newL[i] = host
			found = true
		} else {
			newL[i] = l[i]
		}
	}

	if found {
		c.list.Store(&newL)
	}

	c.mu.Unlock()
}

func (c *cowHostList) remove(addr string) bool {
	c.mu.Lock()
	l := c.get()
	size := len(l)
	if size == 0 {
		c.mu.Unlock()
		return false
	}

	found := false
	newL := make([]*HostInfo, 0, size)
	for i := 0; i < len(l); i++ {
		if l[i].Peer() != addr {
			newL = append(newL, l[i])
		} else {
			found = true
		}
	}

	if !found {
		c.mu.Unlock()
		return false
	}

	newL = newL[:size-1 : size-1]
	c.list.Store(&newL)
	c.mu.Unlock()

	return true
}

// RetryableQuery is an interface that represents a query or batch statement that
// exposes the correct functions for the retry policy logic to evaluate correctly.
type RetryableQuery interface {
	Attempts() int
	GetConsistency() Consistency
}

// RetryPolicy interface is used by gocql to determine if a query can be attempted
// again after a retryable error has been received. The interface allows gocql
// users to implement their own logic to determine if a query can be attempted
// again.
//
// See SimpleRetryPolicy as an example of implementing and using a RetryPolicy
// interface.
type RetryPolicy interface {
	Attempt(RetryableQuery) bool
}

// SimpleRetryPolicy has simple logic for attempting a query a fixed number of times.
//
// See below for examples of usage:
//
//     //Assign to the cluster
//     cluster.RetryPolicy = &gocql.SimpleRetryPolicy{NumRetries: 3}
//
//     //Assign to a query
//     query.RetryPolicy(&gocql.SimpleRetryPolicy{NumRetries: 1})
//
type SimpleRetryPolicy struct {
	NumRetries int //Number of times to retry a query
}

// Attempt tells gocql to attempt the query again based on query.Attempts being less
// than the NumRetries defined in the policy.
func (s *SimpleRetryPolicy) Attempt(q RetryableQuery) bool {
	return q.Attempts() <= s.NumRetries
}

type HostStateNotifier interface {
	AddHost(host *HostInfo)
	RemoveHost(addr string)
	HostUp(host *HostInfo)
	HostDown(addr string)
}

// HostSelectionPolicy is an interface for selecting
// the most appropriate host to execute a given query.
type HostSelectionPolicy interface {
	HostStateNotifier
	SetPartitioner
	//Pick returns an iteration function over selected hosts
	Pick(ExecutableQuery) NextHost
}

// SelectedHost is an interface returned when picking a host from a host
// selection policy.
type SelectedHost interface {
	Info() *HostInfo
	Mark(error)
}

type selectedHost HostInfo

func (host *selectedHost) Info() *HostInfo {
	return (*HostInfo)(host)
}

func (host *selectedHost) Mark(err error) {}

// NextHost is an iteration function over picked hosts
type NextHost func() SelectedHost

// RoundRobinHostPolicy is a round-robin load balancing policy, where each host
// is tried sequentially for each query.
func RoundRobinHostPolicy() HostSelectionPolicy {
	return &roundRobinHostPolicy{}
}

type roundRobinHostPolicy struct {
	hosts cowHostList
	pos   uint32
	mu    sync.RWMutex
}

func (r *roundRobinHostPolicy) SetPartitioner(partitioner string) {
	// noop
}

func (r *roundRobinHostPolicy) Pick(qry ExecutableQuery) NextHost {
	// i is used to limit the number of attempts to find a host
	// to the number of hosts known to this policy
	var i int
	return func() SelectedHost {
		hosts := r.hosts.get()
		if len(hosts) == 0 {
			return nil
		}

		// always increment pos to evenly distribute traffic in case of
		// failures
		pos := atomic.AddUint32(&r.pos, 1) - 1
		if i >= len(hosts) {
			return nil
		}
		host := hosts[(pos)%uint32(len(hosts))]
		i++
		return (*selectedHost)(host)
	}
}

func (r *roundRobinHostPolicy) AddHost(host *HostInfo) {
	r.hosts.add(host)
}

func (r *roundRobinHostPolicy) RemoveHost(addr string) {
	r.hosts.remove(addr)
}

func (r *roundRobinHostPolicy) HostUp(host *HostInfo) {
	r.AddHost(host)
}

func (r *roundRobinHostPolicy) HostDown(addr string) {
	r.RemoveHost(addr)
}

// TokenAwareHostPolicy is a token aware host selection policy, where hosts are
// selected based on the partition key, so queries are sent to the host which
// owns the partition. Fallback is used when routing information is not available.
func TokenAwareHostPolicy(fallback HostSelectionPolicy) HostSelectionPolicy {
	return &tokenAwareHostPolicy{fallback: fallback}
}

type tokenAwareHostPolicy struct {
	hosts       cowHostList
	mu          sync.RWMutex
	partitioner string
	tokenRing   *tokenRing
	fallback    HostSelectionPolicy
}

func (t *tokenAwareHostPolicy) SetPartitioner(partitioner string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.partitioner != partitioner {
		t.fallback.SetPartitioner(partitioner)
		t.partitioner = partitioner

		t.resetTokenRing()
	}
}

func (t *tokenAwareHostPolicy) AddHost(host *HostInfo) {
	t.hosts.add(host)
	t.fallback.AddHost(host)

	t.mu.Lock()
	t.resetTokenRing()
	t.mu.Unlock()
}

func (t *tokenAwareHostPolicy) RemoveHost(addr string) {
	t.hosts.remove(addr)
	t.fallback.RemoveHost(addr)

	t.mu.Lock()
	t.resetTokenRing()
	t.mu.Unlock()
}

func (t *tokenAwareHostPolicy) HostUp(host *HostInfo) {
	t.AddHost(host)
}

func (t *tokenAwareHostPolicy) HostDown(addr string) {
	t.RemoveHost(addr)
}

func (t *tokenAwareHostPolicy) resetTokenRing() {
	if t.partitioner == "" {
		// partitioner not yet set
		return
	}

	// create a new token ring
	hosts := t.hosts.get()
	tokenRing, err := newTokenRing(t.partitioner, hosts)
	if err != nil {
		log.Printf("Unable to update the token ring due to error: %s", err)
		return
	}

	// replace the token ring
	t.tokenRing = tokenRing
}

func (t *tokenAwareHostPolicy) Pick(qry ExecutableQuery) NextHost {
	if qry == nil {
		return t.fallback.Pick(qry)
	}

	routingKey, err := qry.GetRoutingKey()
	if err != nil {
		return t.fallback.Pick(qry)
	}
	if routingKey == nil {
		return t.fallback.Pick(qry)
	}

	t.mu.RLock()
	// TODO retrieve a list of hosts based on the replication strategy
	host := t.tokenRing.GetHostForPartitionKey(routingKey)
	t.mu.RUnlock()

	if host == nil {
		return t.fallback.Pick(qry)
	}

	// scope these variables for the same lifetime as the iterator function
	var (
		hostReturned bool
		fallbackIter NextHost
	)

	return func() SelectedHost {
		if !hostReturned {
			hostReturned = true
			return (*selectedHost)(host)
		}

		// fallback
		if fallbackIter == nil {
			fallbackIter = t.fallback.Pick(qry)
		}

		fallbackHost := fallbackIter()

		// filter the token aware selected hosts from the fallback hosts
		if fallbackHost != nil && fallbackHost.Info() == host {
			fallbackHost = fallbackIter()
		}

		return fallbackHost
	}
}

// HostPoolHostPolicy is a host policy which uses the bitly/go-hostpool library
// to distribute queries between hosts and prevent sending queries to
// unresponsive hosts. When creating the host pool that is passed to the policy
// use an empty slice of hosts as the hostpool will be populated later by gocql.
// See below for examples of usage:
//
//     // Create host selection policy using a simple host pool
//     cluster.PoolConfig.HostSelectionPolicy = HostPoolHostPolicy(hostpool.New(nil))
//
//     // Create host selection policy using an epsilon greddy pool
//     cluster.PoolConfig.HostSelectionPolicy = HostPoolHostPolicy(
//         hostpool.NewEpsilonGreedy(nil, 0, &hostpool.LinearEpsilonValueCalculator{}),
//     )
//
func HostPoolHostPolicy(hp hostpool.HostPool) HostSelectionPolicy {
	return &hostPoolHostPolicy{hostMap: map[string]*HostInfo{}, hp: hp}
}

type hostPoolHostPolicy struct {
	hp      hostpool.HostPool
	mu      sync.RWMutex
	hostMap map[string]*HostInfo
}

func (r *hostPoolHostPolicy) SetHosts(hosts []*HostInfo) {
	peers := make([]string, len(hosts))
	hostMap := make(map[string]*HostInfo, len(hosts))

	for i, host := range hosts {
		peers[i] = host.Peer()
		hostMap[host.Peer()] = host
	}

	r.mu.Lock()
	r.hp.SetHosts(peers)
	r.hostMap = hostMap
	r.mu.Unlock()
}

func (r *hostPoolHostPolicy) AddHost(host *HostInfo) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.hostMap[host.Peer()]; ok {
		return
	}

	hosts := make([]string, 0, len(r.hostMap)+1)
	for addr := range r.hostMap {
		hosts = append(hosts, addr)
	}
	hosts = append(hosts, host.Peer())

	r.hp.SetHosts(hosts)
	r.hostMap[host.Peer()] = host
}

func (r *hostPoolHostPolicy) RemoveHost(addr string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.hostMap[addr]; !ok {
		return
	}

	delete(r.hostMap, addr)
	hosts := make([]string, 0, len(r.hostMap))
	for addr := range r.hostMap {
		hosts = append(hosts, addr)
	}

	r.hp.SetHosts(hosts)
}

func (r *hostPoolHostPolicy) HostUp(host *HostInfo) {
	r.AddHost(host)
}

func (r *hostPoolHostPolicy) HostDown(addr string) {
	r.RemoveHost(addr)
}

func (r *hostPoolHostPolicy) SetPartitioner(partitioner string) {
	// noop
}

func (r *hostPoolHostPolicy) Pick(qry ExecutableQuery) NextHost {
	return func() SelectedHost {
		r.mu.RLock()
		defer r.mu.RUnlock()

		if len(r.hostMap) == 0 {
			return nil
		}

		hostR := r.hp.Get()
		host, ok := r.hostMap[hostR.Host()]
		if !ok {
			return nil
		}

		return selectedHostPoolHost{
			policy: r,
			info:   host,
			hostR:  hostR,
		}
	}
}

// selectedHostPoolHost is a host returned by the hostPoolHostPolicy and
// implements the SelectedHost interface
type selectedHostPoolHost struct {
	policy *hostPoolHostPolicy
	info   *HostInfo
	hostR  hostpool.HostPoolResponse
}

func (host selectedHostPoolHost) Info() *HostInfo {
	return host.info
}

func (host selectedHostPoolHost) Mark(err error) {
	host.policy.mu.RLock()
	defer host.policy.mu.RUnlock()

	if _, ok := host.policy.hostMap[host.info.Peer()]; !ok {
		// host was removed between pick and mark
		return
	}

	host.hostR.Mark(err)
}
