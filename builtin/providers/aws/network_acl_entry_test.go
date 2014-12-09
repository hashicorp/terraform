package aws

import (
	"reflect"
	"testing"

	"github.com/mitchellh/goamz/ec2"
)

func Test_expandNetworkAclEntry(t *testing.T) {
	input := []interface{}{
		map[string]interface{}{
			"protocol":   "tcp",
			"from_port":  22,
			"to_port":    22,
			"cidr_block": "0.0.0.0/0",
			"action":     "deny",
			"rule_no":    1,
		},
		map[string]interface{}{
			"protocol":   "tcp",
			"from_port":  443,
			"to_port":    443,
			"cidr_block": "0.0.0.0/0",
			"action":     "deny",
			"rule_no":    2,
		},
	}
	expanded, _ := expandNetworkAclEntries(input, "egress")

	expected := []ec2.NetworkAclEntry{
		ec2.NetworkAclEntry{
			Protocol: 6,
			PortRange: ec2.PortRange{
				From: 22,
				To:   22,
			},
			RuleAction: "deny",
			RuleNumber: 1,
			CidrBlock:  "0.0.0.0/0",
			Egress:     true,
			IcmpCode:   ec2.IcmpCode{Code: 0, Type: 0},
		},
		ec2.NetworkAclEntry{
			Protocol: 6,
			PortRange: ec2.PortRange{
				From: 443,
				To:   443,
			},
			RuleAction: "deny",
			RuleNumber: 2,
			CidrBlock:  "0.0.0.0/0",
			Egress:     true,
			IcmpCode:   ec2.IcmpCode{Code: 0, Type: 0},
		},
	}

	if !reflect.DeepEqual(expanded, expected) {
		t.Fatalf(
			"Got:\n\n%#v\n\nExpected:\n\n%#v\n",
			expanded,
			expected)
	}

}

func Test_flattenNetworkAclEntry(t *testing.T) {

	apiInput := []ec2.NetworkAclEntry{
		ec2.NetworkAclEntry{
			Protocol: 6,
			PortRange: ec2.PortRange{
				From: 22,
				To:   22,
			},
			RuleAction: "deny",
			RuleNumber: 1,
			CidrBlock:  "0.0.0.0/0",
		},
		ec2.NetworkAclEntry{
			Protocol: 6,
			PortRange: ec2.PortRange{
				From: 443,
				To:   443,
			},
			RuleAction: "deny",
			RuleNumber: 2,
			CidrBlock:  "0.0.0.0/0",
		},
	}
	flattened := flattenNetworkAclEntries(apiInput)

	expected := []map[string]interface{}{
		map[string]interface{}{
			"protocol":   "tcp",
			"from_port":  22,
			"to_port":    22,
			"cidr_block": "0.0.0.0/0",
			"action":     "deny",
			"rule_no":    1,
		},
		map[string]interface{}{
			"protocol":   "tcp",
			"from_port":  443,
			"to_port":    443,
			"cidr_block": "0.0.0.0/0",
			"action":     "deny",
			"rule_no":    2,
		},
	}

	if !reflect.DeepEqual(flattened, expected) {
		t.Fatalf(
			"Got:\n\n%#v\n\nExpected:\n\n%#v\n",
			flattened[0],
			expected)
	}

}
