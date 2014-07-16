package aws

import (
	"reflect"
	"testing"

	"github.com/hashicorp/terraform/flatmap"
	"github.com/mitchellh/goamz/ec2"
	"github.com/mitchellh/goamz/elb"
)

// Returns test configuration
func testConf() map[string]string {
	return map[string]string{
		"listener.#":                   "1",
		"listener.0.lb_port":           "80",
		"listener.0.lb_protocol":       "http",
		"listener.0.instance_port":     "8000",
		"listener.0.instance_protocol": "http",
		"availability_zones.#":         "2",
		"availability_zones.0":         "us-east-1a",
		"availability_zones.1":         "us-east-1b",
		"egress.#":                     "1",
		"egress.0.protocol":            "icmp",
		"egress.0.from_port":           "1",
		"egress.0.to_port":             "-1",
		"egress.0.cidr_blocks.#":       "1",
		"egress.0.cidr_blocks.0":       "0.0.0.0/0",
		"egress.0.security_groups.#":   "1",
		"egress.0.security_groups.0":   "sg-11111",
	}
}

func Test_expandIPPerms(t *testing.T) {
	expanded := flatmap.Expand(testConf(), "egress").([]interface{})
	perms := expandIPPerms(expanded)
	expected := ec2.IPPerm{
		Protocol:  "icmp",
		FromPort:  1,
		ToPort:    -1,
		SourceIPs: []string{"0.0.0.0/0"},
		SourceGroups: []ec2.UserSecurityGroup{
			ec2.UserSecurityGroup{
				Id: "sg-11111",
			},
		},
	}

	if !reflect.DeepEqual(perms[0], expected) {
		t.Fatalf(
			"Got:\n\n%#v\n\nExpected:\n\n%#v\n",
			perms[0],
			expected)
	}

}

func Test_expandListeners(t *testing.T) {
	expanded := flatmap.Expand(testConf(), "listener").([]interface{})
	listeners := expandListeners(expanded)
	expected := elb.Listener{
		InstancePort:     8000,
		LoadBalancerPort: 80,
		InstanceProtocol: "http",
		Protocol:         "http",
	}

	if !reflect.DeepEqual(listeners[0], expected) {
		t.Fatalf(
			"Got:\n\n%#v\n\nExpected:\n\n%#v\n",
			listeners[0],
			expected)
	}

}

func Test_expandStringList(t *testing.T) {
	expanded := flatmap.Expand(testConf(), "availability_zones").([]interface{})
	stringList := expandStringList(expanded)
	expected := []string{
		"us-east-1a",
		"us-east-1b",
	}

	if !reflect.DeepEqual(stringList, expected) {
		t.Fatalf(
			"Got:\n\n%#v\n\nExpected:\n\n%#v\n",
			stringList,
			expected)
	}

}

func Test_expandStringListWildcard(t *testing.T) {
	stringList := expandStringList([]interface{}{"us-east-1a,us-east-1b"})
	expected := []string{
		"us-east-1a",
		"us-east-1b",
	}

	if !reflect.DeepEqual(stringList, expected) {
		t.Fatalf(
			"Got:\n\n%#v\n\nExpected:\n\n%#v\n",
			stringList,
			expected)
	}

}
