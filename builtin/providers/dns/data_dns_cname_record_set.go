package dns

import (
	"fmt"
	"net"

	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourceDnsCnameRecordSet() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceDnsCnameRecordSetRead,

		Schema: map[string]*schema.Schema{
			"host": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"cname": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func dataSourceDnsCnameRecordSetRead(d *schema.ResourceData, meta interface{}) error {
	host := d.Get("host").(string)

	cname, err := net.LookupCNAME(host)
	if err != nil {
		return fmt.Errorf("error looking up CNAME records for %q: %s", host, err)
	}

	d.Set("cname", cname)
	d.SetId(host)

	return nil
}
