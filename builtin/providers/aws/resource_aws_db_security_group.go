package aws

import (
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/terraform/flatmap"
	"github.com/hashicorp/terraform/helper/config"
	"github.com/hashicorp/terraform/helper/diff"
	"github.com/hashicorp/terraform/helper/multierror"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/goamz/rds"
)

func resource_aws_db_security_group_create(
	s *terraform.InstanceState,
	d *terraform.InstanceDiff,
	meta interface{}) (*terraform.InstanceState, error) {
	p := meta.(*ResourceProvider)
	conn := p.rdsconn

	// Merge the diff into the state so that we have all the attributes
	// properly.
	rs := s.MergeDiff(d)

	var err error
	var errs []error

	opts := rds.CreateDBSecurityGroup{
		DBSecurityGroupName:        rs.Attributes["name"],
		DBSecurityGroupDescription: rs.Attributes["description"],
	}

	log.Printf("[DEBUG] DB Security Group create configuration: %#v", opts)
	_, err = conn.CreateDBSecurityGroup(&opts)
	if err != nil {
		return nil, fmt.Errorf("Error creating DB Security Group: %s", err)
	}

	rs.ID = rs.Attributes["name"]

	log.Printf("[INFO] DB Security Group ID: %s", rs.ID)

	v, err := resource_aws_db_security_group_retrieve(rs.ID, conn)
	if err != nil {
		return rs, err
	}

	if _, ok := rs.Attributes["ingress.#"]; ok {
		ingresses := flatmap.Expand(
			rs.Attributes, "ingress").([]interface{})

		for _, ing := range ingresses {
			err = authorize_ingress_rule(ing, v.Name, conn)

			if err != nil {
				errs = append(errs, err)
			}
		}

		if len(errs) > 0 {
			return rs, &multierror.Error{Errors: errs}
		}
	}

	log.Println(
		"[INFO] Waiting for Ingress Authorizations to be authorized")

	stateConf := &resource.StateChangeConf{
		Pending: []string{"authorizing"},
		Target:  "authorized",
		Refresh: DBSecurityGroupStateRefreshFunc(rs.ID, conn),
		Timeout: 10 * time.Minute,
	}

	// Wait, catching any errors
	_, err = stateConf.WaitForState()
	if err != nil {
		return rs, err
	}

	return resource_aws_db_security_group_update_state(rs, v)
}

func resource_aws_db_security_group_update(
	s *terraform.InstanceState,
	d *terraform.InstanceDiff,
	meta interface{}) (*terraform.InstanceState, error) {
	panic("Cannot update DB security group")
}

func resource_aws_db_security_group_destroy(
	s *terraform.InstanceState,
	meta interface{}) error {
	p := meta.(*ResourceProvider)
	conn := p.rdsconn

	log.Printf("[DEBUG] DB Security Group destroy: %v", s.ID)

	opts := rds.DeleteDBSecurityGroup{DBSecurityGroupName: s.ID}

	log.Printf("[DEBUG] DB Security Group destroy configuration: %v", opts)
	_, err := conn.DeleteDBSecurityGroup(&opts)

	if err != nil {
		newerr, ok := err.(*rds.Error)
		if ok && newerr.Code == "InvalidDBSecurityGroup.NotFound" {
			return nil
		}
		return err
	}

	return nil
}

func resource_aws_db_security_group_refresh(
	s *terraform.InstanceState,
	meta interface{}) (*terraform.InstanceState, error) {
	p := meta.(*ResourceProvider)
	conn := p.rdsconn

	v, err := resource_aws_db_security_group_retrieve(s.ID, conn)

	if err != nil {
		return s, err
	}

	return resource_aws_db_security_group_update_state(s, v)
}

func resource_aws_db_security_group_diff(
	s *terraform.InstanceState,
	c *terraform.ResourceConfig,
	meta interface{}) (*terraform.InstanceDiff, error) {

	b := &diff.ResourceBuilder{
		Attrs: map[string]diff.AttrType{
			"name":        diff.AttrTypeCreate,
			"description": diff.AttrTypeCreate,
			"ingress":     diff.AttrTypeCreate,
		},

		ComputedAttrs: []string{
			"ingress_cidr",
			"ingress_security_groups",
		},
	}

	return b.Diff(s, c)
}

func resource_aws_db_security_group_update_state(
	s *terraform.InstanceState,
	v *rds.DBSecurityGroup) (*terraform.InstanceState, error) {

	s.Attributes["name"] = v.Name
	s.Attributes["description"] = v.Description

	// Flatten our group values
	toFlatten := make(map[string]interface{})

	if len(v.EC2SecurityGroupOwnerIds) > 0 && v.EC2SecurityGroupOwnerIds[0] != "" {
		toFlatten["ingress_security_groups"] = v.EC2SecurityGroupOwnerIds
	}

	if len(v.CidrIps) > 0 && v.CidrIps[0] != "" {
		toFlatten["ingress_cidr"] = v.CidrIps
	}

	for k, v := range flatmap.Flatten(toFlatten) {
		s.Attributes[k] = v
	}

	return s, nil
}

func resource_aws_db_security_group_retrieve(id string, conn *rds.Rds) (*rds.DBSecurityGroup, error) {
	opts := rds.DescribeDBSecurityGroups{
		DBSecurityGroupName: id,
	}

	log.Printf("[DEBUG] DB Security Group describe configuration: %#v", opts)

	resp, err := conn.DescribeDBSecurityGroups(&opts)

	if err != nil {
		return nil, fmt.Errorf("Error retrieving DB Security Groups: %s", err)
	}

	if len(resp.DBSecurityGroups) != 1 ||
		resp.DBSecurityGroups[0].Name != id {
		if err != nil {
			return nil, fmt.Errorf("Unable to find DB Security Group: %#v", resp.DBSecurityGroups)
		}
	}

	v := resp.DBSecurityGroups[0]

	return &v, nil
}

// Authorizes the ingress rule on the db security group
func authorize_ingress_rule(ingress interface{}, dbSecurityGroupName string, conn *rds.Rds) error {
	ing := ingress.(map[string]interface{})

	opts := rds.AuthorizeDBSecurityGroupIngress{
		DBSecurityGroupName: dbSecurityGroupName,
	}

	if attr, ok := ing["cidr"].(string); ok && attr != "" {
		opts.Cidr = attr
	}

	if attr, ok := ing["security_group_name"].(string); ok && attr != "" {
		opts.EC2SecurityGroupName = attr
	}

	if attr, ok := ing["security_group_id"].(string); ok && attr != "" {
		opts.EC2SecurityGroupId = attr
	}

	if attr, ok := ing["security_group_owner_id"].(string); ok && attr != "" {
		opts.EC2SecurityGroupOwnerId = attr
	}

	log.Printf("[DEBUG] Authorize ingress rule configuration: %#v", opts)

	_, err := conn.AuthorizeDBSecurityGroupIngress(&opts)

	if err != nil {
		return fmt.Errorf("Error authorizing security group ingress: %s", err)
	}

	return nil
}

func resource_aws_db_security_group_validation() *config.Validator {
	return &config.Validator{
		Required: []string{
			"name",
			"description",
		},
		Optional: []string{
			"ingress.*",
			"ingress.*.cidr",
			"ingress.*.security_group_name",
			"ingress.*.security_group_id",
			"ingress.*.security_group_owner_id",
		},
	}
}

func DBSecurityGroupStateRefreshFunc(id string, conn *rds.Rds) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		v, err := resource_aws_db_security_group_retrieve(id, conn)

		if err != nil {
			log.Printf("Error on retrieving DB Security Group when waiting: %s", err)
			return nil, "", err
		}

		statuses := append(v.EC2SecurityGroupStatuses, v.CidrStatuses...)

		for _, stat := range statuses {
			// Not done
			if stat != "authorized" {
				return nil, "authorizing", nil
			}
		}

		return v, "authorized", nil
	}
}
