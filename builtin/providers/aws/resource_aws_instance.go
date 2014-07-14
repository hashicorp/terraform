package aws

import (
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/terraform/helper/diff"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/goamz/ec2"
)

func resource_aws_instance_create(
	s *terraform.ResourceState,
	d *terraform.ResourceDiff,
	meta interface{}) (*terraform.ResourceState, error) {
	p := meta.(*ResourceProvider)
	ec2conn := p.ec2conn

	// Merge the diff into the state so that we have all the attributes
	// properly.
	rs := s.MergeDiff(d)

	// Create the instance
	runOpts := &ec2.RunInstances{
		ImageId:      rs.Attributes["ami"],
		InstanceType: rs.Attributes["instance_type"],
		SubnetId:     rs.Attributes["subnet_id"],
	}
	log.Printf("[DEBUG] Run configuration: %#v", runOpts)
	runResp, err := ec2conn.RunInstances(runOpts)
	if err != nil {
		return nil, fmt.Errorf("Error launching source instance: %s", err)
	}

	instance := &runResp.Instances[0]
	log.Printf("[INFO] Instance ID: %s", instance.InstanceId)

	// Store the resulting ID so we can look this up later
	rs.ID = instance.InstanceId

	// Wait for the instance to become running so we can get some attributes
	// that aren't available until later.
	log.Printf(
		"[DEBUG] Waiting for instance (%s) to become running",
		instance.InstanceId)

	stateConf := &resource.StateChangeConf{
		Pending: []string{"pending"},
		Target:  "running",
		Refresh: InstanceStateRefreshFunc(ec2conn, instance.InstanceId),
		Timeout: 10 * time.Minute,
	}

	instanceRaw, err := stateConf.WaitForState()

	if err != nil {
		return rs, fmt.Errorf(
			"Error waiting for instance (%s) to become ready: %s",
			instance.InstanceId, err)
	}

	instance = instanceRaw.(*ec2.Instance)

	// Set our attributes
	return resource_aws_instance_update_state(rs, instance)
}

func resource_aws_instance_destroy(
	s *terraform.ResourceState,
	meta interface{}) error {
	p := meta.(*ResourceProvider)
	ec2conn := p.ec2conn

	log.Printf("[INFO] Terminating instance: %s", s.ID)
	if _, err := ec2conn.TerminateInstances([]string{s.ID}); err != nil {
		return fmt.Errorf("Error terminating instance: %s", err)
	}

	log.Printf(
		"[DEBUG] Waiting for instance (%s) to become terminated",
		s.ID)

	stateConf := &resource.StateChangeConf{
		Pending: []string{"pending", "running", "shutting-down", "stopped", "stopping"},
		Target:  "terminated",
		Refresh: InstanceStateRefreshFunc(ec2conn, s.ID),
		Timeout: 10 * time.Minute,
	}

	_, err := stateConf.WaitForState()

	if err != nil {
		return fmt.Errorf(
			"Error waiting for instance (%s) to terminate: %s",
			s.ID, err)
	}

	return nil
}

func resource_aws_instance_diff(
	s *terraform.ResourceState,
	c *terraform.ResourceConfig,
	meta interface{}) (*terraform.ResourceDiff, error) {
	b := &diff.ResourceBuilder{
		Attrs: map[string]diff.AttrType{
			"ami":               diff.AttrTypeCreate,
			"availability_zone": diff.AttrTypeCreate,
			"instance_type":     diff.AttrTypeCreate,
			"subnet_id":         diff.AttrTypeCreate,
		},

		ComputedAttrs: []string{
			"availability_zone",
			"public_dns",
			"public_ip",
			"private_dns",
			"private_ip",
		},
	}

	return b.Diff(s, c)
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

	// If nothing was found, then return no state
	if len(resp.Reservations) == 0 {
		return nil, nil
	}

	instance := &resp.Reservations[0].Instances[0]

	// If the instance is terminated, then it is gone
	if instance.State.Name == "terminated" {
		return nil, nil
	}

	return resource_aws_instance_update_state(s, instance)
}

func resource_aws_instance_update_state(
	s *terraform.ResourceState,
	instance *ec2.Instance) (*terraform.ResourceState, error) {
	s.Attributes["availability_zone"] = instance.AvailZone
	s.Attributes["public_dns"] = instance.DNSName
	s.Attributes["public_ip"] = instance.PublicIpAddress
	s.Attributes["private_dns"] = instance.PrivateDNSName
	s.Attributes["private_ip"] = instance.PrivateIpAddress
	s.Attributes["subnet_id"] = instance.SubnetId
	s.Dependencies = nil

	if instance.SubnetId != "" {
		s.Dependencies = append(s.Dependencies,
			terraform.ResourceDependency{ID: instance.SubnetId},
		)
	}

	return s, nil
}

// InstanceStateRefreshFunc returns a resource.StateRefreshFunc that is used to watch
// an EC2 instance.
func InstanceStateRefreshFunc(conn *ec2.EC2, instanceID string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		resp, err := conn.Instances([]string{instanceID}, ec2.NewFilter())
		if err != nil {
			if ec2err, ok := err.(*ec2.Error); ok && ec2err.Code == "InvalidInstanceID.NotFound" {
				// Set this to nil as if we didn't find anything.
				resp = nil
			} else {
				log.Printf("Error on InstanceStateRefresh: %s", err)
				return nil, "", err
			}
		}

		if resp == nil || len(resp.Reservations) == 0 || len(resp.Reservations[0].Instances) == 0 {
			// Sometimes AWS just has consistency issues and doesn't see
			// our instance yet. Return an empty state.
			return nil, "", nil
		}

		i := &resp.Reservations[0].Instances[0]
		return i, i.State.Name, nil
	}
}
