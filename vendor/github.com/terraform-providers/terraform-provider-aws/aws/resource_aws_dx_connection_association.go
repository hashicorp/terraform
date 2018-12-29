package aws

import (
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/directconnect"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsDxConnectionAssociation() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsDxConnectionAssociationCreate,
		Read:   resourceAwsDxConnectionAssociationRead,
		Delete: resourceAwsDxConnectionAssociationDelete,

		Schema: map[string]*schema.Schema{
			"connection_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"lag_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
		},
	}
}

func resourceAwsDxConnectionAssociationCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).dxconn

	input := &directconnect.AssociateConnectionWithLagInput{
		ConnectionId: aws.String(d.Get("connection_id").(string)),
		LagId:        aws.String(d.Get("lag_id").(string)),
	}
	resp, err := conn.AssociateConnectionWithLag(input)
	if err != nil {
		return err
	}

	d.SetId(*resp.ConnectionId)
	return nil
}

func resourceAwsDxConnectionAssociationRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).dxconn

	input := &directconnect.DescribeConnectionsInput{
		ConnectionId: aws.String(d.Id()),
	}

	resp, err := conn.DescribeConnections(input)
	if err != nil {
		return err
	}
	if len(resp.Connections) < 1 {
		d.SetId("")
		return nil
	}
	if len(resp.Connections) != 1 {
		return fmt.Errorf("Found %d DX connections for %s, expected 1", len(resp.Connections), d.Id())
	}
	if *resp.Connections[0].LagId != d.Get("lag_id").(string) {
		d.SetId("")
		return nil
	}

	return nil
}

func resourceAwsDxConnectionAssociationDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).dxconn

	input := &directconnect.DisassociateConnectionFromLagInput{
		ConnectionId: aws.String(d.Id()),
		LagId:        aws.String(d.Get("lag_id").(string)),
	}

	return resource.Retry(1*time.Minute, func() *resource.RetryError {
		_, err := conn.DisassociateConnectionFromLag(input)
		if err != nil {
			if isAWSErr(err, directconnect.ErrCodeClientException, "is in a transitioning state.") {
				return resource.RetryableError(err)
			}
			return resource.NonRetryableError(err)
		}
		return nil
	})
}
