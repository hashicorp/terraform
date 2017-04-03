package opc

import "testing"

func TestValidateIPPrefixCIDR(t *testing.T) {
	validPrefixes := []string{
		"10.0.1.0/24",
		"10.1.0.0/16",
		"192.168.0.1/32",
		"10.20.0.0/18",
		"10.0.12.0/24",
	}

	for _, v := range validPrefixes {
		_, errors := validateIPPrefixCIDR(v, "prefix")
		if len(errors) != 0 {
			t.Fatalf("%q should be a valid IP Address Prefix: %q", v, errors)
		}
	}

	invalidPrefixes := []string{
		"10.0.0.1/35",
		"192.168.1.256/16",
		"256.0.1/16",
	}

	for _, v := range invalidPrefixes {
		_, errors := validateIPPrefixCIDR(v, "prefix")
		if len(errors) == 0 {
			t.Fatalf("%q should not be a valid IP Address", v)
		}
	}
}

func TestValidateAdminDistance(t *testing.T) {
	validDistances := []int{
		0,
		1,
		2,
	}

	for _, v := range validDistances {
		_, errors := validateAdminDistance(v, "distance")
		if len(errors) != 0 {
			t.Fatalf("%q should be a valid Admin Distance: %q", v, errors)
		}
	}

	invalidDistances := []int{
		-1,
		4,
		3,
		42,
	}

	for _, v := range invalidDistances {
		_, errors := validateAdminDistance(v, "distance")
		if len(errors) == 0 {
			t.Fatalf("%q should not be a valid Admin Distance", v)
		}
	}
}

func TestValidateIPProtocol(t *testing.T) {
	validProtocols := []string{
		"all",
		"ah",
		"esp",
		"icmp",
		"icmpv6",
		"igmp",
		"ipip",
		"gre",
		"mplsip",
		"ospf",
		"pim",
		"rdp",
		"sctp",
		"tcp",
		"udp",
	}

	for _, v := range validProtocols {
		_, errors := validateIPProtocol(v, "ip_protocol")
		if len(errors) != 0 {
			t.Fatalf("%q should be a valid Admin Distance: %q", v, errors)
		}
	}

	invalidProtocols := []string{
		"bad",
		"real bad",
		"are you even trying at this point?",
	}
	for _, v := range invalidProtocols {
		_, errors := validateIPProtocol(v, "ip_protocol")
		if len(errors) == 0 {
			t.Fatalf("%q should not be a valid IP Protocol", v)
		}
	}

}
