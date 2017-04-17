package structs

import (
	"fmt"
	"math/rand"
	"net"
	"sync"
)

const (
	// MinDynamicPort is the smallest dynamic port generated
	MinDynamicPort = 20000

	// MaxDynamicPort is the largest dynamic port generated
	MaxDynamicPort = 60000

	// maxRandPortAttempts is the maximum number of attempt
	// to assign a random port
	maxRandPortAttempts = 20

	// maxValidPort is the max valid port number
	maxValidPort = 65536
)

var (
	// bitmapPool is used to pool the bitmaps used for port collision
	// checking. They are fairly large (8K) so we can re-use them to
	// avoid GC pressure. Care should be taken to call Clear() on any
	// bitmap coming from the pool.
	bitmapPool = new(sync.Pool)
)

// NetworkIndex is used to index the available network resources
// and the used network resources on a machine given allocations
type NetworkIndex struct {
	AvailNetworks  []*NetworkResource // List of available networks
	AvailBandwidth map[string]int     // Bandwidth by device
	UsedPorts      map[string]Bitmap  // Ports by IP
	UsedBandwidth  map[string]int     // Bandwidth by device
}

// NewNetworkIndex is used to construct a new network index
func NewNetworkIndex() *NetworkIndex {
	return &NetworkIndex{
		AvailBandwidth: make(map[string]int),
		UsedPorts:      make(map[string]Bitmap),
		UsedBandwidth:  make(map[string]int),
	}
}

// Release is called when the network index is no longer needed
// to attempt to re-use some of the memory it has allocated
func (idx *NetworkIndex) Release() {
	for _, b := range idx.UsedPorts {
		bitmapPool.Put(b)
	}
}

// Overcommitted checks if the network is overcommitted
func (idx *NetworkIndex) Overcommitted() bool {
	for device, used := range idx.UsedBandwidth {
		avail := idx.AvailBandwidth[device]
		if used > avail {
			return true
		}
	}
	return false
}

// SetNode is used to setup the available network resources. Returns
// true if there is a collision
func (idx *NetworkIndex) SetNode(node *Node) (collide bool) {
	// Add the available CIDR blocks
	for _, n := range node.Resources.Networks {
		if n.Device != "" {
			idx.AvailNetworks = append(idx.AvailNetworks, n)
			idx.AvailBandwidth[n.Device] = n.MBits
		}
	}

	// Add the reserved resources
	if r := node.Reserved; r != nil {
		for _, n := range r.Networks {
			if idx.AddReserved(n) {
				collide = true
			}
		}
	}
	return
}

// AddAllocs is used to add the used network resources. Returns
// true if there is a collision
func (idx *NetworkIndex) AddAllocs(allocs []*Allocation) (collide bool) {
	for _, alloc := range allocs {
		for _, task := range alloc.TaskResources {
			if len(task.Networks) == 0 {
				continue
			}
			n := task.Networks[0]
			if idx.AddReserved(n) {
				collide = true
			}
		}
	}
	return
}

// AddReserved is used to add a reserved network usage, returns true
// if there is a port collision
func (idx *NetworkIndex) AddReserved(n *NetworkResource) (collide bool) {
	// Add the port usage
	used := idx.UsedPorts[n.IP]
	if used == nil {
		// Try to get a bitmap from the pool, else create
		raw := bitmapPool.Get()
		if raw != nil {
			used = raw.(Bitmap)
			used.Clear()
		} else {
			used, _ = NewBitmap(maxValidPort)
		}
		idx.UsedPorts[n.IP] = used
	}

	for _, ports := range [][]Port{n.ReservedPorts, n.DynamicPorts} {
		for _, port := range ports {
			// Guard against invalid port
			if port.Value < 0 || port.Value >= maxValidPort {
				return true
			}
			if used.Check(uint(port.Value)) {
				collide = true
			} else {
				used.Set(uint(port.Value))
			}
		}
	}

	// Add the bandwidth
	idx.UsedBandwidth[n.Device] += n.MBits
	return
}

// yieldIP is used to iteratively invoke the callback with
// an available IP
func (idx *NetworkIndex) yieldIP(cb func(net *NetworkResource, ip net.IP) bool) {
	inc := func(ip net.IP) {
		for j := len(ip) - 1; j >= 0; j-- {
			ip[j]++
			if ip[j] > 0 {
				break
			}
		}
	}

	for _, n := range idx.AvailNetworks {
		ip, ipnet, err := net.ParseCIDR(n.CIDR)
		if err != nil {
			continue
		}
		for ip := ip.Mask(ipnet.Mask); ipnet.Contains(ip); inc(ip) {
			if cb(n, ip) {
				return
			}
		}
	}
}

// AssignNetwork is used to assign network resources given an ask.
// If the ask cannot be satisfied, returns nil
func (idx *NetworkIndex) AssignNetwork(ask *NetworkResource) (out *NetworkResource, err error) {
	err = fmt.Errorf("no networks available")
	idx.yieldIP(func(n *NetworkResource, ip net.IP) (stop bool) {
		// Convert the IP to a string
		ipStr := ip.String()

		// Check if we would exceed the bandwidth cap
		availBandwidth := idx.AvailBandwidth[n.Device]
		usedBandwidth := idx.UsedBandwidth[n.Device]
		if usedBandwidth+ask.MBits > availBandwidth {
			err = fmt.Errorf("bandwidth exceeded")
			return
		}

		used := idx.UsedPorts[ipStr]

		// Check if any of the reserved ports are in use
		for _, port := range ask.ReservedPorts {
			// Guard against invalid port
			if port.Value < 0 || port.Value >= maxValidPort {
				err = fmt.Errorf("invalid port %d (out of range)", port.Value)
				return
			}

			// Check if in use
			if used != nil && used.Check(uint(port.Value)) {
				err = fmt.Errorf("reserved port collision")
				return
			}
		}

		// Create the offer
		offer := &NetworkResource{
			Device:        n.Device,
			IP:            ipStr,
			MBits:         ask.MBits,
			ReservedPorts: ask.ReservedPorts,
			DynamicPorts:  ask.DynamicPorts,
		}

		// Try to stochastically pick the dynamic ports as it is faster and
		// lower memory usage.
		var dynPorts []int
		var dynErr error
		dynPorts, dynErr = getDynamicPortsStochastic(used, ask)
		if dynErr == nil {
			goto BUILD_OFFER
		}

		// Fall back to the precise method if the random sampling failed.
		dynPorts, dynErr = getDynamicPortsPrecise(used, ask)
		if dynErr != nil {
			err = dynErr
			return
		}

	BUILD_OFFER:
		for i, port := range dynPorts {
			offer.DynamicPorts[i].Value = port
		}

		// Stop, we have an offer!
		out = offer
		err = nil
		return true
	})
	return
}

// getDynamicPortsPrecise takes the nodes used port bitmap which may be nil if
// no ports have been allocated yet, the network ask and returns a set of unused
// ports to fullfil the ask's DynamicPorts or an error if it failed. An error
// means the ask can not be satisfied as the method does a precise search.
func getDynamicPortsPrecise(nodeUsed Bitmap, ask *NetworkResource) ([]int, error) {
	// Create a copy of the used ports and apply the new reserves
	var usedSet Bitmap
	var err error
	if nodeUsed != nil {
		usedSet, err = nodeUsed.Copy()
		if err != nil {
			return nil, err
		}
	} else {
		usedSet, err = NewBitmap(maxValidPort)
		if err != nil {
			return nil, err
		}
	}

	for _, port := range ask.ReservedPorts {
		usedSet.Set(uint(port.Value))
	}

	// Get the indexes of the unset
	availablePorts := usedSet.IndexesInRange(false, MinDynamicPort, MaxDynamicPort)

	// Randomize the amount we need
	numDyn := len(ask.DynamicPorts)
	if len(availablePorts) < numDyn {
		return nil, fmt.Errorf("dynamic port selection failed")
	}

	numAvailable := len(availablePorts)
	for i := 0; i < numDyn; i++ {
		j := rand.Intn(numAvailable)
		availablePorts[i], availablePorts[j] = availablePorts[j], availablePorts[i]
	}

	return availablePorts[:numDyn], nil
}

// getDynamicPortsStochastic takes the nodes used port bitmap which may be nil if
// no ports have been allocated yet, the network ask and returns a set of unused
// ports to fullfil the ask's DynamicPorts or an error if it failed. An error
// does not mean the ask can not be satisfied as the method has a fixed amount
// of random probes and if these fail, the search is aborted.
func getDynamicPortsStochastic(nodeUsed Bitmap, ask *NetworkResource) ([]int, error) {
	var reserved, dynamic []int
	for _, port := range ask.ReservedPorts {
		reserved = append(reserved, port.Value)
	}

	for i := 0; i < len(ask.DynamicPorts); i++ {
		attempts := 0
	PICK:
		attempts++
		if attempts > maxRandPortAttempts {
			return nil, fmt.Errorf("stochastic dynamic port selection failed")
		}

		randPort := MinDynamicPort + rand.Intn(MaxDynamicPort-MinDynamicPort)
		if nodeUsed != nil && nodeUsed.Check(uint(randPort)) {
			goto PICK
		}

		for _, ports := range [][]int{reserved, dynamic} {
			if isPortReserved(ports, randPort) {
				goto PICK
			}
		}
		dynamic = append(dynamic, randPort)
	}

	return dynamic, nil
}

// IntContains scans an integer slice for a value
func isPortReserved(haystack []int, needle int) bool {
	for _, item := range haystack {
		if item == needle {
			return true
		}
	}
	return false
}
