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
			"prefix_list_id": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"name": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"cidr_blocks": {
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
		req.PrefixListIds = aws.StringSlice([]string{prefixListID.(string)})
	}
	req.Filters = buildEC2AttributeFilterList(
		map[string]string{
			"prefix-list-name": d.Get("name").(string),
		},
	)

	log.Printf("[DEBUG] Reading Prefix List: %s", req)
	resp, err := conn.DescribePrefixLists(req)
	if err != nil {
		return err
	}
	if resp == nil || len(resp.PrefixLists) == 0 {
		return fmt.Errorf("no matching prefix list found; the prefix list ID or name may be invalid or not exist in the current region")
	}

	pl := resp.PrefixLists[0]

	d.SetId(*pl.PrefixListId)
	d.Set("name", pl.PrefixListName)

	cidrs := make([]string, len(pl.Cidrs))
	for i, v := range pl.Cidrs {
		cidrs[i] = *v
	}
	d.Set("cidr_blocks", cidrs)

	return nil
}
