package coredns

import (
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform/helper/schema"
	"k8s.io/kubernetes/federation/pkg/dnsprovider"
)

func resourceCorednsRecord() *schema.Resource {
	return &schema.Resource{
		Create: resourceCorednsRecordCreateUpdate,
		Read:   resourceCorednsRecordRead,
		Update: resourceCorednsRecordCreateUpdate,
		Delete: resourceCorednsRecordDelete,

		Schema: map[string]*schema.Schema{
			// Required
			"fqdn": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"type": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"rdata": &schema.Schema{
				Type:     schema.TypeSet,
				Set:      schema.HashString,
				Required: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			// Optional
			"ttl": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "3600",
			},
			// Computed
			"hostname": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func makeSetFromStrings(ss []string) *schema.Set {
	st := &schema.Set{F: schema.HashString}
	for _, s := range ss {
		st.Add(s)
	}
	return st
}

func newRRSetResource(d *schema.ResourceData) (rrsetData, error) {
	r := rrsetData{}

	fqdn := d.Get("fqdn").(string)
	rrtype := d.Get("type").(string)
	r.key = recordKey{
		RecordType: rrtype,
		FQDN:       fqdn,
	}

	if attr, ok := d.GetOk("rdata"); ok {
		rdata := attr.(*schema.Set).List()
		r.rdata = make([]string, len(rdata))
		for i, j := range rdata {
			r.rdata[i] = j.(string)
		}
	}

	if attr, ok := d.GetOk("ttl"); ok {
		r.ttl, _ = strconv.ParseInt(attr.(string), 10, 64)
	}

	return r, nil
}

func populateResourceDataFromRRSet(rs []dnsprovider.ResourceRecordSet, d *schema.ResourceData) error {
	// type
	d.Set("type", rs[0].Type())
	// ttl
	d.Set("ttl", rs[0].Ttl())
	// rdata
	var rdata []string
	for _, r := range rs {
		for _, d := range r.Rrdatas() {
			rdata = append(rdata, d)
		}
	}
	err := d.Set("rdata", makeSetFromStrings(rdata))
	if err != nil {
		return fmt.Errorf("coredns_record.rdata set failed: %#v", err)
	}
	// hostname
	d.Set("hostname", rs[0].Name())
	return nil
}

func resourceCorednsRecordCreateUpdate(d *schema.ResourceData, meta interface{}) error {
	dns := meta.(*dnsOp)
	r, err := newRRSetResource(d)
	if err != nil {
		return err
	}
	if err := dns.updateRecords(r.key, r.rdata, r.ttl); err != nil {
		return err
	}

	d.SetId(r.ID())
	return resourceCorednsRecordRead(d, meta)
}

func resourceCorednsRecordRead(d *schema.ResourceData, meta interface{}) error {
	dns := meta.(*dnsOp)
	r, err := newRRSetResource(d)
	rset, err := dns.getRecord(r.key)
	if err != nil {
		return err
	}
	return populateResourceDataFromRRSet(rset, d)
}

func resourceCorednsRecordDelete(d *schema.ResourceData, meta interface{}) error {
	dns := meta.(*dnsOp)
	r, err := newRRSetResource(d)
	if err != nil {
		return err
	}
	if err := dns.deleteRecords(r.key); err != nil {
		return err
	}

	return nil
}
