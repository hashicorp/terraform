package dme

import (
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/soniah/dnsmadeeasy"
)

func resourceDMERecord() *schema.Resource {
	return &schema.Resource{
		Create: resourceDMERecordCreate,
		Read:   resourceDMERecordRead,
		Update: resourceDMERecordUpdate,
		Delete: resourceDMERecordDelete,

		Schema: map[string]*schema.Schema{
			// Use recordid for TF ID.
			"domainid": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"type": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"value": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"ttl": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
			},
			"mxLevel": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
			},
			"weight": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
			},
			"priority": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
			},
			"port": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
			},
			"keywords": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"title": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"hardLink": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
			},
			"redirectType": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"description": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
		},
	}
}

func resourceDMERecordCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*dnsmadeeasy.Client)

	domainid := d.Get("domainid").(string)
	log.Printf("[INFO] Creating record for domainid: %s", domainid)

	cr := make(map[string]interface{})
	if err := getAll(d, cr); err != nil {
		return err
	}
	log.Printf("[DEBUG] record create configuration: %#v", cr)

	result, err := client.CreateRecord(domainid, cr)
	if err != nil {
		return fmt.Errorf("Failed to create record: %s", err)
	}

	d.SetId(result)
	log.Printf("[INFO] record ID: %s", d.Id())

	return resourceDMERecordRead(d, meta)
}

func resourceDMERecordRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*dnsmadeeasy.Client)

	domainid := d.Get("domainid").(string)
	recordid := d.Id()
	log.Printf("[INFO] Reading record for domainid: %s recordid: %s", domainid, recordid)

	rec, err := client.ReadRecord(domainid, recordid)
	if err != nil {
		if strings.Contains(err.Error(), "Unable to find") {
			d.SetId("")
			return nil
		}

		return fmt.Errorf("Couldn't find record: %s", err)
	}

	return setAll(d, rec)
}

func resourceDMERecordUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*dnsmadeeasy.Client)

	domainid := d.Get("domainid").(string)
	recordid := d.Id()

	cr := make(map[string]interface{})
	if err := getAll(d, cr); err != nil {
		return err
	}
	log.Printf("[DEBUG] record update configuration: %+#v", cr)

	if _, err := client.UpdateRecord(domainid, recordid, cr); err != nil {
		return fmt.Errorf("Error updating record: %s", err)
	}

	return resourceDMERecordRead(d, meta)
}

func resourceDMERecordDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*dnsmadeeasy.Client)

	domainid := d.Get("domainid").(string)
	recordid := d.Id()
	log.Printf("[INFO] Deleting record for domainid: %s recordid: %s", domainid, recordid)

	if err := client.DeleteRecord(domainid, recordid); err != nil {
		return fmt.Errorf("Error deleting record: %s", err)
	}

	return nil
}

func getAll(d *schema.ResourceData, cr map[string]interface{}) error {

	if attr, ok := d.GetOk("name"); ok {
		cr["name"] = attr.(string)
	}
	if attr, ok := d.GetOk("type"); ok {
		cr["type"] = attr.(string)
	}
	if attr, ok := d.GetOk("ttl"); ok {
		cr["ttl"] = int64(attr.(int))
	}
	if attr, ok := d.GetOk("value"); ok {
		cr["value"] = attr.(string)
	}

	switch strings.ToUpper(d.Get("type").(string)) {
	case "A", "CNAME", "ANAME", "TXT", "SPF", "NS", "PTR", "AAAA":
		// all done
	case "MX":
		if attr, ok := d.GetOk("mxLevel"); ok {
			cr["mxLevel"] = int64(attr.(int))
		}
	case "SRV":
		if attr, ok := d.GetOk("priority"); ok {
			cr["priority"] = int64(attr.(int))
		}
		if attr, ok := d.GetOk("weight"); ok {
			cr["weight"] = int64(attr.(int))
		}
		if attr, ok := d.GetOk("port"); ok {
			cr["port"] = int64(attr.(int))
		}
	case "HTTPRED":
		if attr, ok := d.GetOk("hardLink"); ok && attr.(bool) {
			cr["hardLink"] = "true"
		}
		if attr, ok := d.GetOk("redirectType"); ok {
			cr["redirectType"] = attr.(string)
		}
		if attr, ok := d.GetOk("title"); ok {
			cr["title"] = attr.(string)
		}
		if attr, ok := d.GetOk("keywords"); ok {
			cr["keywords"] = attr.(string)
		}
		if attr, ok := d.GetOk("description"); ok {
			cr["description"] = attr.(string)
		}
	default:
		return fmt.Errorf("getAll: type not found")
	}
	return nil
}

func setAll(d *schema.ResourceData, rec *dnsmadeeasy.Record) error {
	d.Set("type", rec.Type)
	d.Set("name", rec.Name)
	d.Set("ttl", rec.TTL)
	d.Set("value", rec.Value)

	switch rec.Type {
	case "A", "CNAME", "ANAME", "TXT", "SPF", "NS", "PTR":
		// all done
	case "AAAA":
		// overwrite value set above - DME ipv6 is lower case
		d.Set("value", strings.ToLower(rec.Value))
	case "MX":
		d.Set("mxLevel", rec.MXLevel)
	case "SRV":
		d.Set("priority", rec.Priority)
		d.Set("weight", rec.Weight)
		d.Set("port", rec.Port)
	case "HTTPRED":
		d.Set("hardLink", rec.HardLink)
		d.Set("redirectType", rec.RedirectType)
		d.Set("title", rec.Title)
		d.Set("keywords", rec.Keywords)
		d.Set("description", rec.Description)
	default:
		return fmt.Errorf("setAll: type not found")
	}
	return nil
}
