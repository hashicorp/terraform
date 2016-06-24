package dns

import (
	"github.com/hashicorp/terraform/helper/schema"
	"net"
	"sort"
)

func dataSourceDnsARecord() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceDnsARecordRead,
		Schema: map[string]*schema.Schema{
			"host": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			// Optionally sort A records in alphabetical order.
			// This is helpful when a name uses round-robin DNS, which may
			// sort records with multiple addresses in a non-deterministic order.
			// This random sorting can cause flapping in terraform plans, where
			// the changes in sort order cause dependent resources to update
			// despite having no real change in the set of addresses.
			"sort": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},

			"addrs": &schema.Schema{
				Type:     schema.TypeList,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Computed: true,
			},
		},
	}
}

func dataSourceDnsARecordRead(d *schema.ResourceData, meta interface{}) error {
	host := d.Get("host").(string)

	records, err := net.LookupIP(host)
	if err != nil {
		return err
	}

	addrs := make([]string, 0)
	sortingEnabled := d.Get("sort").(bool)

	for _, ip := range records {
		// LookupIP returns A (IPv4) and AAAA (IPv6) records
		// Filter out AAAA records
		if ipv4 := ip.To4(); ipv4 != nil {
			addrs = append(addrs, ipv4.String())
		}
	}

	if sortingEnabled {
		sort.Strings(addrs)
	}

	d.Set("addrs", addrs)
	d.SetId(host)

	return nil
}
