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

func resourceAwsNetworkAclAssociation() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsNetworkAclAssociationCreate,
		Read:   resourceAwsNetworkAclAssociationRead,
		Update: resourceAwsNetworkAclAssociationUpdate,
		Delete: resourceAwsNetworkAclAssociationDelete,

		Schema: map[string]*schema.Schema{
			"subnet_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"network_acl_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
		},
	}
}

func resourceAwsNetworkAclAssociationCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	naclId := d.Get("network_acl_id").(string)
	subnetId := d.Get("subnet_id").(string)

	log.Printf(
		"[INFO] Creating network acl association: %s => %s",
		subnetId,
		naclId)

	association, err_association := findNetworkAclAssociation(subnetId, conn)
	if err_association != nil {
		return fmt.Errorf("Failed to create acl %s with nacl %s: %s", d.Id(), naclId, err_association)
	}

	associationOpts := ec2.ReplaceNetworkAclAssociationInput{
		AssociationId: association.NetworkAclAssociationId,
		NetworkAclId:  aws.String(naclId),
	}

	var err error
	err = resource.Retry(5*time.Minute, func() *resource.RetryError {
		_, err = conn.ReplaceNetworkAclAssociation(&associationOpts)
		if err != nil {
			if awsErr, ok := err.(awserr.Error); ok {
				if awsErr != nil {
					return resource.RetryableError(awsErr)
				}
			}
			return resource.NonRetryableError(err)
		}
		return nil
	})
	if err != nil {
		return err
	}

	// Set the ID and return
	d.SetId(naclId)
	log.Printf("[INFO] Association ID: %s", d.Id())

	return nil
}

func resourceAwsNetworkAclAssociationRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	// Inspect that the association exists
	subnetId := d.Get("subnet_id").(string)
	_, err_association := findNetworkAclAssociation(subnetId, conn)
	if err_association != nil {
		return fmt.Errorf("Failed to read acl %s with subnet %s: %s", d.Id(), subnetId, err_association)
		d.SetId("")
	}

	return nil
}

func resourceAwsNetworkAclAssociationUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	naclId := d.Get("network_acl_id").(string)
	subnetId := d.Get("subnet_id").(string)

	log.Printf(
		"[INFO] Creating network acl association: %s => %s",
		subnetId,
		naclId)

	association, err_association := findNetworkAclAssociation(subnetId, conn)
	if err_association != nil {
		return fmt.Errorf("Failed to update acl %s with subnet %s: %s", d.Id(), naclId, err_association)
	}

	req := &ec2.ReplaceNetworkAclAssociationInput{
		AssociationId: association.NetworkAclAssociationId,
		NetworkAclId:  aws.String(naclId),
	}
	resp, err := conn.ReplaceNetworkAclAssociation(req)

	if err != nil {
		ec2err, ok := err.(awserr.Error)
		if ok && ec2err.Code() == "InvalidAssociationID.NotFound" {
			// Not found, so just create a new one
			return resourceAwsNetworkAclAssociationCreate(d, meta)
		}

		return err
	}

	// Update the ID
	d.SetId(*resp.NewAssociationId)
	log.Printf("[INFO] Association ID: %s", d.Id())

	return nil
}

func resourceAwsNetworkAclAssociationDelete(d *schema.ResourceData, meta interface{}) error {

	log.Printf("[INFO] Do nothing on network acl associatioØ destroy phase: %s", d.Id())

	return nil
}
