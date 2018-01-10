package aws

import (
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsVpc() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsVpcCreate,
		Read:   resourceAwsVpcRead,
		Update: resourceAwsVpcUpdate,
		Delete: resourceAwsVpcDelete,
		Importer: &schema.ResourceImporter{
			State: resourceAwsVpcInstanceImport,
		},

		SchemaVersion: 1,
		MigrateState:  resourceAwsVpcMigrateState,

		Schema: map[string]*schema.Schema{
			"cidr_block": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validateCIDRNetworkAddress,
			},

			"instance_tenancy": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Computed: true,
			},

			"enable_dns_hostnames": {
				Type:     schema.TypeBool,
				Optional: true,
				Computed: true,
			},

			"enable_dns_support": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},

			"enable_classiclink": {
				Type:     schema.TypeBool,
				Optional: true,
				Computed: true,
			},

			"enable_classiclink_dns_support": {
				Type:     schema.TypeBool,
				Optional: true,
				Computed: true,
			},

			"assign_generated_ipv6_cidr_block": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},

			"main_route_table_id": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"default_network_acl_id": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"dhcp_options_id": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"default_security_group_id": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"default_route_table_id": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"ipv6_association_id": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"ipv6_cidr_block": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"tags": tagsSchema(),
		},
	}
}

func resourceAwsVpcCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn
	instance_tenancy := "default"
	if v, ok := d.GetOk("instance_tenancy"); ok {
		instance_tenancy = v.(string)
	}

	// Create the VPC
	createOpts := &ec2.CreateVpcInput{
		CidrBlock:                   aws.String(d.Get("cidr_block").(string)),
		InstanceTenancy:             aws.String(instance_tenancy),
		AmazonProvidedIpv6CidrBlock: aws.Bool(d.Get("assign_generated_ipv6_cidr_block").(bool)),
	}

	log.Printf("[DEBUG] VPC create config: %#v", *createOpts)
	vpcResp, err := conn.CreateVpc(createOpts)
	if err != nil {
		return fmt.Errorf("Error creating VPC: %s", err)
	}

	// Get the ID and store it
	vpc := vpcResp.Vpc
	d.SetId(*vpc.VpcId)
	log.Printf("[INFO] VPC ID: %s", d.Id())

	// Set partial mode and say that we setup the cidr block
	d.Partial(true)
	d.SetPartial("cidr_block")

	// Wait for the VPC to become available
	log.Printf(
		"[DEBUG] Waiting for VPC (%s) to become available",
		d.Id())
	stateConf := &resource.StateChangeConf{
		Pending: []string{"pending"},
		Target:  []string{"available"},
		Refresh: VPCStateRefreshFunc(conn, d.Id()),
		Timeout: 10 * time.Minute,
	}
	if _, err := stateConf.WaitForState(); err != nil {
		return fmt.Errorf(
			"Error waiting for VPC (%s) to become available: %s",
			d.Id(), err)
	}

	// Update our attributes and return
	return resourceAwsVpcUpdate(d, meta)
}

func resourceAwsVpcRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	// Refresh the VPC state
	vpcRaw, _, err := VPCStateRefreshFunc(conn, d.Id())()
	if err != nil {
		return err
	}
	if vpcRaw == nil {
		d.SetId("")
		return nil
	}

	// VPC stuff
	vpc := vpcRaw.(*ec2.Vpc)
	vpcid := d.Id()
	d.Set("cidr_block", vpc.CidrBlock)
	d.Set("dhcp_options_id", vpc.DhcpOptionsId)
	d.Set("instance_tenancy", vpc.InstanceTenancy)

	// Tags
	d.Set("tags", tagsToMap(vpc.Tags))

	for _, a := range vpc.Ipv6CidrBlockAssociationSet {
		if *a.Ipv6CidrBlockState.State == "associated" { //we can only ever have 1 IPv6 block associated at once
			d.Set("assign_generated_ipv6_cidr_block", true)
			d.Set("ipv6_association_id", a.AssociationId)
			d.Set("ipv6_cidr_block", a.Ipv6CidrBlock)
		} else {
			d.Set("assign_generated_ipv6_cidr_block", false)
			d.Set("ipv6_association_id", "") // we blank these out to remove old entries
			d.Set("ipv6_cidr_block", "")
		}
	}

	resp, err := awsVpcDescribeVpcAttribute("enableDnsSupport", vpcid, conn)
	if err != nil {
		return err
	}
	d.Set("enable_dns_support", resp.EnableDnsSupport.Value)

	resp, err = awsVpcDescribeVpcAttribute("enableDnsHostnames", vpcid, conn)
	if err != nil {
		return err
	}
	d.Set("enable_dns_hostnames", resp.EnableDnsHostnames.Value)

	describeClassiclinkOpts := &ec2.DescribeVpcClassicLinkInput{
		VpcIds: []*string{&vpcid},
	}

	// Classic Link is only available in regions that support EC2 Classic
	respClassiclink, err := conn.DescribeVpcClassicLink(describeClassiclinkOpts)
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok && awsErr.Code() == "UnsupportedOperation" {
			log.Printf("[WARN] VPC Classic Link is not supported in this region")
		} else {
			return err
		}
	} else {
		classiclink_enabled := false
		for _, v := range respClassiclink.Vpcs {
			if *v.VpcId == vpcid {
				if v.ClassicLinkEnabled != nil {
					classiclink_enabled = *v.ClassicLinkEnabled
				}
				break
			}
		}
		d.Set("enable_classiclink", classiclink_enabled)
	}

	describeClassiclinkDnsOpts := &ec2.DescribeVpcClassicLinkDnsSupportInput{
		VpcIds: []*string{&vpcid},
	}

	respClassiclinkDnsSupport, err := conn.DescribeVpcClassicLinkDnsSupport(describeClassiclinkDnsOpts)
	if err != nil {
		if isAWSErr(err, "UnsupportedOperation", "The functionality you requested is not available in this region") ||
			isAWSErr(err, "AuthFailure", "This request has been administratively disabled") {
			log.Printf("[WARN] VPC Classic Link DNS Support is not supported in this region")
		} else {
			return err
		}
	} else {
		classiclinkdns_enabled := false
		for _, v := range respClassiclinkDnsSupport.Vpcs {
			if *v.VpcId == vpcid {
				if v.ClassicLinkDnsSupported != nil {
					classiclinkdns_enabled = *v.ClassicLinkDnsSupported
				}
				break
			}
		}
		d.Set("enable_classiclink_dns_support", classiclinkdns_enabled)
	}

	// Get the main routing table for this VPC
	// Really Ugly need to make this better - rmenn
	filter1 := &ec2.Filter{
		Name:   aws.String("association.main"),
		Values: []*string{aws.String("true")},
	}
	filter2 := &ec2.Filter{
		Name:   aws.String("vpc-id"),
		Values: []*string{aws.String(d.Id())},
	}
	describeRouteOpts := &ec2.DescribeRouteTablesInput{
		Filters: []*ec2.Filter{filter1, filter2},
	}
	routeResp, err := conn.DescribeRouteTables(describeRouteOpts)
	if err != nil {
		return err
	}
	if v := routeResp.RouteTables; len(v) > 0 {
		d.Set("main_route_table_id", *v[0].RouteTableId)
	}

	if err := resourceAwsVpcSetDefaultNetworkAcl(conn, d); err != nil {
		log.Printf("[WARN] Unable to set Default Network ACL: %s", err)
	}
	if err := resourceAwsVpcSetDefaultSecurityGroup(conn, d); err != nil {
		log.Printf("[WARN] Unable to set Default Security Group: %s", err)
	}
	if err := resourceAwsVpcSetDefaultRouteTable(conn, d); err != nil {
		log.Printf("[WARN] Unable to set Default Route Table: %s", err)
	}

	return nil
}

func resourceAwsVpcUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	// Turn on partial mode
	d.Partial(true)
	vpcid := d.Id()
	if d.HasChange("enable_dns_hostnames") {
		val := d.Get("enable_dns_hostnames").(bool)
		modifyOpts := &ec2.ModifyVpcAttributeInput{
			VpcId: &vpcid,
			EnableDnsHostnames: &ec2.AttributeBooleanValue{
				Value: &val,
			},
		}

		log.Printf(
			"[INFO] Modifying enable_dns_hostnames vpc attribute for %s: %s",
			d.Id(), modifyOpts)
		if _, err := conn.ModifyVpcAttribute(modifyOpts); err != nil {
			return err
		}

		d.SetPartial("enable_dns_hostnames")
	}

	_, hasEnableDnsSupportOption := d.GetOk("enable_dns_support")

	if !hasEnableDnsSupportOption || d.HasChange("enable_dns_support") {
		val := d.Get("enable_dns_support").(bool)
		modifyOpts := &ec2.ModifyVpcAttributeInput{
			VpcId: &vpcid,
			EnableDnsSupport: &ec2.AttributeBooleanValue{
				Value: &val,
			},
		}

		log.Printf(
			"[INFO] Modifying enable_dns_support vpc attribute for %s: %s",
			d.Id(), modifyOpts)
		if _, err := conn.ModifyVpcAttribute(modifyOpts); err != nil {
			return err
		}

		d.SetPartial("enable_dns_support")
	}

	if d.HasChange("enable_classiclink") {
		val := d.Get("enable_classiclink").(bool)
		if val {
			modifyOpts := &ec2.EnableVpcClassicLinkInput{
				VpcId: &vpcid,
			}
			log.Printf(
				"[INFO] Modifying enable_classiclink vpc attribute for %s: %#v",
				d.Id(), modifyOpts)
			if _, err := conn.EnableVpcClassicLink(modifyOpts); err != nil {
				return err
			}
		} else {
			modifyOpts := &ec2.DisableVpcClassicLinkInput{
				VpcId: &vpcid,
			}
			log.Printf(
				"[INFO] Modifying enable_classiclink vpc attribute for %s: %#v",
				d.Id(), modifyOpts)
			if _, err := conn.DisableVpcClassicLink(modifyOpts); err != nil {
				return err
			}
		}

		d.SetPartial("enable_classiclink")
	}

	if d.HasChange("enable_classiclink_dns_support") {
		val := d.Get("enable_classiclink_dns_support").(bool)
		if val {
			modifyOpts := &ec2.EnableVpcClassicLinkDnsSupportInput{
				VpcId: &vpcid,
			}
			log.Printf(
				"[INFO] Modifying enable_classiclink_dns_support vpc attribute for %s: %#v",
				d.Id(), modifyOpts)
			if _, err := conn.EnableVpcClassicLinkDnsSupport(modifyOpts); err != nil {
				return err
			}
		} else {
			modifyOpts := &ec2.DisableVpcClassicLinkDnsSupportInput{
				VpcId: &vpcid,
			}
			log.Printf(
				"[INFO] Modifying enable_classiclink_dns_support vpc attribute for %s: %#v",
				d.Id(), modifyOpts)
			if _, err := conn.DisableVpcClassicLinkDnsSupport(modifyOpts); err != nil {
				return err
			}
		}

		d.SetPartial("enable_classiclink_dns_support")
	}

	if d.HasChange("assign_generated_ipv6_cidr_block") && !d.IsNewResource() {
		toAssign := d.Get("assign_generated_ipv6_cidr_block").(bool)

		log.Printf("[INFO] Modifying assign_generated_ipv6_cidr_block to %#v", toAssign)

		if toAssign {
			modifyOpts := &ec2.AssociateVpcCidrBlockInput{
				VpcId: &vpcid,
				AmazonProvidedIpv6CidrBlock: aws.Bool(toAssign),
			}
			log.Printf("[INFO] Enabling assign_generated_ipv6_cidr_block vpc attribute for %s: %#v",
				d.Id(), modifyOpts)
			resp, err := conn.AssociateVpcCidrBlock(modifyOpts)
			if err != nil {
				return err
			}

			// Wait for the CIDR to become available
			log.Printf(
				"[DEBUG] Waiting for IPv6 CIDR (%s) to become associated",
				d.Id())
			stateConf := &resource.StateChangeConf{
				Pending: []string{"associating", "disassociated"},
				Target:  []string{"associated"},
				Refresh: Ipv6CidrStateRefreshFunc(conn, d.Id(), *resp.Ipv6CidrBlockAssociation.AssociationId),
				Timeout: 1 * time.Minute,
			}
			if _, err := stateConf.WaitForState(); err != nil {
				return fmt.Errorf(
					"Error waiting for IPv6 CIDR (%s) to become associated: %s",
					d.Id(), err)
			}
		} else {
			modifyOpts := &ec2.DisassociateVpcCidrBlockInput{
				AssociationId: aws.String(d.Get("ipv6_association_id").(string)),
			}
			log.Printf("[INFO] Disabling assign_generated_ipv6_cidr_block vpc attribute for %s: %#v",
				d.Id(), modifyOpts)
			if _, err := conn.DisassociateVpcCidrBlock(modifyOpts); err != nil {
				return err
			}

			// Wait for the CIDR to become available
			log.Printf(
				"[DEBUG] Waiting for IPv6 CIDR (%s) to become disassociated",
				d.Id())
			stateConf := &resource.StateChangeConf{
				Pending: []string{"disassociating", "associated"},
				Target:  []string{"disassociated"},
				Refresh: Ipv6CidrStateRefreshFunc(conn, d.Id(), d.Get("ipv6_association_id").(string)),
				Timeout: 1 * time.Minute,
			}
			if _, err := stateConf.WaitForState(); err != nil {
				return fmt.Errorf(
					"Error waiting for IPv6 CIDR (%s) to become disassociated: %s",
					d.Id(), err)
			}
		}

		d.SetPartial("assign_generated_ipv6_cidr_block")
	}

	if err := setTags(conn, d); err != nil {
		return err
	} else {
		d.SetPartial("tags")
	}

	d.Partial(false)
	return resourceAwsVpcRead(d, meta)
}

func resourceAwsVpcDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn
	vpcID := d.Id()
	deleteVpcOpts := &ec2.DeleteVpcInput{
		VpcId: &vpcID,
	}
	log.Printf("[INFO] Deleting VPC: %s", d.Id())

	return resource.Retry(5*time.Minute, func() *resource.RetryError {
		_, err := conn.DeleteVpc(deleteVpcOpts)
		if err == nil {
			return nil
		}

		ec2err, ok := err.(awserr.Error)
		if !ok {
			return resource.NonRetryableError(err)
		}

		switch ec2err.Code() {
		case "InvalidVpcID.NotFound":
			return nil
		case "DependencyViolation":
			return resource.RetryableError(err)
		}

		return resource.NonRetryableError(fmt.Errorf("Error deleting VPC: %s", err))
	})
}

// VPCStateRefreshFunc returns a resource.StateRefreshFunc that is used to watch
// a VPC.
func VPCStateRefreshFunc(conn *ec2.EC2, id string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		describeVpcOpts := &ec2.DescribeVpcsInput{
			VpcIds: []*string{aws.String(id)},
		}
		resp, err := conn.DescribeVpcs(describeVpcOpts)
		if err != nil {
			if ec2err, ok := err.(awserr.Error); ok && ec2err.Code() == "InvalidVpcID.NotFound" {
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

		vpc := resp.Vpcs[0]
		return vpc, *vpc.State, nil
	}
}

func Ipv6CidrStateRefreshFunc(conn *ec2.EC2, id string, associationId string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		describeVpcOpts := &ec2.DescribeVpcsInput{
			VpcIds: []*string{aws.String(id)},
		}
		resp, err := conn.DescribeVpcs(describeVpcOpts)
		if err != nil {
			if ec2err, ok := err.(awserr.Error); ok && ec2err.Code() == "InvalidVpcID.NotFound" {
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

		if resp.Vpcs[0].Ipv6CidrBlockAssociationSet == nil {
			return nil, "", nil
		}

		for _, association := range resp.Vpcs[0].Ipv6CidrBlockAssociationSet {
			if *association.AssociationId == associationId {
				return association, *association.Ipv6CidrBlockState.State, nil
			}
		}

		return nil, "", nil
	}
}

func resourceAwsVpcSetDefaultNetworkAcl(conn *ec2.EC2, d *schema.ResourceData) error {
	filter1 := &ec2.Filter{
		Name:   aws.String("default"),
		Values: []*string{aws.String("true")},
	}
	filter2 := &ec2.Filter{
		Name:   aws.String("vpc-id"),
		Values: []*string{aws.String(d.Id())},
	}
	describeNetworkACLOpts := &ec2.DescribeNetworkAclsInput{
		Filters: []*ec2.Filter{filter1, filter2},
	}
	networkAclResp, err := conn.DescribeNetworkAcls(describeNetworkACLOpts)

	if err != nil {
		return err
	}
	if v := networkAclResp.NetworkAcls; len(v) > 0 {
		d.Set("default_network_acl_id", v[0].NetworkAclId)
	}

	return nil
}

func resourceAwsVpcSetDefaultSecurityGroup(conn *ec2.EC2, d *schema.ResourceData) error {
	filter1 := &ec2.Filter{
		Name:   aws.String("group-name"),
		Values: []*string{aws.String("default")},
	}
	filter2 := &ec2.Filter{
		Name:   aws.String("vpc-id"),
		Values: []*string{aws.String(d.Id())},
	}
	describeSgOpts := &ec2.DescribeSecurityGroupsInput{
		Filters: []*ec2.Filter{filter1, filter2},
	}
	securityGroupResp, err := conn.DescribeSecurityGroups(describeSgOpts)

	if err != nil {
		return err
	}
	if v := securityGroupResp.SecurityGroups; len(v) > 0 {
		d.Set("default_security_group_id", v[0].GroupId)
	}

	return nil
}

func resourceAwsVpcSetDefaultRouteTable(conn *ec2.EC2, d *schema.ResourceData) error {
	filter1 := &ec2.Filter{
		Name:   aws.String("association.main"),
		Values: []*string{aws.String("true")},
	}
	filter2 := &ec2.Filter{
		Name:   aws.String("vpc-id"),
		Values: []*string{aws.String(d.Id())},
	}

	findOpts := &ec2.DescribeRouteTablesInput{
		Filters: []*ec2.Filter{filter1, filter2},
	}

	resp, err := conn.DescribeRouteTables(findOpts)
	if err != nil {
		return err
	}

	if len(resp.RouteTables) < 1 || resp.RouteTables[0] == nil {
		return fmt.Errorf("Default Route table not found")
	}

	// There Can Be Only 1 ... Default Route Table
	d.Set("default_route_table_id", resp.RouteTables[0].RouteTableId)

	return nil
}

func resourceAwsVpcInstanceImport(
	d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	d.Set("assign_generated_ipv6_cidr_block", false)
	return []*schema.ResourceData{d}, nil
}

func awsVpcDescribeVpcAttribute(attribute string, vpcId string, conn *ec2.EC2) (*ec2.DescribeVpcAttributeOutput, error) {
	describeAttrOpts := &ec2.DescribeVpcAttributeInput{
		Attribute: aws.String(attribute),
		VpcId:     aws.String(vpcId),
	}
	resp, err := conn.DescribeVpcAttribute(describeAttrOpts)
	if err != nil {
		return nil, err
	}

	return resp, nil
}
