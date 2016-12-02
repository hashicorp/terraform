package aws

import (
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsVpcEndpointRouteTableAssociation() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsVPCEndpointRouteTableAssociationCreate,
		Read:   resourceAwsVPCEndpointRouteTableAssociationRead,
		Update: resourceAwsVPCEndpointRouteTableAssociationUpdate,
		Delete: resourceAwsVPCEndpointRouteTableAssociationDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"vpc_endpoint_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"route_table_ids": {
				Type:     schema.TypeSet,
				Required: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},
		},
	}
}

func resourceAwsVPCEndpointRouteTableAssociationCreate(d *schema.ResourceData, meta interface{}) error {
	return nil
}

func resourceAwsVPCEndpointRouteTableAssociationRead(d *schema.ResourceData, meta interface{}) error {
	return nil
}

func resourceAwsVPCEndpointRouteTableAssociationUpdate(d *schema.ResourceData, meta interface{}) error {
	return nil
}

func resourceAwsVPCEndpointRouteTableAssociationDelete(d *schema.ResourceData, meta interface{}) error {
	return nil
}
