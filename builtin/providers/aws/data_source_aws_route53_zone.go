package aws

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/route53"
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

		if err != nil {
			return err
		}

		if resp == nil || len(resp.HostedZones) == 0 || *resp.HostedZones[0].Name != name {
			return fmt.Errorf("no matching Route53Zone found")
		}

		hostedZone := resp.HostedZones[0]
		id := cleanZoneID(*hostedZone.Id)
		d.SetId(id)
		d.Set("zone_id", id)
		d.Set("name", hostedZone.Name)
		d.Set("comment", hostedZone.Config.Comment)

	} else if idExists {
		req := &route53.GetHostedZoneInput{}
		req.Id = aws.String(id.(string))

		resp, err := conn.GetHostedZone(req)

		if err != nil {
			return err
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
	} else {
		return fmt.Errorf("name or zone_id have to be setted")
	}

	return nil
}
