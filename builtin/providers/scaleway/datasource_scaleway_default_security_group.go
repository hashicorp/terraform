package scaleway

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/schema"
)

func datasourceScalewayDefaultSecurityGroup() *schema.Resource {
	return &schema.Resource{
		Read: datasourceScalewayDefaultSecurityGroupRead,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"description": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func datasourceScalewayDefaultSecurityGroupRead(d *schema.ResourceData, meta interface{}) error {
	scaleway := meta.(*Client).scaleway

	log.Printf("[DEBUG] Reading default security group")
	resp, err := scaleway.GetSecurityGroups()
	if err != nil {
		return fmt.Errorf("Error fetching security groups")
	}

	for _, sg := range resp.SecurityGroups {
		if sg.OrganizationDefault {
			d.SetId(sg.ID)
			d.Set("name", sg.Name)
			d.Set("description", sg.Description)
			break
		}
	}

	return nil
}
