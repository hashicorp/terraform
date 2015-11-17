package ultradns

import (
	"fmt"
	"github.com/Ensighten/udnssdk"
	"github.com/hashicorp/terraform/helper/schema"
	"log"
	"strconv"
	"strings"
)

func resourceUltraDNSRecord() *schema.Resource {
	return &schema.Resource{
		Create: resourceUltraDNSRecordCreate,
		Read:   resourceUltraDNSRecordRead,
		Update: resourceUltraDNSRecordUpdate,
		Delete: resourceUltraDNSRecordDelete,

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

			"hostname": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
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

			"ttl": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "3600",
			},
		},
	}
}

func resourceUltraDNSRecordCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*udnssdk.Client)
	newRecord := &udnssdk.RRSet{
		OwnerName: d.Get("name").(string),
		RRType:    d.Get("type").(string),
	}
	rdata := d.Get("rdata").([]interface{})
	rdatas := make([]string, len(rdata))
	for i, j := range rdata {
		rdatas[i] = j.(string)
	}
	newRecord.RData = rdatas
	ttl := d.Get("ttl").(string)
	newRecord.TTL, _ = strconv.Atoi(ttl)

	log.Printf("[DEBUG] UltraDNS RRSet create configuration: %#v", newRecord)

	_, err := client.RRSets.CreateRRSet(d.Get("zone").(string), *newRecord)
	recId := fmt.Sprintf("%s.%s", d.Get("name").(string), d.Get("zone").(string))
	if err != nil {
		return fmt.Errorf("Failed to create UltraDNS RRSet: %s", err)
	}

	d.SetId(recId)
	log.Printf("[INFO] record ID: %s", d.Id())

	return resourceUltraDNSRecordRead(d, meta)
}

func resourceUltraDNSRecordRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*udnssdk.Client)

	rrsets, _, err := client.RRSets.GetRRSets(d.Get("zone").(string), d.Get("name").(string), d.Get("type").(string))
	if err != nil {
		uderr, ok := err.(*udnssdk.ErrorResponseList)
		if ok {
			for _, r := range uderr.Responses {
				// 70002 means Records Not Found
				if r.ErrorCode == 70002 {
					d.SetId("")
					return nil
				} else {
					return fmt.Errorf("Couldn't find UltraDNS RRSet: %s", err)
				}
			}
		} else {
			return fmt.Errorf("Couldn't find UltraDNS RRSet: %s", err)
		}
	}
	rec := rrsets[0]
	err = d.Set("rdata", rec.RData)
	if err != nil {
		return fmt.Errorf("[DEBUG] Error setting records: %#v", err)
	}
	d.Set("ttl", rec.TTL)

	if rec.OwnerName == "" {
		d.Set("hostname", d.Get("zone").(string))
	} else {
		if strings.HasSuffix(rec.OwnerName, ".") {
			d.Set("hostname", rec.OwnerName)
		} else {
			d.Set("hostname", fmt.Sprintf("%s.%s", rec.OwnerName, d.Get("zone").(string)))
		}
	}
	return nil
}

func resourceUltraDNSRecordUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*udnssdk.Client)

	updateRecord := &udnssdk.RRSet{}

	if attr, ok := d.GetOk("name"); ok {
		updateRecord.OwnerName = attr.(string)
	}

	if attr, ok := d.GetOk("type"); ok {
		updateRecord.RRType = attr.(string)
	}

	if attr, ok := d.GetOk("rdata"); ok {
		rdata := attr.([]interface{})
		rdatas := make([]string, len(rdata))
		for i, j := range rdata {
			rdatas[i] = j.(string)
		}
		updateRecord.RData = rdatas
	}

	if attr, ok := d.GetOk("ttl"); ok {
		updateRecord.TTL, _ = strconv.Atoi(attr.(string))
	}

	log.Printf("[DEBUG] UltraDNS RRSet update configuration: %#v", updateRecord)

	_, err := client.RRSets.UpdateRRSet(d.Get("zone").(string), *updateRecord)
	if err != nil {
		return fmt.Errorf("Failed to update UltraDNS RRSet: %s", err)
	}

	return resourceUltraDNSRecordRead(d, meta)
}

func resourceUltraDNSRecordDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*udnssdk.Client)

	log.Printf("[INFO] Deleting UltraDNS RRSet: %s, %s", d.Get("zone").(string), d.Id())
	deleteRecord := &udnssdk.RRSet{}

	if attr, ok := d.GetOk("name"); ok {
		deleteRecord.OwnerName = attr.(string)
	}

	if attr, ok := d.GetOk("type"); ok {
		deleteRecord.RRType = attr.(string)
	}

	_, err := client.RRSets.DeleteRRSet(d.Get("zone").(string), *deleteRecord)

	if err != nil {
		return fmt.Errorf("Error deleting UltraDNS RRSet: %s", err)
	}

	return nil
}
