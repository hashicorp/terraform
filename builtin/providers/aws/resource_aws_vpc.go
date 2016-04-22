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

		Schema: map[string]*schema.Schema{
			"cidr_block": &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validateCIDRNetworkAddress,
			},

			"instance_tenancy": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Computed: true,
			},

			"enable_dns_hostnames": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Computed: true,
			},

			"enable_dns_support": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Computed: true,
			},

			"enable_classiclink": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Computed: true,
			},

			"main_route_table_id": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"default_network_acl_id": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"dhcp_options_id": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"default_security_group_id": &schema.Schema{
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
		CidrBlock:       aws.String(d.Get("cidr_block").(string)),
		InstanceTenancy: aws.String(instance_tenancy),
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

	// Attributes
	attribute := "enableDnsSupport"
	DescribeAttrOpts := &ec2.DescribeVpcAttributeInput{
		Attribute: aws.String(attribute),
		VpcId:     aws.String(vpcid),
	}
	resp, err := conn.DescribeVpcAttribute(DescribeAttrOpts)
	if err != nil {
		return err
	}
	d.Set("enable_dns_support", *resp.EnableDnsSupport.Value)
	attribute = "enableDnsHostnames"
	DescribeAttrOpts = &ec2.DescribeVpcAttributeInput{
		Attribute: &attribute,
		VpcId:     &vpcid,
	}
	resp, err = conn.DescribeVpcAttribute(DescribeAttrOpts)
	if err != nil {
		return err
	}
	d.Set("enable_dns_hostnames", *resp.EnableDnsHostnames.Value)

	DescribeClassiclinkOpts := &ec2.DescribeVpcClassicLinkInput{
		VpcIds: []*string{&vpcid},
	}

	// Classic Link is only available in regions that support EC2 Classic
	respClassiclink, err := conn.DescribeVpcClassicLink(DescribeClassiclinkOpts)
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
	DescribeRouteOpts := &ec2.DescribeRouteTablesInput{
		Filters: []*ec2.Filter{filter1, filter2},
	}
	routeResp, err := conn.DescribeRouteTables(DescribeRouteOpts)
	if err != nil {
		return err
	}
	if v := routeResp.RouteTables; len(v) > 0 {
		d.Set("main_route_table_id", *v[0].RouteTableId)
	}

	resourceAwsVpcSetDefaultNetworkAcl(conn, d)
	resourceAwsVpcSetDefaultSecurityGroup(conn, d)

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
			"[INFO] Modifying enable_dns_support vpc attribute for %s: %#v",
			d.Id(), modifyOpts)
		if _, err := conn.ModifyVpcAttribute(modifyOpts); err != nil {
			return err
		}

		d.SetPartial("enable_dns_support")
	}

	if d.HasChange("enable_dns_support") {
		val := d.Get("enable_dns_support").(bool)
		modifyOpts := &ec2.ModifyVpcAttributeInput{
			VpcId: &vpcid,
			EnableDnsSupport: &ec2.AttributeBooleanValue{
				Value: &val,
			},
		}

		log.Printf(
			"[INFO] Modifying enable_dns_support vpc attribute for %s: %#v",
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
	DeleteVpcOpts := &ec2.DeleteVpcInput{
		VpcId: &vpcID,
	}
	log.Printf("[INFO] Deleting VPC: %s", d.Id())

	return resource.Retry(5*time.Minute, func() *resource.RetryError {
		_, err := conn.DeleteVpc(DeleteVpcOpts)
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
		DescribeVpcOpts := &ec2.DescribeVpcsInput{
			VpcIds: []*string{aws.String(id)},
		}
		resp, err := conn.DescribeVpcs(DescribeVpcOpts)
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

func resourceAwsVpcSetDefaultNetworkAcl(conn *ec2.EC2, d *schema.ResourceData) error {
	filter1 := &ec2.Filter{
		Name:   aws.String("default"),
		Values: []*string{aws.String("true")},
	}
	filter2 := &ec2.Filter{
		Name:   aws.String("vpc-id"),
		Values: []*string{aws.String(d.Id())},
	}
	DescribeNetworkACLOpts := &ec2.DescribeNetworkAclsInput{
		Filters: []*ec2.Filter{filter1, filter2},
	}
	networkAclResp, err := conn.DescribeNetworkAcls(DescribeNetworkACLOpts)

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
	DescribeSgOpts := &ec2.DescribeSecurityGroupsInput{
		Filters: []*ec2.Filter{filter1, filter2},
	}
	securityGroupResp, err := conn.DescribeSecurityGroups(DescribeSgOpts)

	if err != nil {
		return err
	}
	if v := securityGroupResp.SecurityGroups; len(v) > 0 {
		d.Set("default_security_group_id", v[0].GroupId)
	}

	return nil
}
