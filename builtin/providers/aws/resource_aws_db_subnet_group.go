package aws

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/hashicorp/terraform/flatmap"
	"github.com/hashicorp/terraform/helper/config"
	"github.com/hashicorp/terraform/helper/diff"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/goamz/rds"
)

func resource_aws_db_subnet_group_create(
	s *terraform.ResourceState,
	d *terraform.ResourceDiff,
	meta interface{}) (*terraform.ResourceState, error) {
	p := meta.(*ResourceProvider)
	conn := p.rdsconn

	// Merge the diff into the state so that we have all the attributes
	// properly.
	rs := s.MergeDiff(d)

	var err error

	opts := rds.CreateDBSubnetGroup{
		DBSubnetGroupName:        rs.Attributes["name"],
		DBSubnetGroupDescription: rs.Attributes["description"],
		SubnetIds:                expandStringList(flatmap.Expand(
									rs.Attributes, "subnet_ids").([]interface{})),
	}

	log.Printf("[DEBUG] DB Subnet Group create configuration: %#v", opts)
	_, err = conn.CreateDBSubnetGroup(&opts)
	if err != nil {
		return nil, fmt.Errorf("Error creating DB Subnet Group: %s", err)
	}

	rs.ID = rs.Attributes["name"]

	log.Printf("[INFO] DB Subnet Group ID: %s", rs.ID)

	log.Println(
		"[INFO] Waiting for DB Subnet Group creation to be complete")

	stateConf := &resource.StateChangeConf{
		// TODO are there any other states?
		Pending: []string{},
		Target:  "Complete",
		Refresh: DBSubnetGroupStateRefreshFunc(rs.ID, conn),
		Timeout: 10 * time.Minute,
	}

	// Wait, catching any errors
	_, err = stateConf.WaitForState()
	if err != nil {
		return rs, err
	}

	v, err := resource_aws_db_subnet_group_retrieve(rs.ID, conn)
	if err != nil {
		return rs, err
	}

	return resource_aws_db_subnet_group_update_state(rs, v)
}

func resource_aws_db_subnet_group_update(
	s *terraform.ResourceState,
	d *terraform.ResourceDiff,
	meta interface{}) (*terraform.ResourceState, error) {

	panic("Cannot update DB Subnet Group")

	return nil, nil
}

func resource_aws_db_subnet_group_destroy(
	s *terraform.ResourceState,
	meta interface{}) error {
	p := meta.(*ResourceProvider)
	conn := p.rdsconn

	log.Printf("[DEBUG] DB Subnet Group destroy: %v", s.ID)

	opts := rds.DeleteDBSubnetGroup{DBSubnetGroupName: s.ID}

	log.Printf("[DEBUG] DB Subnet Group destroy configuration: %v", opts)
	_, err := conn.DeleteDBSubnetGroup(&opts)

	if err != nil {
		newerr, ok := err.(*rds.Error)
		if ok && newerr.Code == "DBSubnetGroupNotFoundFault" {
			return nil
		}
		return err
	}

	return nil
}

func resource_aws_db_subnet_group_refresh(
	s *terraform.ResourceState,
	meta interface{}) (*terraform.ResourceState, error) {
	p := meta.(*ResourceProvider)
	conn := p.rdsconn

	v, err := resource_aws_db_subnet_group_retrieve(s.ID, conn)

	if err != nil || v == nil {
		return s, err
	}

	return resource_aws_db_subnet_group_update_state(s, v)
}

func resource_aws_db_subnet_group_diff(
	s *terraform.ResourceState,
	c *terraform.ResourceConfig,
	meta interface{}) (*terraform.ResourceDiff, error) {

	b := &diff.ResourceBuilder{
		Attrs: map[string]diff.AttrType{
			"name":        diff.AttrTypeCreate,
			"description": diff.AttrTypeCreate,
			"subnet_ids":  diff.AttrTypeCreate,
		},
	}

	return b.Diff(s, c)
}

func resource_aws_db_subnet_group_update_state(
	s *terraform.ResourceState,
	v *rds.DBSubnetGroup) (*terraform.ResourceState, error) {

	s.Attributes["name"] = v.Name
	s.Attributes["description"] = v.Description

	// Flatten our group values
	toFlatten := make(map[string]interface{})

	if len(v.SubnetIds) > 0 && v.SubnetIds[0] != "" {
		toFlatten["subnet_ids"] = v.SubnetIds
	}

	for k, v := range flatmap.Flatten(toFlatten) {
		s.Attributes[k] = v
	}

	return s, nil
}

func resource_aws_db_subnet_group_retrieve(name string, conn *rds.Rds) (*rds.DBSubnetGroup, error) {
	opts := rds.DescribeDBSubnetGroups{
		DBSubnetGroupName: name,
	}

	log.Printf("[DEBUG] DB Subnet Group describe configuration: %#v", opts)

	resp, err := conn.DescribeDBSubnetGroups(&opts)

	if err != nil {
		if strings.Contains(err.Error(), "DBSubnetGroupNotFound") {
			return nil, nil
		}
		return nil, fmt.Errorf("Error retrieving DB Subnet Groups: %s", err)
	}

	if len(resp.DBSubnetGroups) != 1 ||
		resp.DBSubnetGroups[0].Name != name {
		if err != nil {
			return nil, nil
		}
	}

	v := resp.DBSubnetGroups[0]

	return &v, nil
}

func resource_aws_db_subnet_group_validation() *config.Validator {
	return &config.Validator{
		Required: []string{
			"name",
			"description",
			"subnet_ids.*",
		},
	}
}

func DBSubnetGroupStateRefreshFunc(name string, conn *rds.Rds) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		v, err := resource_aws_db_subnet_group_retrieve(name, conn)

		if err != nil {
			log.Printf("Error on retrieving DB Subnet Group when waiting: %s", err)
			return nil, "", err
		}

		return v, v.Status, nil
	}
}
