package aws

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourceAwsPrefixList() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceAwsPrefixListRead,

		Schema: map[string]*schema.Schema{
			"prefix_list_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			// Computed values.
			"id": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"cidr_blocks": &schema.Schema{
				Type:     schema.TypeList,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
		},
	}
}

func dataSourceAwsPrefixListRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	req := &ec2.DescribePrefixListsInput{}

	if prefixListID := d.Get("prefix_list_id"); prefixListID != "" {
		req.PrefixListIds = []*string{aws.String(prefixListID.(string))}
	}

	log.Printf("[DEBUG] DescribePrefixLists %s\n", req)
	resp, err := conn.DescribePrefixLists(req)
	if err != nil {
		return err
	}
	if resp == nil || len(resp.PrefixLists) == 0 {
		return fmt.Errorf("no matching prefix list found; the prefix list ID may be invalid or not exist in the current region")
	}

	pl := resp.PrefixLists[0]

	d.SetId(*pl.PrefixListId)
	d.Set("id", pl.PrefixListId)
	d.Set("name", pl.PrefixListName)

	cidrs := make([]string, len(pl.Cidrs))
	for i, v := range pl.Cidrs {
		cidrs[i] = *v
	}
	d.Set("cidr_blocks", cidrs)

	return nil
}
