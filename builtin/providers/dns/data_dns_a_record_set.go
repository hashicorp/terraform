package dns

import (
	"fmt"
	"net"
	"sort"

	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourceDnsARecordSet() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceDnsARecordSetRead,
		Schema: map[string]*schema.Schema{
			"host": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"addrs": &schema.Schema{
				Type:     schema.TypeList,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Computed: true,
			},
		},
	}
}

func dataSourceDnsARecordSetRead(d *schema.ResourceData, meta interface{}) error {
	host := d.Get("host").(string)

	records, err := net.LookupIP(host)
	if err != nil {
		return fmt.Errorf("error looking up A records for %q: %s", host, err)
	}

	addrs := make([]string, 0)

	for _, ip := range records {
		// LookupIP returns A (IPv4) and AAAA (IPv6) records
		// Filter out AAAA records
		if ipv4 := ip.To4(); ipv4 != nil {
			addrs = append(addrs, ipv4.String())
		}
	}

	sort.Strings(addrs)

	d.Set("addrs", addrs)
	d.SetId(host)

	return nil
}
