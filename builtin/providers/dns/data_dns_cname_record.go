package dns

import (
	"github.com/hashicorp/terraform/helper/schema"
	"net"
)

func dataSourceDnsCnameRecord() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceDnsCnameRecordRead,

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

func dataSourceDnsCnameRecordRead(d *schema.ResourceData, meta interface{}) error {
	host := d.Get("host").(string)

	cname, err := net.LookupCNAME(host)
	if err != nil {
		return err
	}

	d.Set("cname", cname)
	d.SetId(host)

	return nil
}
