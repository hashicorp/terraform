package ultradns

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/Ensighten/udnssdk"
	"github.com/hashicorp/terraform/helper/schema"
)

func newRRSetResource(d *schema.ResourceData) (rRSetResource, error) {
	r := rRSetResource{}

	// TODO: return error if required attributes aren't ok

	if attr, ok := d.GetOk("name"); ok {
		r.OwnerName = attr.(string)
	}

	if attr, ok := d.GetOk("type"); ok {
		r.RRType = attr.(string)
	}

	if attr, ok := d.GetOk("zone"); ok {
		r.Zone = attr.(string)
	}

	if attr, ok := d.GetOk("rdata"); ok {
		rdata := attr.([]interface{})
		r.RData = make([]string, len(rdata))
		for i, j := range rdata {
			r.RData[i] = j.(string)
		}
	}

	if attr, ok := d.GetOk("ttl"); ok {
		r.TTL, _ = strconv.Atoi(attr.(string))
	}

	return r, nil
}

func populateResourceDataFromRRSet(r udnssdk.RRSet, d *schema.ResourceData) error {
	zone := d.Get("zone")
	// ttl
	d.Set("ttl", r.TTL)
	// rdata
	err := d.Set("rdata", r.RData)
	if err != nil {
		return fmt.Errorf("ultradns_record.rdata set failed: %#v", err)
	}
	// hostname
	if r.OwnerName == "" {
		d.Set("hostname", zone)
	} else {
		if strings.HasSuffix(r.OwnerName, ".") {
			d.Set("hostname", r.OwnerName)
		} else {
			d.Set("hostname", fmt.Sprintf("%s.%s", r.OwnerName, zone))
		}
	}
	return nil
}

func resourceUltraDNSRecord() *schema.Resource {
	return &schema.Resource{
		Create: resourceUltraDNSRecordCreate,
		Read:   resourceUltraDNSRecordRead,
		Update: resourceUltraDNSRecordUpdate,
		Delete: resourceUltraDNSRecordDelete,

		Schema: map[string]*schema.Schema{
			// Required
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
			"type": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"rdata": &schema.Schema{
				Type:     schema.TypeList,
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

// CRUD Operations

func resourceUltraDNSRecordCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*udnssdk.Client)

	r, err := newRRSetResource(d)
	if err != nil {
		return err
	}

	log.Printf("[INFO] ultradns_record create: %+v", r)
	_, err = client.RRSets.Create(r.RRSetKey(), r.RRSet())
	if err != nil {
		return fmt.Errorf("ultradns_record create failed: %v", err)
	}

	d.SetId(r.ID())
	log.Printf("[INFO] ultradns_record.id: %v", d.Id())

	return resourceUltraDNSRecordRead(d, meta)
}

func resourceUltraDNSRecordRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*udnssdk.Client)

	r, err := newRRSetResource(d)
	if err != nil {
		return err
	}

	rrsets, err := client.RRSets.Select(r.RRSetKey())
	if err != nil {
		uderr, ok := err.(*udnssdk.ErrorResponseList)
		if ok {
			for _, r := range uderr.Responses {
				// 70002 means Records Not Found
				if r.ErrorCode == 70002 {
					d.SetId("")
					return nil
				}
				return fmt.Errorf("ultradns_record not found: %v", err)
			}
		}
		return fmt.Errorf("ultradns_record not found: %v", err)
	}
	rec := rrsets[0]
	return populateResourceDataFromRRSet(rec, d)
}

func resourceUltraDNSRecordUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*udnssdk.Client)

	r, err := newRRSetResource(d)
	if err != nil {
		return err
	}

	log.Printf("[INFO] ultradns_record update: %+v", r)
	_, err = client.RRSets.Update(r.RRSetKey(), r.RRSet())
	if err != nil {
		return fmt.Errorf("ultradns_record update failed: %v", err)
	}

	return resourceUltraDNSRecordRead(d, meta)
}

func resourceUltraDNSRecordDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*udnssdk.Client)

	r, err := newRRSetResource(d)
	if err != nil {
		return err
	}

	log.Printf("[INFO] ultradns_record delete: %+v", r)
	_, err = client.RRSets.Delete(r.RRSetKey())
	if err != nil {
		return fmt.Errorf("ultradns_record delete failed: %v", err)
	}

	return nil
}

// Conversion helper functions
