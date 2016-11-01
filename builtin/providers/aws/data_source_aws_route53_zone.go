package aws

import (
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/route53"
	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourceAwsRoute53Zone() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceAwsRoute53ZoneRead,

		Schema: map[string]*schema.Schema{
			"zone_id": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"name": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"private_zone": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"comment": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"caller_reference": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"resource_record_set_count": {
				Type:     schema.TypeInt,
				Optional: true,
				Computed: true,
			},
		},
	}
}

func dataSourceAwsRoute53ZoneRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).r53conn
	name, nameExists := d.GetOk("name")
	id, idExists := d.GetOk("zone_id")
	if nameExists && idExists {
		return fmt.Errorf("zone_id and name arguments can't be used together")
	}

	if nameExists {
		req := &route53.ListHostedZonesByNameInput{}
		req.DNSName = aws.String(name.(string))
		req.MaxItems = aws.String("1")
		resp, err := conn.ListHostedZonesByName(req)
		name := hostedZoneName(name.(string))
		if err != nil {
			errwrap.Wrapf("Error finding Route 53 Hosted Zone: {{err}}", err)
		}

		if resp == nil || len(resp.HostedZones) == 0 || *resp.HostedZones[0].Name != name {
			return fmt.Errorf("no matching Route53Zone found")
		}
		// We test that the first HZ is private or not, if it's not match the field private_zone, we test the second one
		index := -1
		if *resp.HostedZones[0].Config.PrivateZone == d.Get("private_zone").(bool) {
			index = 0
		} else if len(resp.HostedZones) >= 2 && *resp.HostedZones[1].Name != name {
			index = 1
		} else {
			return fmt.Errorf("no matching Route53Zone found")
		}
		hostedZone := resp.HostedZones[index]
		id := cleanZoneID(*hostedZone.Id)
		d.SetId(id)
		d.Set("zone_id", id)
		d.Set("name", hostedZone.Name)
		d.Set("comment", hostedZone.Config.Comment)
		d.Set("private_zone", hostedZone.Config.PrivateZone)
		d.Set("caller_reference", hostedZone.CallerReference)
		d.Set("resource_record_set_count", hostedZone.ResourceRecordSetCount)

	} else if idExists {
		req := &route53.GetHostedZoneInput{}
		req.Id = aws.String(id.(string))

		resp, err := conn.GetHostedZone(req)

		if err != nil {
			errwrap.Wrapf("Error finding Route 53 Hosted Zone: {{err}}", err)
		}

		if resp == nil {
			return fmt.Errorf("no matching Route53Zone found")
		}
		hostedZone := resp.HostedZone
		id := cleanZoneID(*resp.HostedZone.Id)
		d.SetId(id)
		d.Set("zone_id", id)
		d.Set("name", hostedZone.Name)
		d.Set("comment", hostedZone.Config.Comment)
		d.Set("private_zone", hostedZone.Config.PrivateZone)
		d.Set("caller_reference", hostedZone.CallerReference)
		d.Set("resource_record_set_count", hostedZone.ResourceRecordSetCount)

	} else {
		return fmt.Errorf("Either name or zone_id must be set")
	}

	return nil
}

// used to manage trailing .
func hostedZoneName(name string) string {
	if strings.HasSuffix(name, ".") {
		return name
	} else {
		return name + "."
	}
}
