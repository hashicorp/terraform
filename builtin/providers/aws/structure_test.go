package aws

import (
	"reflect"
	"testing"

	"github.com/hashicorp/aws-sdk-go/aws"
	ec2 "github.com/hashicorp/aws-sdk-go/gen/ec2"
	"github.com/hashicorp/aws-sdk-go/gen/elb"
	"github.com/hashicorp/aws-sdk-go/gen/rds"
	"github.com/hashicorp/terraform/flatmap"
	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/schema"
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

func TestExpandIPPerms(t *testing.T) {
	hash := func(v interface{}) int {
		return hashcode.String(v.(string))
	}

	expanded := []interface{}{
		map[string]interface{}{
			"protocol":    "icmp",
			"from_port":   1,
			"to_port":     -1,
			"cidr_blocks": []interface{}{"0.0.0.0/0"},
			"security_groups": schema.NewSet(hash, []interface{}{
				"sg-11111",
				"foo/sg-22222",
			}),
		},
		map[string]interface{}{
			"protocol":  "icmp",
			"from_port": 1,
			"to_port":   -1,
			"self":      true,
		},
	}
	group := ec2.SecurityGroup{
		GroupID: aws.String("foo"),
		VPCID:   aws.String("bar"),
	}
	perms := expandIPPerms(group, expanded)

	expected := []ec2.IPPermission{
		ec2.IPPermission{
			IPProtocol: aws.String("icmp"),
			FromPort:   aws.Integer(1),
			ToPort:     aws.Integer(-1),
			IPRanges:   []ec2.IPRange{ec2.IPRange{aws.String("0.0.0.0/0")}},
			UserIDGroupPairs: []ec2.UserIDGroupPair{
				ec2.UserIDGroupPair{
					UserID:  aws.String("foo"),
					GroupID: aws.String("sg-22222"),
				},
				ec2.UserIDGroupPair{
					GroupID: aws.String("sg-22222"),
				},
			},
		},
		ec2.IPPermission{
			IPProtocol: aws.String("icmp"),
			FromPort:   aws.Integer(1),
			ToPort:     aws.Integer(-1),
			UserIDGroupPairs: []ec2.UserIDGroupPair{
				ec2.UserIDGroupPair{
					UserID: aws.String("foo"),
				},
			},
		},
	}

	exp := expected[0]
	perm := perms[0]

	if *exp.FromPort != *perm.FromPort {
		t.Fatalf(
			"Got:\n\n%#v\n\nExpected:\n\n%#v\n",
			*perm.FromPort,
			*exp.FromPort)
	}

	if *exp.IPRanges[0].CIDRIP != *perm.IPRanges[0].CIDRIP {
		t.Fatalf(
			"Got:\n\n%#v\n\nExpected:\n\n%#v\n",
			*perm.IPRanges[0].CIDRIP,
			*exp.IPRanges[0].CIDRIP)
	}

	if *exp.UserIDGroupPairs[0].UserID != *perm.UserIDGroupPairs[0].UserID {
		t.Fatalf(
			"Got:\n\n%#v\n\nExpected:\n\n%#v\n",
			*perm.UserIDGroupPairs[0].UserID,
			*exp.UserIDGroupPairs[0].UserID)
	}

}

func TestExpandIPPerms_nonVPC(t *testing.T) {
	hash := func(v interface{}) int {
		return hashcode.String(v.(string))
	}

	expanded := []interface{}{
		map[string]interface{}{
			"protocol":    "icmp",
			"from_port":   1,
			"to_port":     -1,
			"cidr_blocks": []interface{}{"0.0.0.0/0"},
			"security_groups": schema.NewSet(hash, []interface{}{
				"sg-11111",
				"foo/sg-22222",
			}),
		},
		map[string]interface{}{
			"protocol":  "icmp",
			"from_port": 1,
			"to_port":   -1,
			"self":      true,
		},
	}
	group := ec2.SecurityGroup{
		GroupName: aws.String("foo"),
	}
	perms := expandIPPerms(group, expanded)

	expected := []ec2.IPPermission{
		ec2.IPPermission{
			IPProtocol: aws.String("icmp"),
			FromPort:   aws.Integer(1),
			ToPort:     aws.Integer(-1),
			IPRanges:   []ec2.IPRange{ec2.IPRange{aws.String("0.0.0.0/0")}},
			UserIDGroupPairs: []ec2.UserIDGroupPair{
				ec2.UserIDGroupPair{
					GroupName: aws.String("sg-22222"),
				},
				ec2.UserIDGroupPair{
					GroupName: aws.String("sg-22222"),
				},
			},
		},
		ec2.IPPermission{
			IPProtocol: aws.String("icmp"),
			FromPort:   aws.Integer(1),
			ToPort:     aws.Integer(-1),
			UserIDGroupPairs: []ec2.UserIDGroupPair{
				ec2.UserIDGroupPair{
					GroupName: aws.String("foo"),
				},
			},
		},
	}

	exp := expected[0]
	perm := perms[0]

	if *exp.FromPort != *perm.FromPort {
		t.Fatalf(
			"Got:\n\n%#v\n\nExpected:\n\n%#v\n",
			*perm.FromPort,
			*exp.FromPort)
	}

	if *exp.IPRanges[0].CIDRIP != *perm.IPRanges[0].CIDRIP {
		t.Fatalf(
			"Got:\n\n%#v\n\nExpected:\n\n%#v\n",
			*perm.IPRanges[0].CIDRIP,
			*exp.IPRanges[0].CIDRIP)
	}
}

func TestExpandListeners(t *testing.T) {
	expanded := []interface{}{
		map[string]interface{}{
			"instance_port":     8000,
			"lb_port":           80,
			"instance_protocol": "http",
			"lb_protocol":       "http",
		},
	}
	listeners, err := expandListeners(expanded)
	if err != nil {
		t.Fatalf("bad: %#v", err)
	}

	expected := elb.Listener{
		InstancePort:     aws.Integer(8000),
		LoadBalancerPort: aws.Integer(80),
		InstanceProtocol: aws.String("http"),
		Protocol:         aws.String("http"),
	}

	if !reflect.DeepEqual(listeners[0], expected) {
		t.Fatalf(
			"Got:\n\n%#v\n\nExpected:\n\n%#v\n",
			listeners[0],
			expected)
	}

}

func TestFlattenHealthCheck(t *testing.T) {
	cases := []struct {
		Input  elb.HealthCheck
		Output []map[string]interface{}
	}{
		{
			Input: elb.HealthCheck{
				UnhealthyThreshold: aws.Integer(10),
				HealthyThreshold:   aws.Integer(10),
				Target:             aws.String("HTTP:80/"),
				Timeout:            aws.Integer(30),
				Interval:           aws.Integer(30),
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
		output := flattenHealthCheck(&tc.Input)
		if !reflect.DeepEqual(output, tc.Output) {
			t.Fatalf("Got:\n\n%#v\n\nExpected:\n\n%#v", output, tc.Output)
		}
	}
}

func TestExpandStringList(t *testing.T) {
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

func TestExpandParameters(t *testing.T) {
	expanded := []interface{}{
		map[string]interface{}{
			"name":         "character_set_client",
			"value":        "utf8",
			"apply_method": "immediate",
		},
	}
	parameters, err := expandParameters(expanded)
	if err != nil {
		t.Fatalf("bad: %#v", err)
	}

	expected := rds.Parameter{
		ParameterName:  aws.String("character_set_client"),
		ParameterValue: aws.String("utf8"),
		ApplyMethod:    aws.String("immediate"),
	}

	if !reflect.DeepEqual(parameters[0], expected) {
		t.Fatalf(
			"Got:\n\n%#v\n\nExpected:\n\n%#v\n",
			parameters[0],
			expected)
	}
}

func TestFlattenParameters(t *testing.T) {
	cases := []struct {
		Input  []rds.Parameter
		Output []map[string]interface{}
	}{
		{
			Input: []rds.Parameter{
				rds.Parameter{
					ParameterName:  aws.String("character_set_client"),
					ParameterValue: aws.String("utf8"),
				},
			},
			Output: []map[string]interface{}{
				map[string]interface{}{
					"name":  "character_set_client",
					"value": "utf8",
				},
			},
		},
	}

	for _, tc := range cases {
		output := flattenParameters(tc.Input)
		if !reflect.DeepEqual(output, tc.Output) {
			t.Fatalf("Got:\n\n%#v\n\nExpected:\n\n%#v", output, tc.Output)
		}
	}
}

func TestExpandInstanceString(t *testing.T) {

	expected := []elb.Instance{
		elb.Instance{aws.String("test-one")},
		elb.Instance{aws.String("test-two")},
	}

	ids := []interface{}{
		"test-one",
		"test-two",
	}

	expanded := expandInstanceString(ids)

	if !reflect.DeepEqual(expanded, expected) {
		t.Fatalf("Expand Instance String output did not match.\nGot:\n%#v\n\nexpected:\n%#v", expanded, expected)
	}
}
