package dns

import (
	"github.com/hashicorp/terraform/helper/schema"
	"net"
)

func dataSourceDnsTxtRecord() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceDnsTxtRecordRead,

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

func dataSourceDnsTxtRecordRead(d *schema.ResourceData, meta interface{}) error {
	records, err := net.LookupTXT(d.Get("host").(string))
	if err != nil {
		return err
	}

	if len(records) > 0 {
		d.Set("record", records[0])
	} else {
		d.Set("record", "")
	}
	d.Set("records", records)
	return nil
}
