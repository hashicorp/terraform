package aws

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsDefaultSecurityGroup() *schema.Resource {
	// reuse aws_security_group_rule schema, and methods for READ, UPDATE
	dsg := resourceAwsSecurityGroup()
	dsg.Create = resourceAwsDefaultSecurityGroupCreate
	dsg.Delete = resourceAwsDefaultSecurityGroupDelete

	// Descriptions cannot be updated
	delete(dsg.Schema, "description")

	// name is a computed value for Default Security Groups and cannot be changed
	delete(dsg.Schema, "name_prefix")
	dsg.Schema["name"] = &schema.Schema{
		Type:     schema.TypeString,
		Computed: true,
	}

	// We want explicit management of Rules here, so we do not allow them to be
	// computed. Instead, an empty config will enforce just that; removal of the
	// rules
	dsg.Schema["ingress"].Computed = false
	dsg.Schema["egress"].Computed = false
	return dsg
}

func resourceAwsDefaultSecurityGroupCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn
	securityGroupOpts := &ec2.DescribeSecurityGroupsInput{
		Filters: []*ec2.Filter{
			&ec2.Filter{
				Name:   aws.String("group-name"),
				Values: []*string{aws.String("default")},
			},
		},
	}

	var vpcId string
	if v, ok := d.GetOk("vpc_id"); ok {
		vpcId = v.(string)
		securityGroupOpts.Filters = append(securityGroupOpts.Filters, &ec2.Filter{
			Name:   aws.String("vpc-id"),
			Values: []*string{aws.String(vpcId)},
		})
	}

	var err error
	log.Printf("[DEBUG] Commandeer Default Security Group: %s", securityGroupOpts)
	resp, err := conn.DescribeSecurityGroups(securityGroupOpts)
	if err != nil {
		return fmt.Errorf("Error creating Default Security Group: %s", err)
	}

	var g *ec2.SecurityGroup
	if vpcId != "" {
		// if vpcId contains a value, then we expect just a single Security Group
		// returned, as default is a protected name for each VPC, and for each
		// Region on EC2 Classic
		if len(resp.SecurityGroups) != 1 {
			return fmt.Errorf("[ERR] Error finding default security group; found (%d) groups: %s", len(resp.SecurityGroups), resp)
		}
		g = resp.SecurityGroups[0]
	} else {
		// we need to filter through any returned security groups for the group
		// named "default", and does not belong to a VPC
		for _, sg := range resp.SecurityGroups {
			if sg.VpcId == nil && *sg.GroupName == "default" {
				g = sg
			}
		}
	}

	if g == nil {
		return fmt.Errorf("[ERR] Error finding default security group: no matching group found")
	}

	d.SetId(*g.GroupId)

	log.Printf("[INFO] Default Security Group ID: %s", d.Id())

	if err := setTags(conn, d); err != nil {
		return err
	}

	if err := revokeDefaultSecurityGroupRules(meta, g); err != nil {
		return errwrap.Wrapf("{{err}}", err)
	}

	return resourceAwsSecurityGroupUpdate(d, meta)
}

func resourceAwsDefaultSecurityGroupDelete(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[WARN] Cannot destroy Default Security Group. Terraform will remove this resource from the state file, however resources may remain.")
	d.SetId("")
	return nil
}

func revokeDefaultSecurityGroupRules(meta interface{}, g *ec2.SecurityGroup) error {
	conn := meta.(*AWSClient).ec2conn

	log.Printf("[WARN] Removing all ingress and egress rules found on Default Security Group (%s)", *g.GroupId)
	if len(g.IpPermissionsEgress) > 0 {
		req := &ec2.RevokeSecurityGroupEgressInput{
			GroupId:       g.GroupId,
			IpPermissions: g.IpPermissionsEgress,
		}

		log.Printf("[DEBUG] Revoking default egress rules for Default Security Group for %s", *g.GroupId)
		if _, err := conn.RevokeSecurityGroupEgress(req); err != nil {
			return fmt.Errorf(
				"Error revoking default egress rules for Default Security Group (%s): %s",
				*g.GroupId, err)
		}
	}
	if len(g.IpPermissions) > 0 {
		// a limitation in EC2 Classic is that a call to RevokeSecurityGroupIngress
		// cannot contain both the GroupName and the GroupId
		for _, p := range g.IpPermissions {
			for _, uigp := range p.UserIdGroupPairs {
				if uigp.GroupId != nil && uigp.GroupName != nil {
					uigp.GroupName = nil
				}
			}
		}
		req := &ec2.RevokeSecurityGroupIngressInput{
			GroupId:       g.GroupId,
			IpPermissions: g.IpPermissions,
		}

		log.Printf("[DEBUG] Revoking default ingress rules for Default Security Group for (%s): %s", *g.GroupId, req)
		if _, err := conn.RevokeSecurityGroupIngress(req); err != nil {
			return fmt.Errorf(
				"Error revoking default ingress rules for Default Security Group (%s): %s",
				*g.GroupId, err)
		}
	}

	return nil
}
