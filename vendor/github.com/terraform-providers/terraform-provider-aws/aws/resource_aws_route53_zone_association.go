package aws

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/route53"
)

func resourceAwsRoute53ZoneAssociation() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsRoute53ZoneAssociationCreate,
		Read:   resourceAwsRoute53ZoneAssociationRead,
		Update: resourceAwsRoute53ZoneAssociationUpdate,
		Delete: resourceAwsRoute53ZoneAssociationDelete,

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

func resourceAwsRoute53ZoneAssociationCreate(d *schema.ResourceData, meta interface{}) error {
	r53 := meta.(*AWSClient).r53conn

	req := &route53.AssociateVPCWithHostedZoneInput{
		HostedZoneId: aws.String(d.Get("zone_id").(string)),
		VPC: &route53.VPC{
			VPCId:     aws.String(d.Get("vpc_id").(string)),
			VPCRegion: aws.String(meta.(*AWSClient).region),
		},
		Comment: aws.String("Managed by Terraform"),
	}
	if w := d.Get("vpc_region"); w != "" {
		req.VPC.VPCRegion = aws.String(w.(string))
	}

	log.Printf("[DEBUG] Associating Route53 Private Zone %s with VPC %s with region %s", *req.HostedZoneId, *req.VPC.VPCId, *req.VPC.VPCRegion)
	var err error
	resp, err := r53.AssociateVPCWithHostedZone(req)
	if err != nil {
		return err
	}

	// Store association id
	d.SetId(fmt.Sprintf("%s:%s", *req.HostedZoneId, *req.VPC.VPCId))
	d.Set("vpc_region", req.VPC.VPCRegion)

	// Wait until we are done initializing
	wait := resource.StateChangeConf{
		Delay:      30 * time.Second,
		Pending:    []string{"PENDING"},
		Target:     []string{"INSYNC"},
		Timeout:    10 * time.Minute,
		MinTimeout: 2 * time.Second,
		Refresh: func() (result interface{}, state string, err error) {
			changeRequest := &route53.GetChangeInput{
				Id: aws.String(cleanChangeID(*resp.ChangeInfo.Id)),
			}
			return resourceAwsGoRoute53Wait(r53, changeRequest)
		},
	}
	_, err = wait.WaitForState()
	if err != nil {
		return err
	}

	return resourceAwsRoute53ZoneAssociationUpdate(d, meta)
}

func resourceAwsRoute53ZoneAssociationRead(d *schema.ResourceData, meta interface{}) error {
	r53 := meta.(*AWSClient).r53conn
	zone_id, vpc_id := resourceAwsRoute53ZoneAssociationParseId(d.Id())
	zone, err := r53.GetHostedZone(&route53.GetHostedZoneInput{Id: aws.String(zone_id)})
	if err != nil {
		// Handle a deleted zone
		if r53err, ok := err.(awserr.Error); ok && r53err.Code() == "NoSuchHostedZone" {
			d.SetId("")
			return nil
		}
		return err
	}

	for _, vpc := range zone.VPCs {
		if vpc_id == *vpc.VPCId {
			// association is there, return
			return nil
		}
	}

	// no association found
	d.SetId("")
	return nil
}

func resourceAwsRoute53ZoneAssociationUpdate(d *schema.ResourceData, meta interface{}) error {
	return resourceAwsRoute53ZoneAssociationRead(d, meta)
}

func resourceAwsRoute53ZoneAssociationDelete(d *schema.ResourceData, meta interface{}) error {
	r53 := meta.(*AWSClient).r53conn
	zone_id, vpc_id := resourceAwsRoute53ZoneAssociationParseId(d.Id())
	log.Printf("[DEBUG] Deleting Route53 Private Zone (%s) association (VPC: %s)",
		zone_id, vpc_id)

	req := &route53.DisassociateVPCFromHostedZoneInput{
		HostedZoneId: aws.String(zone_id),
		VPC: &route53.VPC{
			VPCId:     aws.String(vpc_id),
			VPCRegion: aws.String(d.Get("vpc_region").(string)),
		},
		Comment: aws.String("Managed by Terraform"),
	}

	_, err := r53.DisassociateVPCFromHostedZone(req)
	if err != nil {
		return err
	}

	return nil
}

func resourceAwsRoute53ZoneAssociationParseId(id string) (zone_id, vpc_id string) {
	parts := strings.SplitN(id, ":", 2)
	zone_id = parts[0]
	vpc_id = parts[1]
	return
}
