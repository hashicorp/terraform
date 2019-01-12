package aws

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
)

func resourceAwsEc2TransitGatewayRouteTable() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsEc2TransitGatewayRouteTableCreate,
		Read:   resourceAwsEc2TransitGatewayRouteTableRead,
		Update: resourceAwsEc2TransitGatewayRouteTableUpdate,
		Delete: resourceAwsEc2TransitGatewayRouteTableDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"default_association_route_table": {
				Type:     schema.TypeBool,
				Computed: true,
			},
			"default_propagation_route_table": {
				Type:     schema.TypeBool,
				Computed: true,
			},
			"tags": {
				Type:     schema.TypeMap,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"transit_gateway_id": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.NoZeroValues,
			},
		},
	}
}

func resourceAwsEc2TransitGatewayRouteTableCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	input := &ec2.CreateTransitGatewayRouteTableInput{
		TransitGatewayId:  aws.String(d.Get("transit_gateway_id").(string)),
		TagSpecifications: expandEc2TransitGatewayRouteTableTagSpecifications(d.Get("tags").(map[string]interface{})),
	}

	log.Printf("[DEBUG] Creating EC2 Transit Gateway Route Table: %s", input)
	output, err := conn.CreateTransitGatewayRouteTable(input)
	if err != nil {
		return fmt.Errorf("error creating EC2 Transit Gateway Route Table: %s", err)
	}

	d.SetId(aws.StringValue(output.TransitGatewayRouteTable.TransitGatewayRouteTableId))

	if err := waitForEc2TransitGatewayRouteTableCreation(conn, d.Id()); err != nil {
		return fmt.Errorf("error waiting for EC2 Transit Gateway Route Table (%s) availability: %s", d.Id(), err)
	}

	return resourceAwsEc2TransitGatewayRouteTableRead(d, meta)
}

func resourceAwsEc2TransitGatewayRouteTableRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	transitGatewayRouteTable, err := ec2DescribeTransitGatewayRouteTable(conn, d.Id())

	if isAWSErr(err, "InvalidRouteTableID.NotFound", "") {
		log.Printf("[WARN] EC2 Transit Gateway Route Table (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	if err != nil {
		return fmt.Errorf("error reading EC2 Transit Gateway Route Table: %s", err)
	}

	if transitGatewayRouteTable == nil {
		log.Printf("[WARN] EC2 Transit Gateway Route Table (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	if aws.StringValue(transitGatewayRouteTable.State) == ec2.TransitGatewayRouteTableStateDeleting || aws.StringValue(transitGatewayRouteTable.State) == ec2.TransitGatewayRouteTableStateDeleted {
		log.Printf("[WARN] EC2 Transit Gateway Route Table (%s) in deleted state (%s), removing from state", d.Id(), aws.StringValue(transitGatewayRouteTable.State))
		d.SetId("")
		return nil
	}

	d.Set("default_association_route_table", aws.BoolValue(transitGatewayRouteTable.DefaultAssociationRouteTable))
	d.Set("default_propagation_route_table", aws.BoolValue(transitGatewayRouteTable.DefaultPropagationRouteTable))

	if err := d.Set("tags", tagsToMap(transitGatewayRouteTable.Tags)); err != nil {
		return fmt.Errorf("error setting tags: %s", err)
	}

	d.Set("transit_gateway_id", aws.StringValue(transitGatewayRouteTable.TransitGatewayId))

	return nil
}

func resourceAwsEc2TransitGatewayRouteTableUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	if err := setTags(conn, d); err != nil {
		return fmt.Errorf("error updating EC2 Transit Gateway Route Table (%s) tags: %s", d.Id(), err)
	}

	return nil
}

func resourceAwsEc2TransitGatewayRouteTableDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	input := &ec2.DeleteTransitGatewayRouteTableInput{
		TransitGatewayRouteTableId: aws.String(d.Id()),
	}

	log.Printf("[DEBUG] Deleting EC2 Transit Gateway Route Table (%s): %s", d.Id(), input)
	_, err := conn.DeleteTransitGatewayRouteTable(input)

	if isAWSErr(err, "InvalidRouteTableID.NotFound", "") {
		return nil
	}

	if err != nil {
		return fmt.Errorf("error deleting EC2 Transit Gateway Route Table: %s", err)
	}

	if err := waitForEc2TransitGatewayRouteTableDeletion(conn, d.Id()); err != nil {
		return fmt.Errorf("error waiting for EC2 Transit Gateway Route Table (%s) deletion: %s", d.Id(), err)
	}

	return nil
}
