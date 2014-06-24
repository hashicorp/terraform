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
	return resource_aws_instance_update_state(rs, instance)
}

func resource_aws_instance_refresh(
	s *terraform.ResourceState,
	meta interface{}) (*terraform.ResourceState, error) {
	p := meta.(*ResourceProvider)
	ec2conn := p.ec2conn

	resp, err := ec2conn.Instances([]string{s.ID}, ec2.NewFilter())
	if err != nil {
		// If the instance was not found, return nil so that we can show
		// that the instance is gone.
		if ec2err, ok := err.(*ec2.Error); ok && ec2err.Code == "InvalidInstanceID.NotFound" {
			return nil, nil
		}

		// Some other error, report it
		return s, err
	}

	instance := &resp.Reservations[0].Instances[0]
	return resource_aws_instance_update_state(s, instance)
}

func resource_aws_instance_update_state(
	s *terraform.ResourceState,
	instance *ec2.Instance) (*terraform.ResourceState, error) {
	s.Attributes["public_dns"] = instance.DNSName
	s.Attributes["public_ip"] = instance.PublicIpAddress
	s.Attributes["private_dns"] = instance.PrivateDNSName
	s.Attributes["private_ip"] = instance.PrivateIpAddress
	return s, nil
}
