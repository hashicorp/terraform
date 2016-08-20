package structs

import (
	"net"
	"reflect"
	"testing"
)

func TestNetworkIndex_Overcommitted(t *testing.T) {
	idx := NewNetworkIndex()

	// Consume some network
	reserved := &NetworkResource{
		Device:        "eth0",
		IP:            "192.168.0.100",
		MBits:         505,
		ReservedPorts: []Port{{"one", 8000}, {"two", 9000}},
	}
	collide := idx.AddReserved(reserved)
	if collide {
		t.Fatalf("bad")
	}
	if !idx.Overcommitted() {
		t.Fatalf("have no resources")
	}

	// Add resources
	n := &Node{
		Resources: &Resources{
			Networks: []*NetworkResource{
				&NetworkResource{
					Device: "eth0",
					CIDR:   "192.168.0.100/32",
					MBits:  1000,
				},
			},
		},
	}
	idx.SetNode(n)
	if idx.Overcommitted() {
		t.Fatalf("have resources")
	}

	// Double up our ussage
	idx.AddReserved(reserved)
	if !idx.Overcommitted() {
		t.Fatalf("should be overcommitted")
	}
}

func TestNetworkIndex_SetNode(t *testing.T) {
	idx := NewNetworkIndex()
	n := &Node{
		Resources: &Resources{
			Networks: []*NetworkResource{
				&NetworkResource{
					Device: "eth0",
					CIDR:   "192.168.0.100/32",
					MBits:  1000,
				},
			},
		},
		Reserved: &Resources{
			Networks: []*NetworkResource{
				&NetworkResource{
					Device:        "eth0",
					IP:            "192.168.0.100",
					ReservedPorts: []Port{{"ssh", 22}},
					MBits:         1,
				},
			},
		},
	}
	collide := idx.SetNode(n)
	if collide {
		t.Fatalf("bad")
	}

	if len(idx.AvailNetworks) != 1 {
		t.Fatalf("Bad")
	}
	if idx.AvailBandwidth["eth0"] != 1000 {
		t.Fatalf("Bad")
	}
	if idx.UsedBandwidth["eth0"] != 1 {
		t.Fatalf("Bad")
	}
	if !idx.UsedPorts["192.168.0.100"].Check(22) {
		t.Fatalf("Bad")
	}
}

func TestNetworkIndex_AddAllocs(t *testing.T) {
	idx := NewNetworkIndex()
	allocs := []*Allocation{
		&Allocation{
			TaskResources: map[string]*Resources{
				"web": &Resources{
					Networks: []*NetworkResource{
						&NetworkResource{
							Device:        "eth0",
							IP:            "192.168.0.100",
							MBits:         20,
							ReservedPorts: []Port{{"one", 8000}, {"two", 9000}},
						},
					},
				},
			},
		},
		&Allocation{
			TaskResources: map[string]*Resources{
				"api": &Resources{
					Networks: []*NetworkResource{
						&NetworkResource{
							Device:        "eth0",
							IP:            "192.168.0.100",
							MBits:         50,
							ReservedPorts: []Port{{"one", 10000}},
						},
					},
				},
			},
		},
	}
	collide := idx.AddAllocs(allocs)
	if collide {
		t.Fatalf("bad")
	}

	if idx.UsedBandwidth["eth0"] != 70 {
		t.Fatalf("Bad")
	}
	if !idx.UsedPorts["192.168.0.100"].Check(8000) {
		t.Fatalf("Bad")
	}
	if !idx.UsedPorts["192.168.0.100"].Check(9000) {
		t.Fatalf("Bad")
	}
	if !idx.UsedPorts["192.168.0.100"].Check(10000) {
		t.Fatalf("Bad")
	}
}

func TestNetworkIndex_AddReserved(t *testing.T) {
	idx := NewNetworkIndex()

	reserved := &NetworkResource{
		Device:        "eth0",
		IP:            "192.168.0.100",
		MBits:         20,
		ReservedPorts: []Port{{"one", 8000}, {"two", 9000}},
	}
	collide := idx.AddReserved(reserved)
	if collide {
		t.Fatalf("bad")
	}

	if idx.UsedBandwidth["eth0"] != 20 {
		t.Fatalf("Bad")
	}
	if !idx.UsedPorts["192.168.0.100"].Check(8000) {
		t.Fatalf("Bad")
	}
	if !idx.UsedPorts["192.168.0.100"].Check(9000) {
		t.Fatalf("Bad")
	}

	// Try to reserve the same network
	collide = idx.AddReserved(reserved)
	if !collide {
		t.Fatalf("bad")
	}
}

func TestNetworkIndex_yieldIP(t *testing.T) {
	idx := NewNetworkIndex()
	n := &Node{
		Resources: &Resources{
			Networks: []*NetworkResource{
				&NetworkResource{
					Device: "eth0",
					CIDR:   "192.168.0.100/30",
					MBits:  1000,
				},
			},
		},
		Reserved: &Resources{
			Networks: []*NetworkResource{
				&NetworkResource{
					Device:        "eth0",
					IP:            "192.168.0.100",
					ReservedPorts: []Port{{"ssh", 22}},
					MBits:         1,
				},
			},
		},
	}
	idx.SetNode(n)

	var out []string
	idx.yieldIP(func(n *NetworkResource, ip net.IP) (stop bool) {
		out = append(out, ip.String())
		return
	})

	expect := []string{"192.168.0.100", "192.168.0.101",
		"192.168.0.102", "192.168.0.103"}
	if !reflect.DeepEqual(out, expect) {
		t.Fatalf("bad: %v", out)
	}
}

func TestNetworkIndex_AssignNetwork(t *testing.T) {
	idx := NewNetworkIndex()
	n := &Node{
		Resources: &Resources{
			Networks: []*NetworkResource{
				&NetworkResource{
					Device: "eth0",
					CIDR:   "192.168.0.100/30",
					MBits:  1000,
				},
			},
		},
		Reserved: &Resources{
			Networks: []*NetworkResource{
				&NetworkResource{
					Device:        "eth0",
					IP:            "192.168.0.100",
					ReservedPorts: []Port{{"ssh", 22}},
					MBits:         1,
				},
			},
		},
	}
	idx.SetNode(n)

	allocs := []*Allocation{
		&Allocation{
			TaskResources: map[string]*Resources{
				"web": &Resources{
					Networks: []*NetworkResource{
						&NetworkResource{
							Device:        "eth0",
							IP:            "192.168.0.100",
							MBits:         20,
							ReservedPorts: []Port{{"one", 8000}, {"two", 9000}},
						},
					},
				},
			},
		},
		&Allocation{
			TaskResources: map[string]*Resources{
				"api": &Resources{
					Networks: []*NetworkResource{
						&NetworkResource{
							Device:        "eth0",
							IP:            "192.168.0.100",
							MBits:         50,
							ReservedPorts: []Port{{"main", 10000}},
						},
					},
				},
			},
		},
	}
	idx.AddAllocs(allocs)

	// Ask for a reserved port
	ask := &NetworkResource{
		ReservedPorts: []Port{{"main", 8000}},
	}
	offer, err := idx.AssignNetwork(ask)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if offer == nil {
		t.Fatalf("bad")
	}
	if offer.IP != "192.168.0.101" {
		t.Fatalf("bad: %#v", offer)
	}
	rp := Port{"main", 8000}
	if len(offer.ReservedPorts) != 1 || offer.ReservedPorts[0] != rp {
		t.Fatalf("bad: %#v", offer)
	}

	// Ask for dynamic ports
	ask = &NetworkResource{
		DynamicPorts: []Port{{"http", 0}, {"https", 0}, {"admin", 0}},
	}
	offer, err = idx.AssignNetwork(ask)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if offer == nil {
		t.Fatalf("bad")
	}
	if offer.IP != "192.168.0.100" {
		t.Fatalf("bad: %#v", offer)
	}
	if len(offer.DynamicPorts) != 3 {
		t.Fatalf("There should be three dynamic ports")
	}
	for _, port := range offer.DynamicPorts {
		if port.Value == 0 {
			t.Fatalf("Dynamic Port: %v should have been assigned a host port", port.Label)
		}
	}

	// Ask for reserved + dynamic ports
	ask = &NetworkResource{
		ReservedPorts: []Port{{"main", 2345}},
		DynamicPorts:  []Port{{"http", 0}, {"https", 0}, {"admin", 0}},
	}
	offer, err = idx.AssignNetwork(ask)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if offer == nil {
		t.Fatalf("bad")
	}
	if offer.IP != "192.168.0.100" {
		t.Fatalf("bad: %#v", offer)
	}

	rp = Port{"main", 2345}
	if len(offer.ReservedPorts) != 1 || offer.ReservedPorts[0] != rp {
		t.Fatalf("bad: %#v", offer)
	}

	// Ask for too much bandwidth
	ask = &NetworkResource{
		MBits: 1000,
	}
	offer, err = idx.AssignNetwork(ask)
	if err.Error() != "bandwidth exceeded" {
		t.Fatalf("err: %v", err)
	}
	if offer != nil {
		t.Fatalf("bad")
	}
}

// This test ensures that even with a small domain of available ports we are
// able to make a dynamic port allocation.
func TestNetworkIndex_AssignNetwork_Dynamic_Contention(t *testing.T) {

	// Create a node that only has one free port
	idx := NewNetworkIndex()
	n := &Node{
		Resources: &Resources{
			Networks: []*NetworkResource{
				&NetworkResource{
					Device: "eth0",
					CIDR:   "192.168.0.100/32",
					MBits:  1000,
				},
			},
		},
		Reserved: &Resources{
			Networks: []*NetworkResource{
				&NetworkResource{
					Device: "eth0",
					IP:     "192.168.0.100",
					MBits:  1,
				},
			},
		},
	}
	for i := MinDynamicPort; i < MaxDynamicPort-1; i++ {
		n.Reserved.Networks[0].ReservedPorts = append(n.Reserved.Networks[0].ReservedPorts, Port{Value: i})
	}

	idx.SetNode(n)

	// Ask for dynamic ports
	ask := &NetworkResource{
		DynamicPorts: []Port{{"http", 0}},
	}
	offer, err := idx.AssignNetwork(ask)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if offer == nil {
		t.Fatalf("bad")
	}
	if offer.IP != "192.168.0.100" {
		t.Fatalf("bad: %#v", offer)
	}
	if len(offer.DynamicPorts) != 1 {
		t.Fatalf("There should be three dynamic ports")
	}
	if p := offer.DynamicPorts[0].Value; p != MaxDynamicPort {
		t.Fatalf("Dynamic Port: should have been assigned %d; got %d", p, MaxDynamicPort)
	}
}

func TestIntContains(t *testing.T) {
	l := []int{1, 2, 10, 20}
	if isPortReserved(l, 50) {
		t.Fatalf("bad")
	}
	if !isPortReserved(l, 20) {
		t.Fatalf("bad")
	}
	if !isPortReserved(l, 1) {
		t.Fatalf("bad")
	}
}
