package dns

import (
	"fmt"
	"net"

	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourceDnsTxtRecordSet() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceDnsTxtRecordSetRead,

		Schema: map[string]*schema.Schema{
			"host": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"record": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"records": &schema.Schema{
				Type:     schema.TypeList,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Computed: true,
			},
		},
	}
}

func dataSourceDnsTxtRecordSetRead(d *schema.ResourceData, meta interface{}) error {
	host := d.Get("host").(string)

	records, err := net.LookupTXT(host)
	if err != nil {
		return fmt.Errorf("error looking up TXT records for %q: %s", host, err)
	}

	if len(records) > 0 {
		d.Set("record", records[0])
	} else {
		d.Set("record", "")
	}
	d.Set("records", records)
	d.SetId(host)

	return nil
}
