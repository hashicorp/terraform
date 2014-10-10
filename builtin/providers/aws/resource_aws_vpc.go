package aws

import (
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/mitchellh/goamz/ec2"
)

func resourceAwsVpc() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsVpcCreate,
		Read:   resourceAwsVpcRead,
		Update: resourceAwsVpcUpdate,
		Delete: resourceAwsVpcDelete,

		Schema: map[string]*schema.Schema{
			"cidr_block": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
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

			"main_route_table_id": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"tags": tagsSchema(),
		},
	}
}

func resourceAwsVpcCreate(d *schema.ResourceData, meta interface{}) error {
	p := meta.(*ResourceProvider)
	ec2conn := p.ec2conn

	// Create the VPC
	createOpts := &ec2.CreateVpc{
		CidrBlock: d.Get("cidr_block").(string),
	}
	log.Printf("[DEBUG] VPC create config: %#v", createOpts)
	vpcResp, err := ec2conn.CreateVpc(createOpts)
	if err != nil {
		return fmt.Errorf("Error creating VPC: %s", err)
	}

	// Get the ID and store it
	vpc := &vpcResp.VPC
	log.Printf("[INFO] VPC ID: %s", vpc.VpcId)
	d.SetId(vpc.VpcId)

	// Set partial mode and say that we setup the cidr block
	d.Partial(true)
	d.SetPartial("cidr_block")

	// Wait for the VPC to become available
	log.Printf(
		"[DEBUG] Waiting for VPC (%s) to become available",
		d.Id())
	stateConf := &resource.StateChangeConf{
		Pending: []string{"pending"},
		Target:  "available",
		Refresh: VPCStateRefreshFunc(ec2conn, d.Id()),
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

func resourceAwsVpcUpdate(d *schema.ResourceData, meta interface{}) error {
	p := meta.(*ResourceProvider)
	ec2conn := p.ec2conn

	// Turn on partial mode
	d.Partial(true)

	if d.HasChange("enable_dns_hostnames") {
		options := new(ec2.ModifyVpcAttribute)
		options.EnableDnsHostnames = d.Get("enable_dns_hostnames").(bool)
		options.SetEnableDnsHostnames = true

		log.Printf(
			"[INFO] Modifying enable_dns_hostnames vpc attribute for %s: %#v",
			d.Id(), options)
		if _, err := ec2conn.ModifyVpcAttribute(d.Id(), options); err != nil {
			return err
		}

		d.SetPartial("enable_dns_hostnames")
	}

	if d.HasChange("enable_dns_support") {
		options := new(ec2.ModifyVpcAttribute)
		options.EnableDnsSupport = d.Get("enable_dns_support").(bool)
		options.SetEnableDnsSupport = true

		log.Printf(
			"[INFO] Modifying enable_dns_support vpc attribute for %s: %#v",
			d.Id(), options)
		if _, err := ec2conn.ModifyVpcAttribute(d.Id(), options); err != nil {
			return err
		}

		d.SetPartial("enable_dns_support")
	}

	if err := setTags(ec2conn, d); err != nil {
		return err
	} else {
		d.SetPartial("tags")
	}

	d.Partial(false)
	return resourceAwsVpcRead(d, meta)
}

func resourceAwsVpcDelete(d *schema.ResourceData, meta interface{}) error {
	p := meta.(*ResourceProvider)
	ec2conn := p.ec2conn

	log.Printf("[INFO] Deleting VPC: %s", d.Id())
	if _, err := ec2conn.DeleteVpc(d.Id()); err != nil {
		ec2err, ok := err.(*ec2.Error)
		if ok && ec2err.Code == "InvalidVpcID.NotFound" {
			return nil
		}

		return fmt.Errorf("Error deleting VPC: %s", err)
	}

	return nil
}

func resourceAwsVpcRead(d *schema.ResourceData, meta interface{}) error {
	p := meta.(*ResourceProvider)
	ec2conn := p.ec2conn

	// Refresh the VPC state
	vpcRaw, _, err := VPCStateRefreshFunc(ec2conn, d.Id())()
	if err != nil {
		return err
	}
	if vpcRaw == nil {
		return nil
	}

	// VPC stuff
	vpc := vpcRaw.(*ec2.VPC)
	d.Set("cidr_block", vpc.CidrBlock)

	// Tags
	d.Set("tags", tagsToMap(vpc.Tags))

	// Attributes
	resp, err := ec2conn.VpcAttribute(d.Id(), "enableDnsSupport")
	if err != nil {
		return err
	}
	d.Set("enable_dns_support", resp.EnableDnsSupport)

	resp, err = ec2conn.VpcAttribute(d.Id(), "enableDnsHostnames")
	if err != nil {
		return err
	}
	d.Set("enable_dns_hostnames", resp.EnableDnsHostnames)

	// Get the main routing table for this VPC
	filter := ec2.NewFilter()
	filter.Add("association.main", "true")
	filter.Add("vpc-id", d.Id())
	routeResp, err := ec2conn.DescribeRouteTables(nil, filter)
	if err != nil {
		return err
	}
	if v := routeResp.RouteTables; len(v) > 0 {
		d.Set("main_route_table_id", v[0].RouteTableId)
	}

	return nil
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
