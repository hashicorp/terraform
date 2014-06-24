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

	log.Printf("[DEBUG] Run configuration: %#v", runOpts)
	runResp, err := ec2conn.RunInstances(runOpts)
	if err != nil {
		return nil, fmt.Errorf("Error launching source instance: %s", err)
	}

	instance := &runResp.Instances[0]
	log.Printf("[INFO] Instance ID: %s", instance.InstanceId)

	// Store the resource state now so that we can return it in the case
	// of any errors.
	rs := new(terraform.ResourceState)
	rs.ID = instance.InstanceId

	// Wait for the instance to become running so we can get some attributes
	// that aren't available until later.
	log.Printf(
		"[DEBUG] Waiting for instance (%s) to become running",
		instance.InstanceId)
	instanceRaw, err := WaitForState(&StateChangeConf{
		Pending: []string{"pending"},
		Target:  "running",
		Refresh: InstanceStateRefreshFunc(ec2conn, instance),
	})
	if err != nil {
		return rs, fmt.Errorf(
			"Error waiting for instance (%s) to become ready: %s",
			instance.InstanceId, err)
	}
	instance = instanceRaw.(*ec2.Instance)

	// Set our attributes
	rs = rs.MergeDiff(d)
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
