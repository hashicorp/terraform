package aws

import (
	"reflect"
	"testing"

	"github.com/hashicorp/aws-sdk-go/aws"
	"github.com/hashicorp/aws-sdk-go/gen/ec2"
)

func Test_expandNetworkACLEntry(t *testing.T) {
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

	expected := []ec2.NetworkACLEntry{
		ec2.NetworkACLEntry{
			Protocol: aws.String("6"),
			PortRange: &ec2.PortRange{
				From: aws.Integer(22),
				To:   aws.Integer(22),
			},
			RuleAction: aws.String("deny"),
			RuleNumber: aws.Integer(1),
			CIDRBlock:  aws.String("0.0.0.0/0"),
			Egress:     aws.Boolean(true),
		},
		ec2.NetworkACLEntry{
			Protocol: aws.String("6"),
			PortRange: &ec2.PortRange{
				From: aws.Integer(443),
				To:   aws.Integer(443),
			},
			RuleAction: aws.String("deny"),
			RuleNumber: aws.Integer(2),
			CIDRBlock:  aws.String("0.0.0.0/0"),
			Egress:     aws.Boolean(true),
		},
	}

	if !reflect.DeepEqual(expanded, expected) {
		t.Fatalf(
			"Got:\n\n%#v\n\nExpected:\n\n%#v\n",
			expanded,
			expected)
	}

}

func Test_flattenNetworkACLEntry(t *testing.T) {

	apiInput := []ec2.NetworkACLEntry{
		ec2.NetworkACLEntry{
			Protocol: aws.String("tcp"),
			PortRange: &ec2.PortRange{
				From: aws.Integer(22),
				To:   aws.Integer(22),
			},
			RuleAction: aws.String("deny"),
			RuleNumber: aws.Integer(1),
			CIDRBlock:  aws.String("0.0.0.0/0"),
		},
		ec2.NetworkACLEntry{
			Protocol: aws.String("tcp"),
			PortRange: &ec2.PortRange{
				From: aws.Integer(443),
				To:   aws.Integer(443),
			},
			RuleAction: aws.String("deny"),
			RuleNumber: aws.Integer(2),
			CIDRBlock:  aws.String("0.0.0.0/0"),
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
