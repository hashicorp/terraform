package ultradns

import (
	"fmt"
	"log"
	"strings"

	"github.com/Ensighten/udnssdk"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
)

func resourceUltradnsRdpool() *schema.Resource {
	return &schema.Resource{
		Create: resourceUltradnsRdpoolCreate,
		Read:   resourceUltradnsRdpoolRead,
		Update: resourceUltradnsRdpoolUpdate,
		Delete: resourceUltradnsRdpoolDelete,

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
			"rdata": &schema.Schema{
				Type:     schema.TypeSet,
				Set:      schema.HashString,
				Required: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			// Optional
			"order": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "ROUND_ROBIN",
				ValidateFunc: validation.StringInSlice([]string{
					"ROUND_ROBIN",
					"FIXED",
					"RANDOM",
				}, false),
			},
			"description": &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.StringLenBetween(0, 255),
			},
			"ttl": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				Default:  3600,
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

func resourceUltradnsRdpoolCreate(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[INFO] ultradns_rdpool create")
	client := meta.(*udnssdk.Client)

	r, err := newRRSetResourceFromRdpool(d)
	if err != nil {
		return err
	}

	log.Printf("[INFO] ultradns_rdpool create: %#v", r)
	_, err = client.RRSets.Create(r.RRSetKey(), r.RRSet())
	if err != nil {
		return fmt.Errorf("create failed: %#v -> %v", r, err)
	}

	d.SetId(r.ID())
	log.Printf("[INFO] ultradns_rdpool.id: %v", d.Id())

	return resourceUltradnsRdpoolRead(d, meta)
}

func resourceUltradnsRdpoolRead(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[INFO] ultradns_rdpool read")
	client := meta.(*udnssdk.Client)

	rr, err := newRRSetResourceFromRdpool(d)
	if err != nil {
		return err
	}

	rrsets, err := client.RRSets.Select(rr.RRSetKey())
	if err != nil {
		uderr, ok := err.(*udnssdk.ErrorResponseList)
		if ok {
			for _, resps := range uderr.Responses {
				// 70002 means Records Not Found
				if resps.ErrorCode == 70002 {
					d.SetId("")
					return nil
				}
				return fmt.Errorf("resource not found: %v", err)
			}
		}
		return fmt.Errorf("resource not found: %v", err)
	}

	r := rrsets[0]

	zone := d.Get("zone")

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

	// And now... the Profile!
	if r.Profile == nil {
		return fmt.Errorf("RRSet.profile missing: invalid RDPool schema in: %#v", r)
	}
	p, err := r.Profile.RDPoolProfile()
	if err != nil {
		return fmt.Errorf("RRSet.profile could not be unmarshalled: %v\n", err)
	}

	// Set simple values
	d.Set("ttl", r.TTL)
	d.Set("description", p.Description)
	d.Set("order", p.Order)

	err = d.Set("rdata", makeSetFromStrings(r.RData))
	if err != nil {
		return fmt.Errorf("rdata set failed: %#v", err)
	}
	return nil
}

func resourceUltradnsRdpoolUpdate(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[INFO] ultradns_rdpool update")
	client := meta.(*udnssdk.Client)

	r, err := newRRSetResourceFromRdpool(d)
	if err != nil {
		return err
	}

	log.Printf("[INFO] ultradns_rdpool update: %+v", r)
	_, err = client.RRSets.Update(r.RRSetKey(), r.RRSet())
	if err != nil {
		return fmt.Errorf("resource update failed: %v", err)
	}

	return resourceUltradnsRdpoolRead(d, meta)
}

func resourceUltradnsRdpoolDelete(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[INFO] ultradns_rdpool delete")
	client := meta.(*udnssdk.Client)

	r, err := newRRSetResourceFromRdpool(d)
	if err != nil {
		return err
	}

	log.Printf("[INFO] ultradns_rdpool delete: %+v", r)
	_, err = client.RRSets.Delete(r.RRSetKey())
	if err != nil {
		return fmt.Errorf("resource delete failed: %v", err)
	}

	return nil
}

// Resource Helpers

func newRRSetResourceFromRdpool(d *schema.ResourceData) (rRSetResource, error) {
	//rDataRaw := d.Get("rdata").(*schema.Set).List()
	r := rRSetResource{
		// "The only valid rrtype value for RDpools is A"
		// per https://portal.ultradns.com/static/docs/REST-API_User_Guide.pdf
		RRType:    "A",
		Zone:      d.Get("zone").(string),
		OwnerName: d.Get("name").(string),
		TTL:       d.Get("ttl").(int),
	}
	if attr, ok := d.GetOk("rdata"); ok {
		rdata := attr.(*schema.Set).List()
		r.RData = make([]string, len(rdata))
		for i, j := range rdata {
			r.RData[i] = j.(string)
		}
	}

	profile := udnssdk.RDPoolProfile{
		Context:     udnssdk.RDPoolSchema,
		Order:       d.Get("order").(string),
		Description: d.Get("description").(string),
	}

	rp := profile.RawProfile()
	r.Profile = rp

	return r, nil
}
