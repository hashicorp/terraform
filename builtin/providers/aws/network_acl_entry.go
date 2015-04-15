package aws

import (
	"fmt"
	"strconv"

	"github.com/awslabs/aws-sdk-go/aws"
	"github.com/awslabs/aws-sdk-go/service/ec2"
)

func expandNetworkAclEntries(configured []interface{}, entryType string) ([]*ec2.NetworkACLEntry, error) {
	entries := make([]*ec2.NetworkACLEntry, 0, len(configured))
	for _, eRaw := range configured {
		data := eRaw.(map[string]interface{})
		protocol := data["protocol"].(string)
		_, ok := protocolIntegers()[protocol]
		if !ok {
			return nil, fmt.Errorf("Invalid Protocol %s for rule %#v", protocol, data)
		}
		p := extractProtocolInteger(data["protocol"].(string))
		e := &ec2.NetworkACLEntry{
			Protocol: aws.String(strconv.Itoa(p)),
			PortRange: &ec2.PortRange{
				From: aws.Long(int64(data["from_port"].(int))),
				To:   aws.Long(int64(data["to_port"].(int))),
			},
			Egress:     aws.Boolean((entryType == "egress")),
			RuleAction: aws.String(data["action"].(string)),
			RuleNumber: aws.Long(int64(data["rule_no"].(int))),
			CIDRBlock:  aws.String(data["cidr_block"].(string)),
		}
		entries = append(entries, e)
	}
	return entries, nil
}

func flattenNetworkAclEntries(list []*ec2.NetworkACLEntry) []map[string]interface{} {
	entries := make([]map[string]interface{}, 0, len(list))

	for _, entry := range list {
		entries = append(entries, map[string]interface{}{
			"from_port":  *entry.PortRange.From,
			"to_port":    *entry.PortRange.To,
			"action":     *entry.RuleAction,
			"rule_no":    *entry.RuleNumber,
			"protocol":   *entry.Protocol,
			"cidr_block": *entry.CIDRBlock,
		})
	}

	return entries

}

func extractProtocolInteger(protocol string) int {
	return protocolIntegers()[protocol]
}

func extractProtocolString(protocol int) string {
	for key, value := range protocolIntegers() {
		if value == protocol {
			return key
		}
	}
	return ""
}

func protocolIntegers() map[string]int {
	var protocolIntegers = make(map[string]int)
	protocolIntegers = map[string]int{
		"udp":  17,
		"tcp":  6,
		"icmp": 1,
		"all":  -1,
	}
	return protocolIntegers
}
