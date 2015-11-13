package aws

import (
	"reflect"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/elasticache"
	"github.com/aws/aws-sdk-go/service/elb"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/aws/aws-sdk-go/service/route53"
	"github.com/hashicorp/terraform/flatmap"
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
	hash := schema.HashString

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
	group := &ec2.SecurityGroup{
		GroupId: aws.String("foo"),
		VpcId:   aws.String("bar"),
	}
	perms, err := expandIPPerms(group, expanded)
	if err != nil {
		t.Fatalf("error expanding perms: %v", err)
	}

	expected := []ec2.IpPermission{
		ec2.IpPermission{
			IpProtocol: aws.String("icmp"),
			FromPort:   aws.Int64(int64(1)),
			ToPort:     aws.Int64(int64(-1)),
			IpRanges:   []*ec2.IpRange{&ec2.IpRange{CidrIp: aws.String("0.0.0.0/0")}},
			UserIdGroupPairs: []*ec2.UserIdGroupPair{
				&ec2.UserIdGroupPair{
					UserId:  aws.String("foo"),
					GroupId: aws.String("sg-22222"),
				},
				&ec2.UserIdGroupPair{
					GroupId: aws.String("sg-22222"),
				},
			},
		},
		ec2.IpPermission{
			IpProtocol: aws.String("icmp"),
			FromPort:   aws.Int64(int64(1)),
			ToPort:     aws.Int64(int64(-1)),
			UserIdGroupPairs: []*ec2.UserIdGroupPair{
				&ec2.UserIdGroupPair{
					UserId: aws.String("foo"),
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

	if *exp.IpRanges[0].CidrIp != *perm.IpRanges[0].CidrIp {
		t.Fatalf(
			"Got:\n\n%#v\n\nExpected:\n\n%#v\n",
			*perm.IpRanges[0].CidrIp,
			*exp.IpRanges[0].CidrIp)
	}

	if *exp.UserIdGroupPairs[0].UserId != *perm.UserIdGroupPairs[0].UserId {
		t.Fatalf(
			"Got:\n\n%#v\n\nExpected:\n\n%#v\n",
			*perm.UserIdGroupPairs[0].UserId,
			*exp.UserIdGroupPairs[0].UserId)
	}

}

func TestExpandIPPerms_NegOneProtocol(t *testing.T) {
	hash := schema.HashString

	expanded := []interface{}{
		map[string]interface{}{
			"protocol":    "-1",
			"from_port":   0,
			"to_port":     0,
			"cidr_blocks": []interface{}{"0.0.0.0/0"},
			"security_groups": schema.NewSet(hash, []interface{}{
				"sg-11111",
				"foo/sg-22222",
			}),
		},
	}
	group := &ec2.SecurityGroup{
		GroupId: aws.String("foo"),
		VpcId:   aws.String("bar"),
	}

	perms, err := expandIPPerms(group, expanded)
	if err != nil {
		t.Fatalf("error expanding perms: %v", err)
	}

	expected := []ec2.IpPermission{
		ec2.IpPermission{
			IpProtocol: aws.String("-1"),
			FromPort:   aws.Int64(int64(0)),
			ToPort:     aws.Int64(int64(0)),
			IpRanges:   []*ec2.IpRange{&ec2.IpRange{CidrIp: aws.String("0.0.0.0/0")}},
			UserIdGroupPairs: []*ec2.UserIdGroupPair{
				&ec2.UserIdGroupPair{
					UserId:  aws.String("foo"),
					GroupId: aws.String("sg-22222"),
				},
				&ec2.UserIdGroupPair{
					GroupId: aws.String("sg-22222"),
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

	if *exp.IpRanges[0].CidrIp != *perm.IpRanges[0].CidrIp {
		t.Fatalf(
			"Got:\n\n%#v\n\nExpected:\n\n%#v\n",
			*perm.IpRanges[0].CidrIp,
			*exp.IpRanges[0].CidrIp)
	}

	if *exp.UserIdGroupPairs[0].UserId != *perm.UserIdGroupPairs[0].UserId {
		t.Fatalf(
			"Got:\n\n%#v\n\nExpected:\n\n%#v\n",
			*perm.UserIdGroupPairs[0].UserId,
			*exp.UserIdGroupPairs[0].UserId)
	}

	// Now test the error case. This *should* error when either from_port
	// or to_port is not zero, but protocol is "-1".
	errorCase := []interface{}{
		map[string]interface{}{
			"protocol":    "-1",
			"from_port":   0,
			"to_port":     65535,
			"cidr_blocks": []interface{}{"0.0.0.0/0"},
			"security_groups": schema.NewSet(hash, []interface{}{
				"sg-11111",
				"foo/sg-22222",
			}),
		},
	}
	securityGroups := &ec2.SecurityGroup{
		GroupId: aws.String("foo"),
		VpcId:   aws.String("bar"),
	}

	_, expandErr := expandIPPerms(securityGroups, errorCase)
	if expandErr == nil {
		t.Fatal("expandIPPerms should have errored!")
	}
}

func TestExpandIPPerms_nonVPC(t *testing.T) {
	hash := schema.HashString

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
	group := &ec2.SecurityGroup{
		GroupName: aws.String("foo"),
	}
	perms, err := expandIPPerms(group, expanded)
	if err != nil {
		t.Fatalf("error expanding perms: %v", err)
	}

	expected := []ec2.IpPermission{
		ec2.IpPermission{
			IpProtocol: aws.String("icmp"),
			FromPort:   aws.Int64(int64(1)),
			ToPort:     aws.Int64(int64(-1)),
			IpRanges:   []*ec2.IpRange{&ec2.IpRange{CidrIp: aws.String("0.0.0.0/0")}},
			UserIdGroupPairs: []*ec2.UserIdGroupPair{
				&ec2.UserIdGroupPair{
					GroupName: aws.String("sg-22222"),
				},
				&ec2.UserIdGroupPair{
					GroupName: aws.String("sg-22222"),
				},
			},
		},
		ec2.IpPermission{
			IpProtocol: aws.String("icmp"),
			FromPort:   aws.Int64(int64(1)),
			ToPort:     aws.Int64(int64(-1)),
			UserIdGroupPairs: []*ec2.UserIdGroupPair{
				&ec2.UserIdGroupPair{
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

	if *exp.IpRanges[0].CidrIp != *perm.IpRanges[0].CidrIp {
		t.Fatalf(
			"Got:\n\n%#v\n\nExpected:\n\n%#v\n",
			*perm.IpRanges[0].CidrIp,
			*exp.IpRanges[0].CidrIp)
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
		map[string]interface{}{
			"instance_port":      8000,
			"lb_port":            80,
			"instance_protocol":  "https",
			"lb_protocol":        "https",
			"ssl_certificate_id": "something",
		},
	}
	listeners, err := expandListeners(expanded)
	if err != nil {
		t.Fatalf("bad: %#v", err)
	}

	expected := &elb.Listener{
		InstancePort:     aws.Int64(int64(8000)),
		LoadBalancerPort: aws.Int64(int64(80)),
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

// this test should produce an error from expandlisteners on an invalid
// combination
func TestExpandListeners_invalid(t *testing.T) {
	expanded := []interface{}{
		map[string]interface{}{
			"instance_port":      8000,
			"lb_port":            80,
			"instance_protocol":  "http",
			"lb_protocol":        "http",
			"ssl_certificate_id": "something",
		},
	}
	_, err := expandListeners(expanded)
	if err != nil {
		// Check the error we got
		if !strings.Contains(err.Error(), "ssl_certificate_id may be set only when protocol") {
			t.Fatalf("Got error in TestExpandListeners_invalid, but not what we expected: %s", err)
		}
	}

	if err == nil {
		t.Fatalf("Expected TestExpandListeners_invalid to fail, but passed")
	}
}

func TestFlattenHealthCheck(t *testing.T) {
	cases := []struct {
		Input  *elb.HealthCheck
		Output []map[string]interface{}
	}{
		{
			Input: &elb.HealthCheck{
				UnhealthyThreshold: aws.Int64(int64(10)),
				HealthyThreshold:   aws.Int64(int64(10)),
				Target:             aws.String("HTTP:80/"),
				Timeout:            aws.Int64(int64(30)),
				Interval:           aws.Int64(int64(30)),
			},
			Output: []map[string]interface{}{
				map[string]interface{}{
					"unhealthy_threshold": int64(10),
					"healthy_threshold":   int64(10),
					"target":              "HTTP:80/",
					"timeout":             int64(30),
					"interval":            int64(30),
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

func TestExpandStringList(t *testing.T) {
	expanded := flatmap.Expand(testConf(), "availability_zones").([]interface{})
	stringList := expandStringList(expanded)
	expected := []*string{
		aws.String("us-east-1a"),
		aws.String("us-east-1b"),
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

	expected := &rds.Parameter{
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

func TestExpandElasticacheParameters(t *testing.T) {
	expanded := []interface{}{
		map[string]interface{}{
			"name":         "activerehashing",
			"value":        "yes",
			"apply_method": "immediate",
		},
	}
	parameters, err := expandElastiCacheParameters(expanded)
	if err != nil {
		t.Fatalf("bad: %#v", err)
	}

	expected := &elasticache.ParameterNameValue{
		ParameterName:  aws.String("activerehashing"),
		ParameterValue: aws.String("yes"),
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
		Input  []*rds.Parameter
		Output []map[string]interface{}
	}{
		{
			Input: []*rds.Parameter{
				&rds.Parameter{
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

func TestFlattenElasticacheParameters(t *testing.T) {
	cases := []struct {
		Input  []*elasticache.Parameter
		Output []map[string]interface{}
	}{
		{
			Input: []*elasticache.Parameter{
				&elasticache.Parameter{
					ParameterName:  aws.String("activerehashing"),
					ParameterValue: aws.String("yes"),
				},
			},
			Output: []map[string]interface{}{
				map[string]interface{}{
					"name":  "activerehashing",
					"value": "yes",
				},
			},
		},
	}

	for _, tc := range cases {
		output := flattenElastiCacheParameters(tc.Input)
		if !reflect.DeepEqual(output, tc.Output) {
			t.Fatalf("Got:\n\n%#v\n\nExpected:\n\n%#v", output, tc.Output)
		}
	}
}

func TestExpandInstanceString(t *testing.T) {

	expected := []*elb.Instance{
		&elb.Instance{InstanceId: aws.String("test-one")},
		&elb.Instance{InstanceId: aws.String("test-two")},
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

func TestFlattenNetworkInterfacesPrivateIPAddresses(t *testing.T) {
	expanded := []*ec2.NetworkInterfacePrivateIpAddress{
		&ec2.NetworkInterfacePrivateIpAddress{PrivateIpAddress: aws.String("192.168.0.1")},
		&ec2.NetworkInterfacePrivateIpAddress{PrivateIpAddress: aws.String("192.168.0.2")},
	}

	result := flattenNetworkInterfacesPrivateIPAddresses(expanded)

	if result == nil {
		t.Fatal("result was nil")
	}

	if len(result) != 2 {
		t.Fatalf("expected result had %d elements, but got %d", 2, len(result))
	}

	if result[0] != "192.168.0.1" {
		t.Fatalf("expected ip to be 192.168.0.1, but was %s", result[0])
	}

	if result[1] != "192.168.0.2" {
		t.Fatalf("expected ip to be 192.168.0.2, but was %s", result[1])
	}
}

func TestFlattenGroupIdentifiers(t *testing.T) {
	expanded := []*ec2.GroupIdentifier{
		&ec2.GroupIdentifier{GroupId: aws.String("sg-001")},
		&ec2.GroupIdentifier{GroupId: aws.String("sg-002")},
	}

	result := flattenGroupIdentifiers(expanded)

	if len(result) != 2 {
		t.Fatalf("expected result had %d elements, but got %d", 2, len(result))
	}

	if result[0] != "sg-001" {
		t.Fatalf("expected id to be sg-001, but was %s", result[0])
	}

	if result[1] != "sg-002" {
		t.Fatalf("expected id to be sg-002, but was %s", result[1])
	}
}

func TestExpandPrivateIPAddresses(t *testing.T) {

	ip1 := "192.168.0.1"
	ip2 := "192.168.0.2"
	flattened := []interface{}{
		ip1,
		ip2,
	}

	result := expandPrivateIPAddresses(flattened)

	if len(result) != 2 {
		t.Fatalf("expected result had %d elements, but got %d", 2, len(result))
	}

	if *result[0].PrivateIpAddress != "192.168.0.1" || !*result[0].Primary {
		t.Fatalf("expected ip to be 192.168.0.1 and Primary, but got %v, %t", *result[0].PrivateIpAddress, *result[0].Primary)
	}

	if *result[1].PrivateIpAddress != "192.168.0.2" || *result[1].Primary {
		t.Fatalf("expected ip to be 192.168.0.2 and not Primary, but got %v, %t", *result[1].PrivateIpAddress, *result[1].Primary)
	}
}

func TestFlattenAttachment(t *testing.T) {
	expanded := &ec2.NetworkInterfaceAttachment{
		InstanceId:   aws.String("i-00001"),
		DeviceIndex:  aws.Int64(int64(1)),
		AttachmentId: aws.String("at-002"),
	}

	result := flattenAttachment(expanded)

	if result == nil {
		t.Fatal("expected result to have value, but got nil")
	}

	if result["instance"] != "i-00001" {
		t.Fatalf("expected instance to be i-00001, but got %s", result["instance"])
	}

	if result["device_index"] != int64(1) {
		t.Fatalf("expected device_index to be 1, but got %d", result["device_index"])
	}

	if result["attachment_id"] != "at-002" {
		t.Fatalf("expected attachment_id to be at-002, but got %s", result["attachment_id"])
	}
}

func TestFlattenResourceRecords(t *testing.T) {
	expanded := []*route53.ResourceRecord{
		&route53.ResourceRecord{
			Value: aws.String("127.0.0.1"),
		},
		&route53.ResourceRecord{
			Value: aws.String("127.0.0.3"),
		},
	}

	result := flattenResourceRecords(expanded)

	if result == nil {
		t.Fatal("expected result to have value, but got nil")
	}

	if len(result) != 2 {
		t.Fatal("expected result to have value, but got nil")
	}
}
