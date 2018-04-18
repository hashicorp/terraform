package cidr

import (
	"fmt"
	"net"
	"testing"

	r "github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestCidrNetwork(t *testing.T) {
	r.UnitTest(t, r.TestCase{
		Providers: testProviders,
		Steps: []r.TestStep{
			r.TestStep{
				Config: networks,
				Check: r.ComposeTestCheckFunc(
					r.TestCheckResourceAttr("data.cidr_network.cake", "subnet_cidrs.private_az1", "10.0.0.0/24"),
					r.TestCheckResourceAttr("data.cidr_network.cake", "subnet_cidrs.private_az2", "10.0.1.0/24"),
					r.TestCheckResourceAttr("data.cidr_network.cake", "subnet_cidrs.private_az3", "10.0.2.0/24"),
					r.TestCheckResourceAttr("data.cidr_network.cake", "subnet_cidrs.public_az1", "10.0.3.0/25"),
					r.TestCheckResourceAttr("data.cidr_network.cake", "subnet_cidrs.public_az2", "10.0.3.128/25"),
					r.TestCheckResourceAttr("data.cidr_network.cake", "subnet_cidrs.public_az3", "10.0.4.0/25"),
					r.TestCheckResourceAttr("data.cidr_network.cake", "subnet_cidrs.elb_az1", "10.0.4.128/28"),
					r.TestCheckResourceAttr("data.cidr_network.cake", "subnet_cidrs.elb_az2", "10.0.4.144/28"),
					r.TestCheckResourceAttr("data.cidr_network.cake", "subnet_cidrs.elb_az3", "10.0.4.160/28"),
					r.TestCheckResourceAttr("data.cidr_network.order", "subnet_cidrs.private_az1", "10.0.0.0/28"),
					r.TestCheckResourceAttr("data.cidr_network.order", "subnet_cidrs.private_az2", "10.0.1.0/24"),
					r.TestCheckResourceAttr("data.cidr_network.order", "subnet_cidrs.elb_az1", "10.0.2.0/28"),
					r.TestCheckResourceAttr("data.cidr_network.order", "subnet_cidrs.elb_az2", "10.0.2.32/27"),
				),
			},
		},
	},
	)
}

type testCase struct {
	CurrentSubnet string
	Zones         int
	SubnetMasks   []*subnetMask
	CIDRByZone    []map[string]string
	CIDRByName    map[string]string
	CIDRList      []string
}

func TestCalculateSubnets(t *testing.T) {
	failCases := []*testCase{
		&testCase{
			CurrentSubnet: "255.255.255.0/24",
			Zones:         1,
			SubnetMasks: []*subnetMask{
				&subnetMask{Name: "private", Mask: 27},
				&subnetMask{Name: "public", Mask: 28},
				&subnetMask{Name: "nat", Mask: 29},
				&subnetMask{Name: "4_special", Mask: 22},
			},
		},
		&testCase{
			CurrentSubnet: "255.255.252.0/24",
			Zones:         1,
			SubnetMasks: []*subnetMask{
				&subnetMask{Name: "private", Mask: 27},
				&subnetMask{Name: "public", Mask: 24},
				&subnetMask{Name: "nat", Mask: 29},
				&subnetMask{Name: "4_special", Mask: 22},
			},
		},
	}
	testCases := []*testCase{
		&testCase{
			CurrentSubnet: "9.255.255.0/24",
			Zones:         1,
			SubnetMasks: []*subnetMask{
				&subnetMask{Name: "private", Mask: 24},
				&subnetMask{Name: "public", Mask: 25},
				&subnetMask{Name: "nat", Mask: 26},
				&subnetMask{Name: "4_special", Mask: 27},
			},
			CIDRList: []string{
				"10.0.0.0/24",
				"10.0.1.0/25",
				"10.0.1.128/26",
				"10.0.1.192/27",
			},
			CIDRByName: map[string]string{
				"private":   "10.0.0.0/24",
				"public":    "10.0.1.0/25",
				"nat":       "10.0.1.128/26",
				"4_special": "10.0.1.192/27",
			},
		},
		&testCase{
			CurrentSubnet: "192.168.7.0/24",
			Zones:         1,
			SubnetMasks: []*subnetMask{
				&subnetMask{Name: "private", Mask: 24},
				&subnetMask{Name: "public", Mask: 25},
				&subnetMask{Name: "nat", Mask: 26},
				&subnetMask{Name: "4_special", Mask: 27},
			},
			CIDRList: []string{
				"192.168.8.0/24",
				"192.168.9.0/25",
				"192.168.9.128/26",
				"192.168.9.192/27",
			},
			CIDRByName: map[string]string{
				"private":   "192.168.8.0/24",
				"public":    "192.168.9.0/25",
				"nat":       "192.168.9.128/26",
				"4_special": "192.168.9.192/27",
			},
		},
		&testCase{
			CurrentSubnet: "2001:db8:c001:b9c0::/58",
			Zones:         1,
			SubnetMasks: []*subnetMask{
				&subnetMask{Name: "private", Mask: 58},
				&subnetMask{Name: "public", Mask: 58},
				&subnetMask{Name: "nat", Mask: 58},
				&subnetMask{Name: "4_special", Mask: 58},
			},
			CIDRList: []string{
				"2001:db8:c001:ba00::/58",
				"2001:db8:c001:ba40::/58",
				"2001:db8:c001:ba80::/58",
				"2001:db8:c001:bac0::/58",
			},
			CIDRByName: map[string]string{
				"private":   "2001:db8:c001:ba00::/58",
				"public":    "2001:db8:c001:ba40::/58",
				"nat":       "2001:db8:c001:ba80::/58",
				"4_special": "2001:db8:c001:bac0::/58",
			},
		},
	}

	for _, tc := range failCases {
		_, IPNet, err := net.ParseCIDR(tc.CurrentSubnet)
		if err != nil {
			t.Errorf("Failed to parse %s\n", tc.CurrentSubnet)
		}
		_, _, fail := calculateSubnets(IPNet, tc.SubnetMasks)
		if fail == nil {
			t.Errorf("calculateSubnets should have failed starting at %s\n", tc.CurrentSubnet)
		}
	}
	for _, tc := range testCases {
		_, IPNet, err := net.ParseCIDR(tc.CurrentSubnet)
		if err != nil {
			t.Errorf("Failed to parse %s\n", tc.CurrentSubnet)
		}
		cByName, _, err := calculateSubnets(IPNet, tc.SubnetMasks)
		if err != nil {
			t.Errorf("calculateSubnets with failed %v\n", err)
		}
		for name, got := range cByName {
			if got != tc.CIDRByName[name] {
				t.Errorf("Got %v, expected %v\n", got, tc.CIDRByName[name])
			}
		}
	}
}

func outputCheckNetwork(s *terraform.State) error {
	answers := [][]string{
		[]string{"private_subnet1", "192.168.0.0/24"},
		[]string{"private_subnet2", "192.168.1.0/24"},
		[]string{"private_subnet3", "192.168.2.0/24"},
		[]string{"public_subnet1", "192.168.3.0/25"},
		[]string{"public_subnet2", "192.168.3.128/25"},
		[]string{"public_subnet3", "192.168.4.0/25"},
		[]string{"elb_subnet1", "192.168.4.128/28"},
		[]string{"elb_subnet2", "192.168.4.144/28"},
		[]string{"elb_subnet3", "192.168.4.160/28"},
	}
	for _, ans := range answers {
		got := s.RootModule().Outputs[ans[0]]
		if ans[1] != got.Value {
			fmt.Printf("Outputs %v\n", s.RootModule().Outputs)
			return fmt.Errorf("Output expected %s, got %s\n", ans[1], got.Value)
		}
	}
	return nil
}

const networks = `
data "cidr_network" "cake" {
	cidr_block = "10.0.0.0/21"
	subnet {
		mask = 24
		name = "private_az1" 
	}
	subnet {
		mask = 24
		name = "private_az2" 
	}
	subnet {
		mask = 24
		name = "private_az3" 
	}
	subnet {
		mask = 25
		name = "public_az1"
	} 
	subnet {
		mask = 25
		name = "public_az2"
	}
	subnet {
		mask = 25
		name = "public_az3"
	} 
	subnet {
		mask = 28
		name = "elb_az1" 
	}
	subnet {
		mask = 28
		name = "elb_az2" 
	}
	subnet {
		mask = 28
		name = "elb_az3" 
	}
}
data "cidr_network" "order" {
	cidr_block = "10.0.0.0/21"
	subnet {
		mask = 28
		name = "private_az1" 
	}
	subnet {
		mask = 24
		name = "private_az2" 
	}
	subnet {
		mask = 28
		name = "elb_az1" 
	}
	subnet {
		mask = 27
		name = "elb_az2" 
	}
}
`
