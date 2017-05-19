package opc

import (
	"fmt"
	"net"

	"github.com/hashicorp/go-oracle-terraform/compute"
)

// Validate whether an IP Prefix CIDR is correct or not
func validateIPPrefixCIDR(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)

	_, ipnet, err := net.ParseCIDR(value)
	if err != nil {
		errors = append(errors, fmt.Errorf(
			"%q must contain a valid CIDR, got error while parsing: %s", k, err))
		return
	}

	if ipnet == nil || value != ipnet.String() {
		errors = append(errors, fmt.Errorf(
			"%q must contain a valid network CIDR, expected %q, got %q", k, ipnet, value))
		return
	}
	return
}

// Admin distance can either be a 0, 1, or a 2. Defaults to 0.
func validateAdminDistance(v interface{}, k string) (ws []string, errors []error) {
	value := v.(int)

	if value < 0 || value > 2 {
		errors = append(errors, fmt.Errorf(
			"%q can only be an interger between 0-2. Got: %d", k, value))
	}
	return
}

// Admin distance can either be a 0, 1, or a 2. Defaults to 0.
func validateIPProtocol(v interface{}, k string) (ws []string, errors []error) {
	validProtocols := map[string]struct{}{
		string(compute.All):    {},
		string(compute.AH):     {},
		string(compute.ESP):    {},
		string(compute.ICMP):   {},
		string(compute.ICMPV6): {},
		string(compute.IGMP):   {},
		string(compute.IPIP):   {},
		string(compute.GRE):    {},
		string(compute.MPLSIP): {},
		string(compute.OSPF):   {},
		string(compute.PIM):    {},
		string(compute.RDP):    {},
		string(compute.SCTP):   {},
		string(compute.TCP):    {},
		string(compute.UDP):    {},
	}

	value := v.(string)
	if _, ok := validProtocols[value]; !ok {
		errors = append(errors, fmt.Errorf(
			`%q must contain a valid Image owner , expected ["all",	"ah",	"esp", "icmp",	"icmpv6",	"igmp",	"ipip",	"gre",	"mplsip",	"ospf",	"pim",	"rdp",	"sctp",	"tcp",	"udp"] got %q`,
			k, value))
	}
	return
}
