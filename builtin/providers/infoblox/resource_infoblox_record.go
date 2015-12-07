package infoblox

import (
	"fmt"
	"log"
	"net/url"
	"strings"

	"github.com/fanatic/go-infoblox"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceInfobloxRecord() *schema.Resource {
	return &schema.Resource{
		Create: resourceInfobloxRecordCreate,
		Read:   resourceInfobloxRecordRead,
		Update: resourceInfobloxRecordUpdate,
		Delete: resourceInfobloxRecordDelete,

		Schema: map[string]*schema.Schema{
			"domain": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"value": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"type": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"ttl": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "3600",
			},
		},
	}
}

func resourceInfobloxRecordCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*infoblox.Client)

	record := url.Values{}
	if err := getAll(d, record); err != nil {
		return err
	}

	log.Printf("[DEBUG] Infoblox Record create configuration: %#v", record)

	var recId string
	var err error

	switch strings.ToUpper(d.Get("type").(string)) {
	case "A":
		opts := &infoblox.Options{
			ReturnFields: []string{"ttl", "ipv4addr", "name"},
		}
		recId, err = client.RecordA().Create(record, opts, nil)
	case "AAAA":
		opts := &infoblox.Options{
			ReturnFields: []string{"ttl", "ipv6addr", "name"},
		}
		recId, err = client.RecordAAAA().Create(record, opts, nil)
	case "CNAME":
		opts := &infoblox.Options{
			ReturnFields: []string{"ttl", "canonical", "name"},
		}
		recId, err = client.RecordCname().Create(record, opts, nil)
	}

	if err != nil {
		return fmt.Errorf("Failed to create Infblox Record: %s", err)
	}

	d.SetId(recId)

	log.Printf("[INFO] record ID: %s", d.Id())

	return resourceInfobloxRecordRead(d, meta)
}

func resourceInfobloxRecordRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*infoblox.Client)

	switch strings.ToUpper(d.Get("type").(string)) {
	case "A":
		rec, err := client.GetRecordA(d.Id())
		if err != nil {
			return fmt.Errorf("Couldn't find Infoblox A record: %s", err)
		}
		d.Set("value", rec.Ipv4Addr)
		d.Set("type", "A")
		fqdn := strings.Split(rec.Name, ".")
		d.Set("name", fqdn[0])
		d.Set("domain", strings.Join(fqdn[1:], "."))
		d.Set("ttl", rec.Ttl)

	case "AAAA":
		rec, err := client.GetRecordAAAA(d.Id())
		if err != nil {
			return fmt.Errorf("Couldn't find Infoblox AAAA record: %s", err)
		}
		d.Set("value", rec.Ipv6Addr)
		d.Set("type", "AAAA")
		fqdn := strings.Split(rec.Name, ".")
		d.Set("name", fqdn[0])
		d.Set("domain", strings.Join(fqdn[1:], "."))
		d.Set("ttl", rec.Ttl)

	case "CNAME":
		rec, err := client.GetRecordCname(d.Id())
		if err != nil {
			return fmt.Errorf("Couldn't find Infoblox CNAME record: %s", err)
		}
		d.Set("value", rec.Canoncial)
		d.Set("type", "CNAME")
		fqdn := strings.Split(rec.Name, ".")
		d.Set("name", fqdn[0])
		d.Set("domain", strings.Join(fqdn[1:], "."))
		d.Set("ttl", rec.Ttl)

	}

	return nil
}

func resourceInfobloxRecordUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*infoblox.Client)
	var recId string
	var err, update_err error
	switch strings.ToUpper(d.Get("type").(string)) {
	case "A":
		_, err = client.GetRecordA(d.Id())
	case "AAAA":
		_, err = client.GetRecordAAAA(d.Id())
	case "CNAME":
		_, err = client.GetRecordCname(d.Id())
	}

	if err != nil {
		return fmt.Errorf("Couldn't find Infoblox record: %s", err)
	}

	record := url.Values{}
	if err := getAll(d, record); err != nil {
		return err
	}

	log.Printf("[DEBUG] Infoblox Record update configuration: %#v", record)

	switch strings.ToUpper(d.Get("type").(string)) {
	case "A":
		opts := &infoblox.Options{
			ReturnFields: []string{"ttl", "ipv4addr", "name"},
		}
		recId, update_err = client.RecordAObject(d.Id()).Update(record, opts)
	case "AAAA":
		opts := &infoblox.Options{
			ReturnFields: []string{"ttl", "ipv6addr", "name"},
		}
		recId, update_err = client.RecordAAAAObject(d.Id()).Update(record, opts)
	case "CNAME":
		opts := &infoblox.Options{
			ReturnFields: []string{"ttl", "canonical", "name"},
		}
		recId, update_err = client.RecordCnameObject(d.Id()).Update(record, opts)
	}

	if update_err != nil {
		return fmt.Errorf("Failed to update Infblox Record: %s", err)
	}

	d.SetId(recId)

	return resourceInfobloxRecordRead(d, meta)
}

func resourceInfobloxRecordDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*infoblox.Client)

	log.Printf("[INFO] Deleting Infoblox Record: %s, %s", d.Get("name").(string), d.Id())
	switch strings.ToUpper(d.Get("type").(string)) {
	case "A":
		_, err := client.GetRecordA(d.Id())
		if err != nil {
			return fmt.Errorf("Couldn't find Infoblox A record: %s", err)
		}

		delete_err := client.RecordAObject(d.Id()).Delete(nil)
		if delete_err != nil {
			return fmt.Errorf("Error deleting Infoblox A Record: %s", err)
		}
	case "AAAA":
		_, err := client.GetRecordAAAA(d.Id())
		if err != nil {
			return fmt.Errorf("Couldn't find Infoblox AAAA record: %s", err)
		}

		delete_err := client.RecordAAAAObject(d.Id()).Delete(nil)
		if delete_err != nil {
			return fmt.Errorf("Error deleting Infoblox AAAA Record: %s", err)
		}
	case "CNAME":
		_, err := client.GetRecordCname(d.Id())
		if err != nil {
			return fmt.Errorf("Couldn't find Infoblox CNAME record: %s", err)
		}

		delete_err := client.RecordCnameObject(d.Id()).Delete(nil)
		if delete_err != nil {
			return fmt.Errorf("Error deleting Infoblox CNAME Record: %s", err)
		}
	}
	return nil
}

func getAll(d *schema.ResourceData, record url.Values) error {
	if attr, ok := d.GetOk("name"); ok {
		record.Set("name", attr.(string))
	}

	if attr, ok := d.GetOk("domain"); ok {
		record.Set("name", strings.Join([]string{record.Get("name"), attr.(string)}, "."))
	}

	if attr, ok := d.GetOk("ttl"); ok {
		record.Set("ttl", attr.(string))
	}

	var value string
	if attr, ok := d.GetOk("value"); ok {
		value = attr.(string)
	}

	switch strings.ToUpper(d.Get("type").(string)) {
	case "A":
		record.Set("ipv4addr", value)
	case "AAAA":
		record.Set("ipv6addr", value)
	case "CNAME":
		record.Set("canonical", value)
	default:
		return fmt.Errorf("getAll: type not found")
	}

	return nil
}
