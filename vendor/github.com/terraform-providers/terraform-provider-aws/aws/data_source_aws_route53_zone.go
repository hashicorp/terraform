package aws

import (
	"fmt"
	"strings"

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
			"vpc_id": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"tags": tagsSchemaComputed(),
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
	name = hostedZoneName(name.(string))
	id, idExists := d.GetOk("zone_id")
	vpcId, vpcIdExists := d.GetOk("vpc_id")
	tags := tagsFromMap(d.Get("tags").(map[string]interface{}))
	if nameExists && idExists {
		return fmt.Errorf("zone_id and name arguments can't be used together")
	} else if !nameExists && !idExists {
		return fmt.Errorf("Either name or zone_id must be set")
	}

	var nextMarker *string

	var hostedZoneFound *route53.HostedZone
	// We loop through all hostedzone
	for allHostedZoneListed := false; !allHostedZoneListed; {
		req := &route53.ListHostedZonesInput{}
		if nextMarker != nil {
			req.Marker = nextMarker
		}
		resp, err := conn.ListHostedZones(req)

		if err != nil {
			return fmt.Errorf("Error finding Route 53 Hosted Zone: %v", err)
		}
		for _, hostedZone := range resp.HostedZones {
			hostedZoneId := cleanZoneID(*hostedZone.Id)
			if idExists && hostedZoneId == id.(string) {
				hostedZoneFound = hostedZone
				break
				// we check if the name is the same as requested and if private zone field is the same as requested or if there is a vpc_id
			} else if *hostedZone.Name == name && (*hostedZone.Config.PrivateZone == d.Get("private_zone").(bool) || (*hostedZone.Config.PrivateZone == true && vpcIdExists)) {
				matchingVPC := false
				if vpcIdExists {
					reqHostedZone := &route53.GetHostedZoneInput{}
					reqHostedZone.Id = aws.String(hostedZoneId)

					respHostedZone, errHostedZone := conn.GetHostedZone(reqHostedZone)
					if errHostedZone != nil {
						return fmt.Errorf("Error finding Route 53 Hosted Zone: %v", errHostedZone)
					}
					// we go through all VPCs
					for _, vpc := range respHostedZone.VPCs {
						if *vpc.VPCId == vpcId.(string) {
							matchingVPC = true
							break
						}
					}
				} else {
					matchingVPC = true
				}
				// we check if tags match
				matchingTags := true
				if len(tags) > 0 {
					reqListTags := &route53.ListTagsForResourceInput{}
					reqListTags.ResourceId = aws.String(hostedZoneId)
					reqListTags.ResourceType = aws.String("hostedzone")
					respListTags, errListTags := conn.ListTagsForResource(reqListTags)

					if errListTags != nil {
						return fmt.Errorf("Error finding Route 53 Hosted Zone: %v", errListTags)
					}
					for _, tag := range tags {
						found := false
						for _, tagRequested := range respListTags.ResourceTagSet.Tags {
							if *tag.Key == *tagRequested.Key && *tag.Value == *tagRequested.Value {
								found = true
							}
						}

						if !found {
							matchingTags = false
							break
						}
					}

				}

				if matchingTags && matchingVPC {
					if hostedZoneFound != nil {
						return fmt.Errorf("multiple Route53Zone found please use vpc_id option to filter")
					} else {
						hostedZoneFound = hostedZone
					}
				}
			}

		}
		if *resp.IsTruncated {

			nextMarker = resp.NextMarker
		} else {
			allHostedZoneListed = true
		}
	}
	if hostedZoneFound == nil {
		return fmt.Errorf("no matching Route53Zone found")
	}

	idHostedZone := cleanZoneID(*hostedZoneFound.Id)
	d.SetId(idHostedZone)
	d.Set("zone_id", idHostedZone)
	d.Set("name", hostedZoneFound.Name)
	d.Set("comment", hostedZoneFound.Config.Comment)
	d.Set("private_zone", hostedZoneFound.Config.PrivateZone)
	d.Set("caller_reference", hostedZoneFound.CallerReference)
	d.Set("resource_record_set_count", hostedZoneFound.ResourceRecordSetCount)
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
