package aws

import (
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsVpcDhcpOptionsAssociation() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsVpcDhcpOptionsAssociationCreate,
		Read:   resourceAwsVpcDhcpOptionsAssociationRead,
		Update: resourceAwsVpcDhcpOptionsAssociationUpdate,
		Delete: resourceAwsVpcDhcpOptionsAssociationDelete,

		Schema: map[string]*schema.Schema{
			"vpc_id": {
				Type:     schema.TypeString,
				Required: true,
			},

			"dhcp_options_id": {
				Type:     schema.TypeString,
				Required: true,
			},
		},
	}
}

func resourceAwsVpcDhcpOptionsAssociationCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	log.Printf(
		"[INFO] Creating DHCP Options association: %s => %s",
		d.Get("vpc_id").(string),
		d.Get("dhcp_options_id").(string))

	optsID := aws.String(d.Get("dhcp_options_id").(string))
	vpcID := aws.String(d.Get("vpc_id").(string))

	if _, err := conn.AssociateDhcpOptions(&ec2.AssociateDhcpOptionsInput{
		DhcpOptionsId: optsID,
		VpcId:         vpcID,
	}); err != nil {
		return err
	}

	// Set the ID and return
	d.SetId(*optsID + "-" + *vpcID)
	log.Printf("[INFO] Association ID: %s", d.Id())

	return nil
}

func resourceAwsVpcDhcpOptionsAssociationRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn
	// Get the VPC that this association belongs to
	vpcRaw, _, err := VPCStateRefreshFunc(conn, d.Get("vpc_id").(string))()

	if err != nil {
		return err
	}

	if vpcRaw == nil {
		return nil
	}

	vpc := vpcRaw.(*ec2.Vpc)
	if *vpc.VpcId != d.Get("vpc_id") || *vpc.DhcpOptionsId != d.Get("dhcp_options_id") {
		log.Printf("[INFO] It seems the DHCP Options association is gone. Deleting reference from Graph...")
		d.SetId("")
	}

	return nil
}

// DHCP Options Asociations cannot be updated.
func resourceAwsVpcDhcpOptionsAssociationUpdate(d *schema.ResourceData, meta interface{}) error {
	return resourceAwsVpcDhcpOptionsAssociationCreate(d, meta)
}

// AWS does not provide an API to disassociate a DHCP Options set from a VPC.
// So, we do this by setting the VPC to the default DHCP Options Set.
func resourceAwsVpcDhcpOptionsAssociationDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	log.Printf("[INFO] Disassociating DHCP Options Set %s from VPC %s...", d.Get("dhcp_options_id"), d.Get("vpc_id"))
	_, err := conn.AssociateDhcpOptions(&ec2.AssociateDhcpOptionsInput{
		DhcpOptionsId: aws.String("default"),
		VpcId:         aws.String(d.Get("vpc_id").(string)),
	})

	return err
}
