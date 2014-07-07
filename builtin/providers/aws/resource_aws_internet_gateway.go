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
	s *terraform.ResourceState,
	d *terraform.ResourceDiff,
	meta interface{}) (*terraform.ResourceState, error) {
	p := meta.(*ResourceProvider)
	ec2conn := p.ec2conn

	// Merge the diff so that we have all the proper attributes
	s = s.MergeDiff(d)

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
	return resource_aws_internet_gateway_update_state(s, ig)
}

func resource_aws_internet_gateway_update(
	s *terraform.ResourceState,
	d *terraform.ResourceDiff,
	meta interface{}) (*terraform.ResourceState, error) {
	// This should never be called because we have no update-able
	// attributes
	panic("Update for internet gateway is not supported")

	return nil, nil
}

func resource_aws_internet_gateway_destroy(
	s *terraform.ResourceState,
	meta interface{}) error {
	p := meta.(*ResourceProvider)
	ec2conn := p.ec2conn

	log.Printf("[INFO] Deleting Internet Gateway: %s", s.ID)
	if _, err := ec2conn.DeleteInternetGateway(s.ID); err != nil {
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
			"Error waiting for internet gateway (%s) to destroy",
			s.ID, err)
	}

	return nil
}

func resource_aws_internet_gateway_refresh(
	s *terraform.ResourceState,
	meta interface{}) (*terraform.ResourceState, error) {
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
	s *terraform.ResourceState,
	c *terraform.ResourceConfig,
	meta interface{}) (*terraform.ResourceDiff, error) {
	b := &diff.ResourceBuilder{
		Attrs: map[string]diff.AttrType{},
	}

	return b.Diff(s, c)
}

func resource_aws_internet_gateway_update_state(
	s *terraform.ResourceState,
	ig *ec2.InternetGateway) (*terraform.ResourceState, error) {
	return s, nil
}

// IGStateRefreshFunc returns a resource.StateRefreshFunc that is used to watch
// an internet gateway.
func IGStateRefreshFunc(conn *ec2.EC2, id string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		resp, err := conn.DescribeInternetGateways([]string{id}, ec2.NewFilter())
		if err != nil {
			if ec2err, ok := err.(*ec2.Error); ok && ec2err.Code == "InvalidInternetGatewayID.NotFound" {
				resp = nil
			} else {
				log.Printf("Error on IGStateRefresh: %s", err)
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
