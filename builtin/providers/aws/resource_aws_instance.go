package aws

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/terraform/flatmap"
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
	delete(rs.Attributes, "source_dest_check")

	// Figure out user data
	userData := ""
	if attr, ok := d.Attributes["user_data"]; ok {
		userData = attr.NewExtra.(string)
	}

	associatePublicIPAddress := false
	if rs.Attributes["associate_public_ip_address"] == "true" {
		associatePublicIPAddress = true
	}

	// Build the creation struct
	runOpts := &ec2.RunInstances{
		ImageId:                  rs.Attributes["ami"],
		AvailZone:                rs.Attributes["availability_zone"],
		InstanceType:             rs.Attributes["instance_type"],
		KeyName:                  rs.Attributes["key_name"],
		SubnetId:                 rs.Attributes["subnet_id"],
		AssociatePublicIpAddress: associatePublicIPAddress,
		UserData:                 []byte(userData),
	}
	if raw := flatmap.Expand(rs.Attributes, "security_groups"); raw != nil {
		if sgs, ok := raw.([]interface{}); ok {
			for _, sg := range sgs {
				str, ok := sg.(string)
				if !ok {
					continue
				}

				var g ec2.SecurityGroup
				if runOpts.SubnetId != "" {
					g.Id = str
				} else {
					g.Name = str
				}

				runOpts.SecurityGroups = append(runOpts.SecurityGroups, g)
			}
		}
	}

	// Create the instance
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
		Pending:    []string{"pending"},
		Target:     "running",
		Refresh:    InstanceStateRefreshFunc(ec2conn, instance.InstanceId),
		Timeout:    10 * time.Minute,
		Delay:      10 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	instanceRaw, err := stateConf.WaitForState()

	if err != nil {
		return rs, fmt.Errorf(
			"Error waiting for instance (%s) to become ready: %s",
			instance.InstanceId, err)
	}

	instance = instanceRaw.(*ec2.Instance)

	// Initialize the connection info
	rs.ConnInfo["type"] = "ssh"
	rs.ConnInfo["host"] = instance.PublicIpAddress

	// Set our attributes
	rs, err = resource_aws_instance_update_state(rs, instance)
	if err != nil {
		return rs, err
	}

	// Update if we need to
	return resource_aws_instance_update(rs, d, meta)
}

func resource_aws_instance_update(
	s *terraform.ResourceState,
	d *terraform.ResourceDiff,
	meta interface{}) (*terraform.ResourceState, error) {
	p := meta.(*ResourceProvider)
	ec2conn := p.ec2conn
	rs := s.MergeDiff(d)

	modify := false
	opts := new(ec2.ModifyInstance)

	if attr, ok := d.Attributes["source_dest_check"]; ok {
		modify = true
		opts.SourceDestCheck = attr.New != "" && attr.New != "false"
		opts.SetSourceDestCheck = true
		rs.Attributes["source_dest_check"] = strconv.FormatBool(
			opts.SourceDestCheck)
	}

	if modify {
		log.Printf("[INFO] Modifing instance %s: %#v", s.ID, opts)
		if _, err := ec2conn.ModifyInstance(s.ID, opts); err != nil {
			return s, err
		}

		// TODO(mitchellh): wait for the attributes we modified to
		// persist the change...
	}

	return rs, nil
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
		Pending:    []string{"pending", "running", "shutting-down", "stopped", "stopping"},
		Target:     "terminated",
		Refresh:    InstanceStateRefreshFunc(ec2conn, s.ID),
		Timeout:    10 * time.Minute,
		Delay:      10 * time.Second,
		MinTimeout: 3 * time.Second,
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
			"ami":                         diff.AttrTypeCreate,
			"availability_zone":           diff.AttrTypeCreate,
			"instance_type":               diff.AttrTypeCreate,
			"key_name":                    diff.AttrTypeCreate,
			"security_groups":             diff.AttrTypeCreate,
			"subnet_id":                   diff.AttrTypeCreate,
			"source_dest_check":           diff.AttrTypeUpdate,
			"user_data":                   diff.AttrTypeCreate,
			"associate_public_ip_address": diff.AttrTypeCreate,
		},

		ComputedAttrs: []string{
			"availability_zone",
			"key_name",
			"public_dns",
			"public_ip",
			"private_dns",
			"private_ip",
			"security_groups",
			"subnet_id",
		},

		PreProcess: map[string]diff.PreProcessFunc{
			"user_data": func(v string) string {
				hash := sha1.Sum([]byte(v))
				return hex.EncodeToString(hash[:])
			},
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
	s.Attributes["key_name"] = instance.KeyName
	s.Attributes["public_dns"] = instance.DNSName
	s.Attributes["public_ip"] = instance.PublicIpAddress
	s.Attributes["private_dns"] = instance.PrivateDNSName
	s.Attributes["private_ip"] = instance.PrivateIpAddress
	s.Attributes["subnet_id"] = instance.SubnetId
	s.Dependencies = nil

	// Extract the existing security groups
	useID := false
	if raw := flatmap.Expand(s.Attributes, "security_groups"); raw != nil {
		if sgs, ok := raw.([]interface{}); ok {
			for _, sg := range sgs {
				str, ok := sg.(string)
				if !ok {
					continue
				}

				if strings.HasPrefix(str, "sg-") {
					useID = true
					break
				}
			}
		}
	}

	// Build up the security groups
	sgs := make([]string, len(instance.SecurityGroups))
	for i, sg := range instance.SecurityGroups {
		if instance.SubnetId != "" && useID {
			sgs[i] = sg.Id
		} else {
			sgs[i] = sg.Name
		}

		s.Dependencies = append(s.Dependencies,
			terraform.ResourceDependency{ID: sg.Id},
		)
	}
	flatmap.Map(s.Attributes).Merge(flatmap.Flatten(map[string]interface{}{
		"security_groups": sgs,
	}))

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
