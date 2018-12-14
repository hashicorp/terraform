package aws

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/route53"
	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourceAwsDelegationSet() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceAwsDelegationSetRead,

		Schema: map[string]*schema.Schema{
			"id": {
				Type:     schema.TypeString,
				Required: true,
			},
			"caller_reference": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"name_servers": {
				Type:     schema.TypeList,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Computed: true,
			},
		},
	}
}

func dataSourceAwsDelegationSetRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).r53conn

	dSetID := d.Get("id").(string)

	input := &route53.GetReusableDelegationSetInput{
		Id: aws.String(dSetID),
	}

	log.Printf("[DEBUG] Reading Route53 delegation set: %s", input)

	resp, err := conn.GetReusableDelegationSet(input)
	if err != nil {
		return fmt.Errorf("Failed getting Route53 delegation set: %s Set: %q", err, dSetID)
	}

	d.SetId(dSetID)
	d.Set("caller_reference", resp.DelegationSet.CallerReference)

	servers := []string{}
	for _, server := range resp.DelegationSet.NameServers {
		if server != nil {
			servers = append(servers, *server)
		}
	}
	if err := d.Set("name_servers", expandNameServers(resp.DelegationSet.NameServers)); err != nil {
		return fmt.Errorf("error setting name_servers: %s", err)
	}

	return nil
}
