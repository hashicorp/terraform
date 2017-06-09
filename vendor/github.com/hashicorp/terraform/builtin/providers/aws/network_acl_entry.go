package aws

import (
	"fmt"
	"net"
	"strconv"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
)

func expandNetworkAclEntries(configured []interface{}, entryType string) ([]*ec2.NetworkAclEntry, error) {
	entries := make([]*ec2.NetworkAclEntry, 0, len(configured))
	for _, eRaw := range configured {
		data := eRaw.(map[string]interface{})
		protocol := data["protocol"].(string)
		p, err := strconv.Atoi(protocol)
		if err != nil {
			var ok bool
			p, ok = protocolIntegers()[protocol]
			if !ok {
				return nil, fmt.Errorf("Invalid Protocol %s for rule %#v", protocol, data)
			}
		}

		e := &ec2.NetworkAclEntry{
			Protocol: aws.String(strconv.Itoa(p)),
			PortRange: &ec2.PortRange{
				From: aws.Int64(int64(data["from_port"].(int))),
				To:   aws.Int64(int64(data["to_port"].(int))),
			},
			Egress:     aws.Bool(entryType == "egress"),
			RuleAction: aws.String(data["action"].(string)),
			RuleNumber: aws.Int64(int64(data["rule_no"].(int))),
		}

		if v, ok := data["ipv6_cidr_block"]; ok {
			e.Ipv6CidrBlock = aws.String(v.(string))
		}

		if v, ok := data["cidr_block"]; ok {
			e.CidrBlock = aws.String(v.(string))
		}

		// Specify additional required fields for ICMP
		if p == 1 {
			e.IcmpTypeCode = &ec2.IcmpTypeCode{}
			if v, ok := data["icmp_code"]; ok {
				e.IcmpTypeCode.Code = aws.Int64(int64(v.(int)))
			}
			if v, ok := data["icmp_type"]; ok {
				e.IcmpTypeCode.Type = aws.Int64(int64(v.(int)))
			}
		}

		entries = append(entries, e)
	}
	return entries, nil
}

func flattenNetworkAclEntries(list []*ec2.NetworkAclEntry) []map[string]interface{} {
	entries := make([]map[string]interface{}, 0, len(list))

	for _, entry := range list {

		newEntry := map[string]interface{}{
			"from_port": *entry.PortRange.From,
			"to_port":   *entry.PortRange.To,
			"action":    *entry.RuleAction,
			"rule_no":   *entry.RuleNumber,
			"protocol":  *entry.Protocol,
		}

		if entry.CidrBlock != nil {
			newEntry["cidr_block"] = *entry.CidrBlock
		}

		if entry.Ipv6CidrBlock != nil {
			newEntry["ipv6_cidr_block"] = *entry.Ipv6CidrBlock
		}

		entries = append(entries, newEntry)
	}

	return entries

}

func protocolStrings(protocolIntegers map[string]int) map[int]string {
	protocolStrings := make(map[int]string, len(protocolIntegers))
	for k, v := range protocolIntegers {
		protocolStrings[v] = k
	}

	return protocolStrings
}

func protocolIntegers() map[string]int {
	var protocolIntegers = make(map[string]int)
	protocolIntegers = map[string]int{
		// defined at https://www.iana.org/assignments/protocol-numbers/protocol-numbers.xhtml
		"ah":   51,
		"esp":  50,
		"udp":  17,
		"tcp":  6,
		"icmp": 1,
		"all":  -1,
		"vrrp": 112,
	}
	return protocolIntegers
}

// expectedPortPair stores a pair of ports we expect to see together.
type expectedPortPair struct {
	to_port   int64
	from_port int64
}

// validatePorts ensures the ports and protocol match expected
// values.
func validatePorts(to int64, from int64, expected expectedPortPair) bool {
	if to != expected.to_port || from != expected.from_port {
		return false
	}

	return true
}

// validateCIDRBlock ensures the passed CIDR block represents an implied
// network, and not an overly-specified IP address.
func validateCIDRBlock(cidr string) error {
	_, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		return err
	}
	if ipnet.String() != cidr {
		return fmt.Errorf("%s is not a valid mask; did you mean %s?", cidr, ipnet)
	}

	return nil
}
