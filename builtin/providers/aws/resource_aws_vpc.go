package aws

import (
	"fmt"
	"log"
	"sort"
	"time"

	"github.com/hashicorp/terraform/flatmap"
	"github.com/hashicorp/terraform/helper/diff"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/goamz/ec2"
)

func resource_aws_vpc_create(
	s *terraform.ResourceState,
	d *terraform.ResourceDiff,
	meta interface{}) (*terraform.ResourceState, error) {
	p := meta.(*ResourceProvider)
	ec2conn := p.ec2conn

	// Merge the diff so that we have all the proper attributes
	s = s.MergeDiff(d)

	// Create the VPC
	createOpts := &ec2.CreateVpc{
		CidrBlock: s.Attributes["cidr_block"],
	}
	log.Printf("[DEBUG] VPC create config: %#v", createOpts)
	vpcResp, err := ec2conn.CreateVpc(createOpts)
	if err != nil {
		return nil, fmt.Errorf("Error creating VPC: %s", err)
	}

	// Get the ID and store it
	vpc := &vpcResp.VPC
	log.Printf("[INFO] VPC ID: %s", vpc.VpcId)
	s.ID = vpc.VpcId

	// Wait for the VPC to become available
	log.Printf(
		"[DEBUG] Waiting for VPC (%s) to become available",
		s.ID)
	stateConf := &resource.StateChangeConf{
		Pending: []string{"pending"},
		Target:  "available",
		Refresh: VPCStateRefreshFunc(ec2conn, s.ID),
		Timeout: 10 * time.Minute,
	}
	vpcRaw, err := stateConf.WaitForState()
	if err != nil {
		return s, fmt.Errorf(
			"Error waiting for VPC (%s) to become available: %s",
			s.ID, err)
	}

	tags := resource_aws_build_tags(s.Attributes, "tag")
	if err := resource_aws_sync_tags(ec2conn, s.ID, []ec2.Tag{}, tags); err != nil {
		return nil, err
	}

	// Update our attributes and return
	return resource_aws_vpc_update_state(s, vpcRaw.(*ec2.VPC), tags)
}

func resource_aws_vpc_update(
	s *terraform.ResourceState,
	d *terraform.ResourceDiff,
	meta interface{}) (*terraform.ResourceState, error) {
	p := meta.(*ResourceProvider)
	ec2conn := p.ec2conn
	rs := s.MergeDiff(d)

	oldTags := resource_aws_build_tags(s.Attributes, "tag")
	newTags := resource_aws_build_tags(rs.Attributes, "tag")

	if err := resource_aws_sync_tags(ec2conn, s.ID, oldTags, newTags); err != nil {
		return nil, err
	}

	return rs, nil
}

func resource_aws_vpc_destroy(
	s *terraform.ResourceState,
	meta interface{}) error {
	p := meta.(*ResourceProvider)
	ec2conn := p.ec2conn

	log.Printf("[INFO] Deleting VPC: %s", s.ID)
	if _, err := ec2conn.DeleteVpc(s.ID); err != nil {
		ec2err, ok := err.(*ec2.Error)
		if ok && ec2err.Code == "InvalidVpcID.NotFound" {
			return nil
		}

		return fmt.Errorf("Error deleting VPC: %s", err)
	}

	return nil
}

func resource_aws_vpc_refresh(
	s *terraform.ResourceState,
	meta interface{}) (*terraform.ResourceState, error) {
	p := meta.(*ResourceProvider)
	ec2conn := p.ec2conn

	vpcRaw, _, err := VPCStateRefreshFunc(ec2conn, s.ID)()
	if err != nil {
		return s, err
	}
	if vpcRaw == nil {
		return nil, nil
	}

	filter := ec2.NewFilter()
	filter.Add("resource-id", s.ID)
	tagsResp, err := ec2conn.Tags(filter)
	if err != nil {
		return nil, err
	}

	tags := make([]ec2.Tag, len(tagsResp.Tags))
	for i, v := range tagsResp.Tags {
		tags[i] = v.Tag
	}

	sort.Stable(sortableTags(tags))

	return resource_aws_vpc_update_state(s, vpcRaw.(*ec2.VPC), tags)
}

func resource_aws_vpc_diff(
	s *terraform.ResourceState,
	c *terraform.ResourceConfig,
	meta interface{}) (*terraform.ResourceDiff, error) {
	b := &diff.ResourceBuilder{
		Attrs: map[string]diff.AttrType{
			"cidr_block": diff.AttrTypeCreate,
			"tag":        diff.AttrTypeUpdate,
		},
	}

	return b.Diff(s, c)
}

func resource_aws_vpc_update_state(
	s *terraform.ResourceState,
	vpc *ec2.VPC,
	tags []ec2.Tag) (*terraform.ResourceState, error) {
	s.Attributes["cidr_block"] = vpc.CidrBlock

	toFlatten := make([]map[string]string, 0)
	for _, tag := range tags {
		toFlatten = append(toFlatten, map[string]string{
			"key":   tag.Key,
			"value": tag.Value,
		})
	}
	flatmap.Map(s.Attributes).Merge(flatmap.Flatten(map[string]interface{}{
		"tag": toFlatten,
	}))

	return s, nil
}

// VPCStateRefreshFunc returns a resource.StateRefreshFunc that is used to watch
// a VPC.
func VPCStateRefreshFunc(conn *ec2.EC2, id string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		resp, err := conn.DescribeVpcs([]string{id}, ec2.NewFilter())
		if err != nil {
			if ec2err, ok := err.(*ec2.Error); ok && ec2err.Code == "InvalidVpcID.NotFound" {
				resp = nil
			} else {
				log.Printf("Error on VPCStateRefresh: %s", err)
				return nil, "", err
			}
		}

		if resp == nil {
			// Sometimes AWS just has consistency issues and doesn't see
			// our instance yet. Return an empty state.
			return nil, "", nil
		}

		vpc := &resp.VPCs[0]
		return vpc, vpc.State, nil
	}
}
