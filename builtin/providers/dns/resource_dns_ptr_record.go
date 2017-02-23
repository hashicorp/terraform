package dns

import (
	"fmt"
	"time"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/miekg/dns"
)

func resourceDnsPtrRecord() *schema.Resource {
	return &schema.Resource{
		Create: resourceDnsPtrRecordCreate,
		Read:   resourceDnsPtrRecordRead,
		Update: resourceDnsPtrRecordUpdate,
		Delete: resourceDnsPtrRecordDelete,

		Schema: map[string]*schema.Schema{
			"zone": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"ptr": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"ttl": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				ForceNew: true,
				Default:  3600,
			},
		},
	}
}

func resourceDnsPtrRecordCreate(d *schema.ResourceData, meta interface{}) error {

	rec_name := d.Get("name").(string)
	rec_zone := d.Get("zone").(string)
	rec_ptr := d.Get("ptr").(string)

	if rec_zone != dns.Fqdn(rec_zone) {
		return fmt.Errorf("Error creating DNS record: \"zone\" should be an FQDN")
	}

	if rec_ptr != dns.Fqdn(rec_ptr) {
		return fmt.Errorf("Error creating DNS record: \"ptr\" should be an FQDN")
	}

	rec_fqdn := fmt.Sprintf("%s.%s", rec_name, rec_zone)

	d.SetId(rec_fqdn)

	return resourceDnsPtrRecordUpdate(d, meta)
}

func resourceDnsPtrRecordRead(d *schema.ResourceData, meta interface{}) error {

	if meta != nil {

		rec_name := d.Get("name").(string)
		rec_zone := d.Get("zone").(string)
		rec_ptr := d.Get("ptr").(string)

		if rec_zone != dns.Fqdn(rec_zone) {
			return fmt.Errorf("Error reading DNS record: \"zone\" should be an FQDN")
		}

		if rec_ptr != dns.Fqdn(rec_ptr) {
			return fmt.Errorf("Error reading DNS record: \"ptr\" should be an FQDN")
		}

		rec_fqdn := fmt.Sprintf("%s.%s", rec_name, rec_zone)

		c := meta.(*DNSClient).c
		srv_addr := meta.(*DNSClient).srv_addr

		msg := new(dns.Msg)
		msg.SetQuestion(rec_fqdn, dns.TypePTR)

		r, _, err := c.Exchange(msg, srv_addr)
		if err != nil {
			return fmt.Errorf("Error querying DNS record: %s", err)
		}
		if r.Rcode != dns.RcodeSuccess {
			return fmt.Errorf("Error querying DNS record: %v", r.Rcode)
		}

		if len(r.Answer) > 1 {
			return fmt.Errorf("Error querying DNS record: multiple responses received")
		}
		record := r.Answer[0]
		ptr, err := getPtrVal(record)
		if err != nil {
			return fmt.Errorf("Error querying DNS record: %s", err)
		}
		if rec_ptr != ptr {
			d.SetId("")
			return fmt.Errorf("DNS record differs")
		}
		return nil
	} else {
		return fmt.Errorf("update server is not set")
	}
}

func resourceDnsPtrRecordUpdate(d *schema.ResourceData, meta interface{}) error {

	if meta != nil {

		rec_name := d.Get("name").(string)
		rec_zone := d.Get("zone").(string)
		rec_ptr := d.Get("ptr").(string)
		ttl := d.Get("ttl").(int)

		if rec_zone != dns.Fqdn(rec_zone) {
			return fmt.Errorf("Error updating DNS record: \"zone\" should be an FQDN")
		}

		if rec_ptr != dns.Fqdn(rec_ptr) {
			return fmt.Errorf("Error updating DNS record: \"ptr\" should be an FQDN")
		}

		rec_fqdn := fmt.Sprintf("%s.%s", rec_name, rec_zone)

		c := meta.(*DNSClient).c
		srv_addr := meta.(*DNSClient).srv_addr
		keyname := meta.(*DNSClient).keyname
		keyalgo := meta.(*DNSClient).keyalgo

		msg := new(dns.Msg)

		msg.SetUpdate(rec_zone)

		if d.HasChange("ptr") {
			o, n := d.GetChange("ptr")

			if o != "" {
				rr_remove, _ := dns.NewRR(fmt.Sprintf("%s %d PTR %s", rec_fqdn, ttl, o))
				msg.Remove([]dns.RR{rr_remove})
			}
			if n != "" {
				rr_insert, _ := dns.NewRR(fmt.Sprintf("%s %d PTR %s", rec_fqdn, ttl, n))
				msg.Insert([]dns.RR{rr_insert})
			}

			if keyname != "" {
				msg.SetTsig(keyname, keyalgo, 300, time.Now().Unix())
			}

			r, _, err := c.Exchange(msg, srv_addr)
			if err != nil {
				d.SetId("")
				return fmt.Errorf("Error updating DNS record: %s", err)
			}
			if r.Rcode != dns.RcodeSuccess {
				d.SetId("")
				return fmt.Errorf("Error updating DNS record: %v", r.Rcode)
			}

			ptr := n
			d.Set("ptr", ptr)
		}

		return resourceDnsPtrRecordRead(d, meta)
	} else {
		return fmt.Errorf("update server is not set")
	}
}

func resourceDnsPtrRecordDelete(d *schema.ResourceData, meta interface{}) error {

	if meta != nil {

		rec_name := d.Get("name").(string)
		rec_zone := d.Get("zone").(string)

		if rec_zone != dns.Fqdn(rec_zone) {
			return fmt.Errorf("Error updating DNS record: \"zone\" should be an FQDN")
		}

		rec_fqdn := fmt.Sprintf("%s.%s", rec_name, rec_zone)

		c := meta.(*DNSClient).c
		srv_addr := meta.(*DNSClient).srv_addr
		keyname := meta.(*DNSClient).keyname
		keyalgo := meta.(*DNSClient).keyalgo

		msg := new(dns.Msg)

		msg.SetUpdate(rec_zone)

		rr_remove, _ := dns.NewRR(fmt.Sprintf("%s 0 PTR", rec_fqdn))
		msg.RemoveRRset([]dns.RR{rr_remove})

		if keyname != "" {
			msg.SetTsig(keyname, keyalgo, 300, time.Now().Unix())
		}

		r, _, err := c.Exchange(msg, srv_addr)
		if err != nil {
			return fmt.Errorf("Error deleting DNS record: %s", err)
		}
		if r.Rcode != dns.RcodeSuccess {
			return fmt.Errorf("Error deleting DNS record: %v", r.Rcode)
		}

		return nil
	} else {
		return fmt.Errorf("update server is not set")
	}
}
