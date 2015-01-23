package aws

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/mitchellh/goamz/ec2"
)

func resourceAwsRouteTableAssociation() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsRouteTableAssociationCreate,
		Read:   resourceAwsRouteTableAssociationRead,
		Update: resourceAwsRouteTableAssociationUpdate,
		Delete: resourceAwsRouteTableAssociationDelete,

		Schema: map[string]*schema.Schema{
			"subnet_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"route_table_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
		},
	}
}

func resourceAwsRouteTableAssociationCreate(d *schema.ResourceData, meta interface{}) error {
	ec2conn := meta.(*AWSClient).ec2conn

	log.Printf(
		"[INFO] Creating route table association: %s => %s",
		d.Get("subnet_id").(string),
		d.Get("route_table_id").(string))

	resp, err := ec2conn.AssociateRouteTable(
		d.Get("route_table_id").(string),
		d.Get("subnet_id").(string))

	if err != nil {
		return err
	}

	// Set the ID and return
	d.SetId(resp.AssociationId)
	log.Printf("[INFO] Association ID: %s", d.Id())

	return nil
}

func resourceAwsRouteTableAssociationRead(d *schema.ResourceData, meta interface{}) error {
	ec2conn := meta.(*AWSClient).ec2conn

	// Get the routing table that this association belongs to
	rtRaw, _, err := resourceAwsRouteTableStateRefreshFunc(
		ec2conn, d.Get("route_table_id").(string))()
	if err != nil {
		return err
	}
	if rtRaw == nil {
		return nil
	}
	rt := rtRaw.(*ec2.RouteTable)

	// Inspect that the association exists
	found := false
	for _, a := range rt.Associations {
		if a.AssociationId == d.Id() {
			found = true
			d.Set("subnet_id", a.SubnetId)
			break
		}
	}

	if !found {
		// It seems it doesn't exist anymore, so clear the ID
		d.SetId("")
	}

	return nil
}

func resourceAwsRouteTableAssociationUpdate(d *schema.ResourceData, meta interface{}) error {
	ec2conn := meta.(*AWSClient).ec2conn

	log.Printf(
		"[INFO] Creating route table association: %s => %s",
		d.Get("subnet_id").(string),
		d.Get("route_table_id").(string))

	resp, err := ec2conn.ReassociateRouteTable(
		d.Id(),
		d.Get("route_table_id").(string))

	if err != nil {
		ec2err, ok := err.(*ec2.Error)
		if ok && ec2err.Code == "InvalidAssociationID.NotFound" {
			// Not found, so just create a new one
			return resourceAwsRouteTableAssociationCreate(d, meta)
		}

		return err
	}

	// Update the ID
	d.SetId(resp.AssociationId)
	log.Printf("[INFO] Association ID: %s", d.Id())

	return nil
}

func resourceAwsRouteTableAssociationDelete(d *schema.ResourceData, meta interface{}) error {
	ec2conn := meta.(*AWSClient).ec2conn

	log.Printf("[INFO] Deleting route table association: %s", d.Id())
	if _, err := ec2conn.DisassociateRouteTable(d.Id()); err != nil {
		ec2err, ok := err.(*ec2.Error)
		if ok && ec2err.Code == "InvalidAssociationID.NotFound" {
			return nil
		}

		return fmt.Errorf("Error deleting route table association: %s", err)
	}

	return nil
}
