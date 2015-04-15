package aws

import (
	"reflect"
	"testing"

	"github.com/awslabs/aws-sdk-go/aws"
	"github.com/awslabs/aws-sdk-go/service/ec2"
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

	expected := []*ec2.NetworkACLEntry{
		&ec2.NetworkACLEntry{
			Protocol: aws.String("6"),
			PortRange: &ec2.PortRange{
				From: aws.Long(22),
				To:   aws.Long(22),
			},
			RuleAction: aws.String("deny"),
			RuleNumber: aws.Long(1),
			CIDRBlock:  aws.String("0.0.0.0/0"),
			Egress:     aws.Boolean(true),
		},
		&ec2.NetworkACLEntry{
			Protocol: aws.String("6"),
			PortRange: &ec2.PortRange{
				From: aws.Long(443),
				To:   aws.Long(443),
			},
			RuleAction: aws.String("deny"),
			RuleNumber: aws.Long(2),
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

	apiInput := []*ec2.NetworkACLEntry{
		&ec2.NetworkACLEntry{
			Protocol: aws.String("tcp"),
			PortRange: &ec2.PortRange{
				From: aws.Long(22),
				To:   aws.Long(22),
			},
			RuleAction: aws.String("deny"),
			RuleNumber: aws.Long(1),
			CIDRBlock:  aws.String("0.0.0.0/0"),
		},
		&ec2.NetworkACLEntry{
			Protocol: aws.String("tcp"),
			PortRange: &ec2.PortRange{
				From: aws.Long(443),
				To:   aws.Long(443),
			},
			RuleAction: aws.String("deny"),
			RuleNumber: aws.Long(2),
			CIDRBlock:  aws.String("0.0.0.0/0"),
		},
	}
	flattened := flattenNetworkAclEntries(apiInput)

	expected := []map[string]interface{}{
		map[string]interface{}{
			"protocol":   "tcp",
			"from_port":  int64(22),
			"to_port":    int64(22),
			"cidr_block": "0.0.0.0/0",
			"action":     "deny",
			"rule_no":    int64(1),
		},
		map[string]interface{}{
			"protocol":   "tcp",
			"from_port":  int64(443),
			"to_port":    int64(443),
			"cidr_block": "0.0.0.0/0",
			"action":     "deny",
			"rule_no":    int64(2),
		},
	}

	if !reflect.DeepEqual(flattened, expected) {
		t.Fatalf(
			"Got:\n\n%#v\n\nExpected:\n\n%#v\n",
			flattened,
			expected)
	}

}
