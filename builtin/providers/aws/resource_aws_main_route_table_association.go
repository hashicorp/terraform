package aws

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/mitchellh/goamz/ec2"
)

func resourceAwsMainRouteTableAssociation() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsMainRouteTableAssociationCreate,
		Read:   resourceAwsMainRouteTableAssociationRead,
		Update: resourceAwsMainRouteTableAssociationUpdate,
		Delete: resourceAwsMainRouteTableAssociationDelete,

		Schema: map[string]*schema.Schema{
			"vpc_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"route_table_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			// We use this field to record the main route table that is automatically
			// created when the VPC is created. We need this to be able to "destroy"
			// our main route table association, which we do by returning this route
			// table to its original place as the Main Route Table for the VPC.
			"original_route_table_id": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceAwsMainRouteTableAssociationCreate(d *schema.ResourceData, meta interface{}) error {
	ec2conn := meta.(*AWSClient).ec2conn
	vpcId := d.Get("vpc_id").(string)
	routeTableId := d.Get("route_table_id").(string)

	log.Printf("[INFO] Creating main route table association: %s => %s", vpcId, routeTableId)

	mainAssociation, err := findMainRouteTableAssociation(ec2conn, vpcId)
	if err != nil {
		return err
	}

	resp, err := ec2conn.ReassociateRouteTable(
		mainAssociation.AssociationId,
		routeTableId,
	)
	if err != nil {
		return err
	}

	d.Set("original_route_table_id", mainAssociation.RouteTableId)
	d.SetId(resp.AssociationId)
	log.Printf("[INFO] New main route table association ID: %s", d.Id())

	return nil
}

func resourceAwsMainRouteTableAssociationRead(d *schema.ResourceData, meta interface{}) error {
	ec2conn := meta.(*AWSClient).ec2conn

	mainAssociation, err := findMainRouteTableAssociation(
		ec2conn,
		d.Get("vpc_id").(string))
	if err != nil {
		return err
	}

	if mainAssociation.AssociationId != d.Id() {
		// It seems it doesn't exist anymore, so clear the ID
		d.SetId("")
	}

	return nil
}

// Update is almost exactly like Create, except we want to retain the
// original_route_table_id - this needs to stay recorded as the AWS-created
// table from VPC creation.
func resourceAwsMainRouteTableAssociationUpdate(d *schema.ResourceData, meta interface{}) error {
	ec2conn := meta.(*AWSClient).ec2conn
	vpcId := d.Get("vpc_id").(string)
	routeTableId := d.Get("route_table_id").(string)

	log.Printf("[INFO] Updating main route table association: %s => %s", vpcId, routeTableId)

	resp, err := ec2conn.ReassociateRouteTable(d.Id(), routeTableId)
	if err != nil {
		return err
	}

	d.SetId(resp.AssociationId)
	log.Printf("[INFO] New main route table association ID: %s", d.Id())

	return nil
}

func resourceAwsMainRouteTableAssociationDelete(d *schema.ResourceData, meta interface{}) error {
	ec2conn := meta.(*AWSClient).ec2conn
	vpcId := d.Get("vpc_id").(string)
	originalRouteTableId := d.Get("original_route_table_id").(string)

	log.Printf("[INFO] Deleting main route table association by resetting Main Route Table for VPC: %s to its original Route Table: %s",
		vpcId,
		originalRouteTableId)

	resp, err := ec2conn.ReassociateRouteTable(d.Id(), originalRouteTableId)
	if err != nil {
		return err
	}

	log.Printf("[INFO] Resulting Association ID: %s", resp.AssociationId)

	return nil
}

func findMainRouteTableAssociation(ec2conn *ec2.EC2, vpcId string) (*ec2.RouteTableAssociation, error) {
	mainRouteTable, err := findMainRouteTable(ec2conn, vpcId)
	if err != nil {
		return nil, err
	}

	for _, a := range mainRouteTable.Associations {
		if a.Main {
			return &a, nil
		}
	}
	return nil, fmt.Errorf("Could not find main routing table association for VPC: %s", vpcId)
}

func findMainRouteTable(ec2conn *ec2.EC2, vpcId string) (*ec2.RouteTable, error) {
	filter := ec2.NewFilter()
	filter.Add("association.main", "true")
	filter.Add("vpc-id", vpcId)
	routeResp, err := ec2conn.DescribeRouteTables(nil, filter)
	if err != nil {
		return nil, err
	} else if len(routeResp.RouteTables) != 1 {
		return nil, fmt.Errorf(
			"Expected to find a single main routing table for VPC: %s, but found %d",
			vpcId,
			len(routeResp.RouteTables))
	}

	return &routeResp.RouteTables[0], nil
}
