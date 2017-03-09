package aws

import (
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/terraform/helper/schema"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/route53"
)

func resourceAwsRoute53VPCAssociationAuthorization() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsRoute53VPCAssociationAuthorizationCreate,
		Read:   resourceAwsRoute53VPCAssociationAuthorizationRead,
		Update: resourceAwsRoute53VPCAssociationAuthorizationUpdate,
		Delete: resourceAwsRoute53VPCAssociationAuthorizationDelete,

		Schema: map[string]*schema.Schema{
			"zone_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"vpc_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"vpc_region": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
		},
	}
}

func resourceAwsRoute53VPCAssociationAuthorizationCreate(d *schema.ResourceData, meta interface{}) error {
	r53 := meta.(*AWSClient).r53conn

	req := &route53.CreateVPCAssociationAuthorizationInput{
		HostedZoneId: aws.String(d.Get("zone_id").(string)),
		VPC: &route53.VPC{
			VPCId:     aws.String(d.Get("vpc_id").(string)),
			VPCRegion: aws.String(meta.(*AWSClient).region),
		},
	}
	if w := d.Get("vpc_region"); w != "" {
		req.VPC.VPCRegion = aws.String(w.(string))
	}

	log.Printf("[DEBUG] Creating Route53 VPC Association Authorization for hosted zone %s with VPC %s and region %s", *req.HostedZoneId, *req.VPC.VPCId, *req.VPC.VPCRegion)
	var err error
	_, err = r53.CreateVPCAssociationAuthorization(req)
	if err != nil {
		return err
	}

	// Store association id
	d.SetId(fmt.Sprintf("%s:%s", *req.HostedZoneId, *req.VPC.VPCId))
	d.Set("vpc_region", req.VPC.VPCRegion)

	return resourceAwsRoute53VPCAssociationAuthorizationUpdate(d, meta)
}

func resourceAwsRoute53VPCAssociationAuthorizationRead(d *schema.ResourceData, meta interface{}) error {
	r53 := meta.(*AWSClient).r53conn
	zone_id, vpc_id := resourceAwsRoute53VPCAssociationAuthorizationParseId(d.Id())
	req := route53.ListVPCAssociationAuthorizationsInput{HostedZoneId: aws.String(zone_id)}
	for {
		res, err := r53.ListVPCAssociationAuthorizations(&req)
		if err != nil {
			return err
		}

		for _, vpc := range res.VPCs {
			if vpc_id == *vpc.VPCId {
				return nil
			}
		}

		// Loop till we find our authorization or we reach the end
		if res.NextToken != nil {
			req.NextToken = res.NextToken
		} else {
			break
		}
	}

	// no association found
	d.SetId("")
	return nil
}

func resourceAwsRoute53VPCAssociationAuthorizationUpdate(d *schema.ResourceData, meta interface{}) error {
	return resourceAwsRoute53VPCAssociationAuthorizationRead(d, meta)
}

func resourceAwsRoute53VPCAssociationAuthorizationDelete(d *schema.ResourceData, meta interface{}) error {
	r53 := meta.(*AWSClient).r53conn
	zone_id, vpc_id := resourceAwsRoute53VPCAssociationAuthorizationParseId(d.Id())
	log.Printf("[DEBUG] Deleting Route53 Assocatiation Authorization for (%s) from vpc %s)",
		zone_id, vpc_id)

	req := route53.DeleteVPCAssociationAuthorizationInput{
		HostedZoneId: aws.String(zone_id),
		VPC: &route53.VPC{
			VPCId:     aws.String(vpc_id),
			VPCRegion: aws.String(d.Get("vpc_region").(string)),
		},
	}

	_, err := r53.DeleteVPCAssociationAuthorization(&req)
	if err != nil {
		return err
	}

	return nil
}

func resourceAwsRoute53VPCAssociationAuthorizationParseId(id string) (zone_id, vpc_id string) {
	parts := strings.SplitN(id, ":", 2)
	zone_id = parts[0]
	vpc_id = parts[1]
	return
}
