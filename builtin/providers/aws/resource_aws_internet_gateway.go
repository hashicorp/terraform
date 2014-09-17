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

func resource_aws_internet_gateway_create(
	s *terraform.InstanceState,
	d *terraform.InstanceDiff,
	meta interface{}) (*terraform.InstanceState, error) {
	p := meta.(*ResourceProvider)
	ec2conn := p.ec2conn

	// Create the gateway
	log.Printf("[DEBUG] Creating internet gateway")
	resp, err := ec2conn.CreateInternetGateway(nil)
	if err != nil {
		return nil, fmt.Errorf("Error creating subnet: %s", err)
	}

	// Get the ID and store it
	ig := &resp.InternetGateway
	s.ID = ig.InternetGatewayId
	log.Printf("[INFO] InternetGateway ID: %s", s.ID)

	// Update our attributes and return
	return resource_aws_internet_gateway_update(s, d, meta)
}

func resource_aws_internet_gateway_update(
	s *terraform.InstanceState,
	d *terraform.InstanceDiff,
	meta interface{}) (*terraform.InstanceState, error) {
	p := meta.(*ResourceProvider)
	ec2conn := p.ec2conn

	// Merge the diff so we have the latest attributes
	rs := s.MergeDiff(d)

	// A note on the states below: the AWS docs (as of July, 2014) say
	// that the states would be: attached, attaching, detached, detaching,
	// but when running, I noticed that the state is usually "available" when
	// it is attached.

	// If we're already attached, detach it first
	if err := resource_aws_internet_gateway_detach(ec2conn, s); err != nil {
		return s, err
	}

	// Set the VPC ID to empty since we're detached at this point
	delete(rs.Attributes, "vpc_id")

	if attr, ok := d.Attributes["vpc_id"]; ok && attr.New != "" {
		err := resource_aws_internet_gateway_attach(ec2conn, s, attr.New)
		if err != nil {
			return rs, err
		}

		rs.Attributes["vpc_id"] = attr.New
	}

	return resource_aws_internet_gateway_update_state(rs, nil)
}

func resource_aws_internet_gateway_destroy(
	s *terraform.InstanceState,
	meta interface{}) error {
	p := meta.(*ResourceProvider)
	ec2conn := p.ec2conn

	// Detach if it is attached
	if err := resource_aws_internet_gateway_detach(ec2conn, s); err != nil {
		return err
	}

	log.Printf("[INFO] Deleting Internet Gateway: %s", s.ID)
	if _, err := ec2conn.DeleteInternetGateway(s.ID); err != nil {
		ec2err, ok := err.(*ec2.Error)
		if ok && ec2err.Code == "InvalidInternetGatewayID.NotFound" {
			return nil
		}

		return fmt.Errorf("Error deleting internet gateway: %s", err)
	}

	// Wait for the internet gateway to actually delete
	log.Printf("[DEBUG] Waiting for internet gateway (%s) to delete", s.ID)
	stateConf := &resource.StateChangeConf{
		Pending: []string{"available"},
		Target:  "",
		Refresh: IGStateRefreshFunc(ec2conn, s.ID),
		Timeout: 10 * time.Minute,
	}
	if _, err := stateConf.WaitForState(); err != nil {
		return fmt.Errorf(
			"Error waiting for internet gateway (%s) to destroy: %s",
			s.ID, err)
	}

	return nil
}

func resource_aws_internet_gateway_refresh(
	s *terraform.InstanceState,
	meta interface{}) (*terraform.InstanceState, error) {
	p := meta.(*ResourceProvider)
	ec2conn := p.ec2conn

	igRaw, _, err := IGStateRefreshFunc(ec2conn, s.ID)()
	if err != nil {
		return s, err
	}
	if igRaw == nil {
		return nil, nil
	}

	ig := igRaw.(*ec2.InternetGateway)
	return resource_aws_internet_gateway_update_state(s, ig)
}

func resource_aws_internet_gateway_diff(
	s *terraform.InstanceState,
	c *terraform.ResourceConfig,
	meta interface{}) (*terraform.InstanceDiff, error) {
	b := &diff.ResourceBuilder{
		Attrs: map[string]diff.AttrType{
			"vpc_id": diff.AttrTypeUpdate,
		},
	}

	return b.Diff(s, c)
}

func resource_aws_internet_gateway_attach(
	ec2conn *ec2.EC2,
	s *terraform.InstanceState,
	vpcId string) error {
	log.Printf(
		"[INFO] Attaching Internet Gateway '%s' to VPC '%s'",
		s.ID,
		vpcId)
	_, err := ec2conn.AttachInternetGateway(s.ID, vpcId)
	if err != nil {
		return err
	}

	// Wait for it to be fully attached before continuing
	log.Printf("[DEBUG] Waiting for internet gateway (%s) to attach", s.ID)
	stateConf := &resource.StateChangeConf{
		Pending: []string{"detached", "attaching"},
		Target:  "available",
		Refresh: IGAttachStateRefreshFunc(ec2conn, s.ID, "available"),
		Timeout: 1 * time.Minute,
	}
	if _, err := stateConf.WaitForState(); err != nil {
		return fmt.Errorf(
			"Error waiting for internet gateway (%s) to attach: %s",
			s.ID, err)
	}

	return nil
}

func resource_aws_internet_gateway_detach(
	ec2conn *ec2.EC2,
	s *terraform.InstanceState) error {
	if s.Attributes["vpc_id"] == "" {
		return nil
	}

	log.Printf(
		"[INFO] Detaching Internet Gateway '%s' from VPC '%s'",
		s.ID,
		s.Attributes["vpc_id"])
	wait := true
	_, err := ec2conn.DetachInternetGateway(s.ID, s.Attributes["vpc_id"])
	if err != nil {
		ec2err, ok := err.(*ec2.Error)
		if ok {
			if ec2err.Code == "InvalidInternetGatewayID.NotFound" {
				err = nil
				wait = false
			} else if ec2err.Code == "Gateway.NotAttached" {
				err = nil
				wait = false
			}
		}

		if err != nil {
			return err
		}
	}

	delete(s.Attributes, "vpc_id")

	if !wait {
		return nil
	}

	// Wait for it to be fully detached before continuing
	log.Printf("[DEBUG] Waiting for internet gateway (%s) to detach", s.ID)
	stateConf := &resource.StateChangeConf{
		Pending: []string{"attached", "detaching", "available"},
		Target:  "detached",
		Refresh: IGAttachStateRefreshFunc(ec2conn, s.ID, "detached"),
		Timeout: 1 * time.Minute,
	}
	if _, err := stateConf.WaitForState(); err != nil {
		return fmt.Errorf(
			"Error waiting for internet gateway (%s) to detach: %s",
			s.ID, err)
	}

	return nil
}

func resource_aws_internet_gateway_update_state(
	s *terraform.InstanceState,
	ig *ec2.InternetGateway) (*terraform.InstanceState, error) {
	return s, nil
}

// IGStateRefreshFunc returns a resource.StateRefreshFunc that is used to watch
// an internet gateway.
func IGStateRefreshFunc(conn *ec2.EC2, id string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		resp, err := conn.DescribeInternetGateways([]string{id}, ec2.NewFilter())
		if err != nil {
			ec2err, ok := err.(*ec2.Error)
			if ok && ec2err.Code == "InvalidInternetGatewayID.NotFound" {
				resp = nil
			} else {
				log.Printf("[ERROR] Error on IGStateRefresh: %s", err)
				return nil, "", err
			}
		}

		if resp == nil {
			// Sometimes AWS just has consistency issues and doesn't see
			// our instance yet. Return an empty state.
			return nil, "", nil
		}

		ig := &resp.InternetGateways[0]
		return ig, "available", nil
	}
}

// IGAttachStateRefreshFunc returns a resource.StateRefreshFunc that is used
// watch the state of an internet gateway's attachment.
func IGAttachStateRefreshFunc(conn *ec2.EC2, id string, expected string) resource.StateRefreshFunc {
	var start time.Time
	return func() (interface{}, string, error) {
		if start.IsZero() {
			start = time.Now()
		}

		resp, err := conn.DescribeInternetGateways([]string{id}, ec2.NewFilter())
		if err != nil {
			ec2err, ok := err.(*ec2.Error)
			if ok && ec2err.Code == "InvalidInternetGatewayID.NotFound" {
				resp = nil
			} else {
				log.Printf("[ERROR] Error on IGStateRefresh: %s", err)
				return nil, "", err
			}
		}

		if resp == nil {
			// Sometimes AWS just has consistency issues and doesn't see
			// our instance yet. Return an empty state.
			return nil, "", nil
		}

		ig := &resp.InternetGateways[0]

		if time.Now().Sub(start) > 10*time.Second {
			return ig, expected, nil
		}

		if len(ig.Attachments) == 0 {
			// No attachments, we're detached
			return ig, "detached", nil
		}

		return ig, ig.Attachments[0].State, nil
	}
}
