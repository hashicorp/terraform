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
		"ingress.#":                    "1",
		"ingress.0.protocol":           "icmp",
		"ingress.0.from_port":          "1",
		"ingress.0.to_port":            "-1",
		"ingress.0.cidr_blocks.#":      "1",
		"ingress.0.cidr_blocks.0":      "0.0.0.0/0",
		"ingress.0.security_groups.#":  "2",
		"ingress.0.security_groups.0":  "sg-11111",
		"ingress.0.security_groups.1":  "foo/sg-22222",
	}
}

func Test_expandIPPerms(t *testing.T) {
	expanded := []interface{}{
		map[string]interface{}{
			"protocol":    "icmp",
			"from_port":   1,
			"to_port":     -1,
			"cidr_blocks": []interface{}{"0.0.0.0/0"},
			"security_groups": []interface{}{
				"sg-11111",
				"foo/sg-22222",
			},
		},
		map[string]interface{}{
			"protocol":  "icmp",
			"from_port": 1,
			"to_port":   -1,
			"self":      true,
		},
	}
	perms := expandIPPerms("foo", expanded)

	expected := []ec2.IPPerm{
		ec2.IPPerm{
			Protocol:  "icmp",
			FromPort:  1,
			ToPort:    -1,
			SourceIPs: []string{"0.0.0.0/0"},
			SourceGroups: []ec2.UserSecurityGroup{
				ec2.UserSecurityGroup{
					Id: "sg-11111",
				},
				ec2.UserSecurityGroup{
					OwnerId: "foo",
					Id:      "sg-22222",
				},
			},
		},
		ec2.IPPerm{
			Protocol: "icmp",
			FromPort: 1,
			ToPort:   -1,
			SourceGroups: []ec2.UserSecurityGroup{
				ec2.UserSecurityGroup{
					Id: "foo",
				},
			},
		},
	}

	if !reflect.DeepEqual(perms, expected) {
		t.Fatalf(
			"Got:\n\n%#v\n\nExpected:\n\n%#v\n",
			perms[0],
			expected)
	}

}

func Test_flattenIPPerms(t *testing.T) {
	cases := []struct {
		Input  []ec2.IPPerm
		Output []map[string]interface{}
	}{
		{
			Input: []ec2.IPPerm{
				ec2.IPPerm{
					Protocol:  "icmp",
					FromPort:  1,
					ToPort:    -1,
					SourceIPs: []string{"0.0.0.0/0"},
					SourceGroups: []ec2.UserSecurityGroup{
						ec2.UserSecurityGroup{
							Id: "sg-11111",
						},
					},
				},
			},

			Output: []map[string]interface{}{
				map[string]interface{}{
					"protocol":        "icmp",
					"from_port":       1,
					"to_port":         -1,
					"cidr_blocks":     []string{"0.0.0.0/0"},
					"security_groups": []string{"sg-11111"},
				},
			},
		},

		{
			Input: []ec2.IPPerm{
				ec2.IPPerm{
					Protocol:     "icmp",
					FromPort:     1,
					ToPort:       -1,
					SourceIPs:    []string{"0.0.0.0/0"},
					SourceGroups: nil,
				},
			},

			Output: []map[string]interface{}{
				map[string]interface{}{
					"protocol":    "icmp",
					"from_port":   1,
					"to_port":     -1,
					"cidr_blocks": []string{"0.0.0.0/0"},
				},
			},
		},
		{
			Input: []ec2.IPPerm{
				ec2.IPPerm{
					Protocol:  "icmp",
					FromPort:  1,
					ToPort:    -1,
					SourceIPs: nil,
				},
			},

			Output: []map[string]interface{}{
				map[string]interface{}{
					"protocol":  "icmp",
					"from_port": 1,
					"to_port":   -1,
				},
			},
		},
	}

	for _, tc := range cases {
		output := flattenIPPerms(tc.Input)
		if !reflect.DeepEqual(output, tc.Output) {
			t.Fatalf("Input:\n\n%#v\n\nOutput:\n\n%#v", tc.Input, output)
		}
	}
}

func Test_expandListeners(t *testing.T) {
	expanded := flatmap.Expand(testConf(), "listener").([]interface{})
	listeners, err := expandListeners(expanded)
	if err != nil {
		t.Fatalf("bad: %#v", err)
	}

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

func Test_flattenHealthCheck(t *testing.T) {
	cases := []struct {
		Input  elb.HealthCheck
		Output []map[string]interface{}
	}{
		{
			Input: elb.HealthCheck{
				UnhealthyThreshold: 10,
				HealthyThreshold:   10,
				Target:             "HTTP:80/",
				Timeout:            30,
				Interval:           30,
			},
			Output: []map[string]interface{}{
				map[string]interface{}{
					"unhealthy_threshold": 10,
					"healthy_threshold":   10,
					"target":              "HTTP:80/",
					"timeout":             30,
					"interval":            30,
				},
			},
		},
	}

	for _, tc := range cases {
		output := flattenHealthCheck(tc.Input)
		if !reflect.DeepEqual(output, tc.Output) {
			t.Fatalf("Got:\n\n%#v\n\nExpected:\n\n%#v", output, tc.Output)
		}
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
