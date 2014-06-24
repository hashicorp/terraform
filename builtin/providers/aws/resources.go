package aws

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/goamz/ec2"
)

var resourceMap *resource.Map

func init() {
	resourceMap = &resource.Map{
		Mapping: map[string]resource.Resource{
			"aws_instance": resource.Resource{
				Create:  resource_aws_instance_create,
				Refresh: resource_aws_instance_refresh,
			},
		},
	}
}

func resource_aws_instance_create(
	s *terraform.ResourceState,
	d *terraform.ResourceDiff,
	meta interface{}) (*terraform.ResourceState, error) {
	p := meta.(*ResourceProvider)
	ec2conn := p.ec2conn

	runOpts := &ec2.RunInstances{
		ImageId:      d.Attributes["ami"].New,
		InstanceType: d.Attributes["instance_type"].New,
	}

	log.Printf("Run configuration: %#v", runOpts)
	runResp, err := ec2conn.RunInstances(runOpts)
	if err != nil {
		return nil, fmt.Errorf("Error launching source instance: %s", err)
	}

	instance := &runResp.Instances[0]
	log.Printf("Instance ID: %s", instance.InstanceId)

	// TODO(mitchellh): wait until running

	rs := s.MergeDiff(d)
	rs.ID = instance.InstanceId
	rs.Attributes["public_dns"] = instance.DNSName
	rs.Attributes["public_ip"] = instance.PublicIpAddress
	rs.Attributes["private_dns"] = instance.PrivateDNSName
	rs.Attributes["private_ip"] = instance.PrivateIpAddress

	return rs, nil
}

func resource_aws_instance_refresh(
	s *terraform.ResourceState,
	meta interface{}) (*terraform.ResourceState, error) {
	if s.ID != "" {
		panic("OH MY WOW")
	}

	return s, nil
}
