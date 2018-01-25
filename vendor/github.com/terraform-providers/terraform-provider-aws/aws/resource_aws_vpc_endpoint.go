package aws

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/structure"
)

func resourceAwsVpcEndpoint() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsVPCEndpointCreate,
		Read:   resourceAwsVPCEndpointRead,
		Update: resourceAwsVPCEndpointUpdate,
		Delete: resourceAwsVPCEndpointDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"policy": &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				ValidateFunc: validateJsonString,
				StateFunc: func(v interface{}) string {
					json, _ := structure.NormalizeJsonString(v)
					return json
				},
			},
			"vpc_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"service_name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"route_table_ids": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},
			"prefix_list_id": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"cidr_blocks": &schema.Schema{
				Type:     schema.TypeList,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
		},
	}
}

func resourceAwsVPCEndpointCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn
	input := &ec2.CreateVpcEndpointInput{
		VpcId:       aws.String(d.Get("vpc_id").(string)),
		ServiceName: aws.String(d.Get("service_name").(string)),
	}

	if v, ok := d.GetOk("route_table_ids"); ok {
		list := v.(*schema.Set).List()
		if len(list) > 0 {
			input.RouteTableIds = expandStringList(list)
		}
	}

	if v, ok := d.GetOk("policy"); ok {
		policy, err := structure.NormalizeJsonString(v)
		if err != nil {
			return errwrap.Wrapf("policy contains an invalid JSON: {{err}}", err)
		}
		input.PolicyDocument = aws.String(policy)
	}

	log.Printf("[DEBUG] Creating VPC Endpoint: %#v", input)
	output, err := conn.CreateVpcEndpoint(input)
	if err != nil {
		return fmt.Errorf("Error creating VPC Endpoint: %s", err)
	}
	log.Printf("[DEBUG] VPC Endpoint %q created.", *output.VpcEndpoint.VpcEndpointId)

	d.SetId(*output.VpcEndpoint.VpcEndpointId)

	return resourceAwsVPCEndpointRead(d, meta)
}

func resourceAwsVPCEndpointRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn
	input := &ec2.DescribeVpcEndpointsInput{
		VpcEndpointIds: []*string{aws.String(d.Id())},
	}

	log.Printf("[DEBUG] Reading VPC Endpoint: %q", d.Id())
	output, err := conn.DescribeVpcEndpoints(input)

	if err != nil {
		ec2err, ok := err.(awserr.Error)
		if !ok {
			return fmt.Errorf("Error reading VPC Endpoint: %s", err.Error())
		}

		if ec2err.Code() == "InvalidVpcEndpointId.NotFound" {
			log.Printf("[WARN] VPC Endpoint (%s) not found, removing from state", d.Id())
			d.SetId("")
			return nil
		}

		return fmt.Errorf("Error reading VPC Endpoint: %s", err.Error())
	}

	if len(output.VpcEndpoints) != 1 {
		return fmt.Errorf("There's no unique VPC Endpoint, but %d endpoints: %#v",
			len(output.VpcEndpoints), output.VpcEndpoints)
	}

	vpce := output.VpcEndpoints[0]

	// A VPC Endpoint is associated with exactly one prefix list name (also called Service Name).
	// The prefix list ID can be used in security groups, so retrieve it to support that capability.
	prefixListServiceName := *vpce.ServiceName
	prefixListInput := &ec2.DescribePrefixListsInput{
		Filters: []*ec2.Filter{
			{Name: aws.String("prefix-list-name"), Values: []*string{aws.String(prefixListServiceName)}},
		},
	}

	log.Printf("[DEBUG] Reading VPC Endpoint prefix list: %s", prefixListServiceName)
	prefixListsOutput, err := conn.DescribePrefixLists(prefixListInput)

	if err != nil {
		_, ok := err.(awserr.Error)
		if !ok {
			return fmt.Errorf("Error reading VPC Endpoint prefix list: %s", err.Error())
		}
	}

	if len(prefixListsOutput.PrefixLists) != 1 {
		return fmt.Errorf("There are multiple prefix lists associated with the service name '%s'. Unexpected", prefixListServiceName)
	}

	policy, err := structure.NormalizeJsonString(*vpce.PolicyDocument)
	if err != nil {
		return errwrap.Wrapf("policy contains an invalid JSON: {{err}}", err)
	}

	d.Set("vpc_id", vpce.VpcId)
	d.Set("policy", policy)
	d.Set("service_name", vpce.ServiceName)
	if err := d.Set("route_table_ids", aws.StringValueSlice(vpce.RouteTableIds)); err != nil {
		return err
	}
	pl := prefixListsOutput.PrefixLists[0]
	d.Set("prefix_list_id", pl.PrefixListId)
	d.Set("cidr_blocks", aws.StringValueSlice(pl.Cidrs))

	return nil
}

func resourceAwsVPCEndpointUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn
	input := &ec2.ModifyVpcEndpointInput{
		VpcEndpointId: aws.String(d.Id()),
	}

	if d.HasChange("route_table_ids") {
		o, n := d.GetChange("route_table_ids")
		os := o.(*schema.Set)
		ns := n.(*schema.Set)

		add := expandStringList(ns.Difference(os).List())
		if len(add) > 0 {
			input.AddRouteTableIds = add
		}

		remove := expandStringList(os.Difference(ns).List())
		if len(remove) > 0 {
			input.RemoveRouteTableIds = remove
		}
	}

	if d.HasChange("policy") {
		policy, err := structure.NormalizeJsonString(d.Get("policy"))
		if err != nil {
			return errwrap.Wrapf("policy contains an invalid JSON: {{err}}", err)
		}
		input.PolicyDocument = aws.String(policy)
	}

	log.Printf("[DEBUG] Updating VPC Endpoint: %#v", input)
	_, err := conn.ModifyVpcEndpoint(input)
	if err != nil {
		return fmt.Errorf("Error updating VPC Endpoint: %s", err)
	}
	log.Printf("[DEBUG] VPC Endpoint %q updated", input.VpcEndpointId)

	return resourceAwsVPCEndpointRead(d, meta)
}

func resourceAwsVPCEndpointDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn
	input := &ec2.DeleteVpcEndpointsInput{
		VpcEndpointIds: []*string{aws.String(d.Id())},
	}

	log.Printf("[DEBUG] Deleting VPC Endpoint: %#v", input)
	_, err := conn.DeleteVpcEndpoints(input)

	if err != nil {
		ec2err, ok := err.(awserr.Error)
		if !ok {
			return fmt.Errorf("Error deleting VPC Endpoint: %s", err.Error())
		}

		if ec2err.Code() == "InvalidVpcEndpointId.NotFound" {
			log.Printf("[DEBUG] VPC Endpoint %q is already gone", d.Id())
		} else {
			return fmt.Errorf("Error deleting VPC Endpoint: %s", err.Error())
		}
	}

	log.Printf("[DEBUG] VPC Endpoint %q deleted", d.Id())
	d.SetId("")

	return nil
}
