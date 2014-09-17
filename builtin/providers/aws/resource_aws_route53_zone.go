package aws

import (
	"log"
	"strings"
	"time"

	"github.com/hashicorp/terraform/helper/config"
	"github.com/hashicorp/terraform/helper/diff"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/goamz/route53"
)

func resource_aws_r53_zone_validation() *config.Validator {
	return &config.Validator{
		Required: []string{
			"name",
		},
	}
}

func resource_aws_r53_zone_create(
	s *terraform.InstanceState,
	d *terraform.InstanceDiff,
	meta interface{}) (*terraform.InstanceState, error) {
	p := meta.(*ResourceProvider)
	r53 := p.route53

	// Merge the diff into the state so that we have all the attributes
	// properly.
	rs := s.MergeDiff(d)

	req := &route53.CreateHostedZoneRequest{
		Name:    rs.Attributes["name"],
		Comment: "Managed by Terraform",
	}
	log.Printf("[DEBUG] Creating Route53 hosted zone: %s", req.Name)
	resp, err := r53.CreateHostedZone(req)
	if err != nil {
		return rs, err
	}

	// Store the zone_id
	zone := route53.CleanZoneID(resp.HostedZone.ID)
	rs.ID = zone
	rs.Attributes["zone_id"] = zone

	// Wait until we are done initializing
	wait := resource.StateChangeConf{
		Delay:      30 * time.Second,
		Pending:    []string{"PENDING"},
		Target:     "INSYNC",
		Timeout:    10 * time.Minute,
		MinTimeout: 5 * time.Second,
		Refresh: func() (result interface{}, state string, err error) {
			return resource_aws_r53_wait(r53, resp.ChangeInfo.ID)
		},
	}
	_, err = wait.WaitForState()
	if err != nil {
		return rs, err
	}
	return rs, nil
}

// resource_aws_r53_wait checks the status of a change
func resource_aws_r53_wait(r53 *route53.Route53, ref string) (result interface{}, state string, err error) {
	status, err := r53.GetChange(ref)
	if err != nil {
		return nil, "UNKNOWN", err
	}
	return true, status, nil
}

func resource_aws_r53_zone_destroy(
	s *terraform.InstanceState,
	meta interface{}) error {
	p := meta.(*ResourceProvider)
	r53 := p.route53

	log.Printf("[DEBUG] Deleting Route53 hosted zone: %s (ID: %s)",
		s.Attributes["name"], s.Attributes["zone_id"])
	_, err := r53.DeleteHostedZone(s.Attributes["zone_id"])
	if err != nil {
		return err
	}
	return nil
}

func resource_aws_r53_zone_refresh(
	s *terraform.InstanceState,
	meta interface{}) (*terraform.InstanceState, error) {
	p := meta.(*ResourceProvider)
	r53 := p.route53

	_, err := r53.GetHostedZone(s.Attributes["zone_id"])
	if err != nil {
		// Handle a deleted zone
		if strings.Contains(err.Error(), "404") {
			s.ID = ""
			return s, nil
		}
		return s, err
	}
	return s, nil
}

func resource_aws_r53_zone_diff(
	s *terraform.InstanceState,
	c *terraform.ResourceConfig,
	meta interface{}) (*terraform.InstanceDiff, error) {

	b := &diff.ResourceBuilder{
		Attrs: map[string]diff.AttrType{
			"name": diff.AttrTypeCreate,
		},

		ComputedAttrs: []string{
			"zone_id",
		},
	}
	return b.Diff(s, c)
}
