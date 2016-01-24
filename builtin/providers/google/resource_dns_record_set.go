package google

import (
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/terraform/helper/schema"
	"google.golang.org/api/dns/v1"
	"google.golang.org/api/googleapi"
)

func resourceDnsRecordSet() *schema.Resource {
	return &schema.Resource{
		Create: resourceDnsRecordSetCreate,
		Read:   resourceDnsRecordSetRead,
		Delete: resourceDnsRecordSetDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"managed_zone": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"type": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"ttl": &schema.Schema{
				Type:     schema.TypeInt,
				Required: true,
				ForceNew: true,
			},

			"rrdatas": &schema.Schema{
				Type:     schema.TypeList,
				Required: true,
				ForceNew: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
		},
	}
}

func resourceDnsRecordSetCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	zone := d.Get("managed_zone").(string)

	rrdatasCount := d.Get("rrdatas.#").(int)

	// Build the change
	chg := &dns.Change{
		Additions: []*dns.ResourceRecordSet{
			&dns.ResourceRecordSet{
				Name:    d.Get("name").(string),
				Type:    d.Get("type").(string),
				Ttl:     int64(d.Get("ttl").(int)),
				Rrdatas: make([]string, rrdatasCount),
			},
		},
	}

	for i := 0; i < rrdatasCount; i++ {
		rrdata := fmt.Sprintf("rrdatas.%d", i)
		chg.Additions[0].Rrdatas[i] = d.Get(rrdata).(string)
	}

	log.Printf("[DEBUG] DNS Record create request: %#v", chg)
	chg, err := config.clientDns.Changes.Create(config.Project, zone, chg).Do()
	if err != nil {
		return fmt.Errorf("Error creating DNS RecordSet: %s", err)
	}

	d.SetId(chg.Id)

	w := &DnsChangeWaiter{
		Service:     config.clientDns,
		Change:      chg,
		Project:     config.Project,
		ManagedZone: zone,
	}
	state := w.Conf()
	state.Delay = 10 * time.Second
	state.Timeout = 10 * time.Minute
	state.MinTimeout = 2 * time.Second
	_, err = state.WaitForState()
	if err != nil {
		return fmt.Errorf("Error waiting for Google DNS change: %s", err)
	}

	return resourceDnsRecordSetRead(d, meta)
}

func resourceDnsRecordSetRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	zone := d.Get("managed_zone").(string)

	// name and type are effectively the 'key'
	name := d.Get("name").(string)
	dnsType := d.Get("type").(string)

	resp, err := config.clientDns.ResourceRecordSets.List(
		config.Project, zone).Name(name).Type(dnsType).Do()
	if err != nil {
		if gerr, ok := err.(*googleapi.Error); ok && gerr.Code == 404 {
			log.Printf("[WARN] Removing DNS Record Set %q because it's gone", d.Get("name").(string))
			// The resource doesn't exist anymore
			d.SetId("")

			return nil
		}

		return fmt.Errorf("Error reading DNS RecordSet: %#v", err)
	}
	if len(resp.Rrsets) == 0 {
		// The resource doesn't exist anymore
		d.SetId("")
		return nil
	}

	if len(resp.Rrsets) > 1 {
		return fmt.Errorf("Only expected 1 record set, got %d", len(resp.Rrsets))
	}

	d.Set("ttl", resp.Rrsets[0].Ttl)
	d.Set("rrdatas", resp.Rrsets[0].Rrdatas)

	return nil
}

func resourceDnsRecordSetDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	zone := d.Get("managed_zone").(string)

	rrdatasCount := d.Get("rrdatas.#").(int)

	// Build the change
	chg := &dns.Change{
		Deletions: []*dns.ResourceRecordSet{
			&dns.ResourceRecordSet{
				Name:    d.Get("name").(string),
				Type:    d.Get("type").(string),
				Ttl:     int64(d.Get("ttl").(int)),
				Rrdatas: make([]string, rrdatasCount),
			},
		},
	}

	for i := 0; i < rrdatasCount; i++ {
		rrdata := fmt.Sprintf("rrdatas.%d", i)
		chg.Deletions[0].Rrdatas[i] = d.Get(rrdata).(string)
	}
	log.Printf("[DEBUG] DNS Record delete request: %#v", chg)
	chg, err := config.clientDns.Changes.Create(config.Project, zone, chg).Do()
	if err != nil {
		return fmt.Errorf("Error deleting DNS RecordSet: %s", err)
	}

	w := &DnsChangeWaiter{
		Service:     config.clientDns,
		Change:      chg,
		Project:     config.Project,
		ManagedZone: zone,
	}
	state := w.Conf()
	state.Delay = 10 * time.Second
	state.Timeout = 10 * time.Minute
	state.MinTimeout = 2 * time.Second
	_, err = state.WaitForState()
	if err != nil {
		return fmt.Errorf("Error waiting for Google DNS change: %s", err)
	}

	d.SetId("")
	return nil
}
