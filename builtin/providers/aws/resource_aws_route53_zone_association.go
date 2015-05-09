package aws

import (
	"log"

	"github.com/hashicorp/terraform/helper/schema"

	"github.com/awslabs/aws-sdk-go/aws"
	"github.com/awslabs/aws-sdk-go/service/route53"
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
			},

			"association_id": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceAwsRoute53ZoneAssociationCreate(d *schema.ResourceData, meta interface{}) error {
	r53 := meta.(*AWSClient).r53conn

	req := &route53.AssociateVPCWithHostedZoneInput{
		HostedZoneID: aws.String(d.Get("zone_id").(string)),
		VPC: &route53.VPC{
			VPCID:     aws.String(d.Get("vpc_id").(string)),
			VPCRegion: aws.String(meta.(*AWSClient).region),
		},
		Comment: aws.String("Managed by Terraform"),
	}
	if w := d.Get("vpc_region"); w != "" {
		req.VPC.VPCRegion = aws.String(w.(string))
	}

	log.Printf("[DEBUG] Associating Route53 Private Zone %s with VPC %s", *req.HostedZoneID, *req.VPC.VPCID)
	resp, err := r53.AssociateVPCWithHostedZone(req)
	if err != nil {
		return err
	}

	// Store association id
	association_id := cleanChangeID(*resp.ChangeInfo.ID)
	d.Set("association_id", association_id)
	d.SetId(association_id)

	return resourceAwsRoute53ZoneAssociationUpdate(d, meta)
}

func resourceAwsRoute53ZoneAssociationRead(d *schema.ResourceData, meta interface{}) error {
	r53 := meta.(*AWSClient).r53conn
	zone, err := r53.GetHostedZone(&route53.GetHostedZoneInput{ID: aws.String(d.Id())})
	if err != nil {
		// Handle a deleted zone
		if r53err, ok := err.(aws.APIError); ok && r53err.Code == "NoSuchHostedZone" {
			d.SetId("")
			return nil
		}
		return err
	}

	vpc_id := d.Get("vpc_id")

	for i := range zone.VPCs {
		if vpc_id == *zone.VPCs[i].VPCID {
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

	log.Printf("[DEBUG] Deleting Route53 Private Zone (%s) association (ID: %s)",
		d.Get("zone_id").(string), d.Id())

	req := &route53.DisassociateVPCFromHostedZoneInput{
		HostedZoneID: aws.String(d.Get("zone_id").(string)),
		VPC: &route53.VPC{
			VPCID:     aws.String(d.Get("vpc_id").(string)),
			VPCRegion: aws.String(meta.(*AWSClient).region),
		},
		Comment: aws.String("Managed by Terraform"),
	}
	if w := d.Get("vpc_region"); w != "" {
		req.VPC.VPCRegion = aws.String(w.(string))
	}

	_, err := r53.DisassociateVPCFromHostedZone(req)
	if err != nil {
		return err
	}

	return nil
}
