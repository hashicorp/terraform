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

			"ipv4addr": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
		},
	}
}

func resourceInfobloxRecordCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*infoblox.Client)

	//Create the new record
	newRecord := map[string]string{"ipv4addr": d.Get("ipv4addr").(string), "name": strings.Join([]string{d.Get("name").(string), d.Get("domain").(string)}, ".")}
	data := url.Values{}

	log.Printf("[DEBUG] Infoblox Record create configuration: %#v", newRecord)

	recId, err := client.RecordA().Create(data, nil, newRecord)

	if err != nil {
		return fmt.Errorf("Failed to create Infblox Record: %s", err)
	}

	d.SetId(recId)

	log.Printf("[INFO] record ID: %s", d.Id())

	return resourceInfobloxRecordRead(d, meta)
}

func resourceInfobloxRecordRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*infoblox.Client)

	rec, err := client.GetRecordA(d.Id())
	if err != nil {
		return fmt.Errorf("Couldn't find Infoblox record: %s", err)
	}

	d.Set("ipv4addr", rec.Ipv4Addr)
	fqdn := strings.Split(rec.Name, ".")
	d.Set("name", fqdn[0])
	d.Set("domain", strings.Join(fqdn[1:], "."))

	return nil
}

func resourceInfobloxRecordUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*infoblox.Client)

	_, err := client.GetRecordA(d.Id())
	if err != nil {
		return fmt.Errorf("Couldn't find Infoblox record: %s", err)
	}

	data := url.Values{}

	if attr, ok := d.GetOk("name"); ok {
		data.Set("name", attr.(string))
	}

	if attr, ok := d.GetOk("domain"); ok {
		data.Set("name", strings.Join([]string{data.Get("name"), attr.(string)}, "."))
	}

	if attr, ok := d.GetOk("ipv4addr"); ok {
		data.Set("ipv4addr", attr.(string))
	}

	log.Printf("[DEBUG] Infoblox Record update configuration: %#v", data)

	recId, update_err := client.RecordAObject(d.Id()).Update(data, nil)

	if update_err != nil {
		return fmt.Errorf("Failed to update Infblox Record: %s", err)
	}

	d.SetId(recId)

	return resourceInfobloxRecordRead(d, meta)
}

func resourceInfobloxRecordDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*infoblox.Client)

	log.Printf("[INFO] Deleting Infoblox Record: %s, %s", d.Get("name").(string), d.Id())

	_, err := client.GetRecordA(d.Id())
	if err != nil {
		return fmt.Errorf("Couldn't find Infoblox record: %s", err)
	}

	delete_err := client.RecordAObject(d.Id()).Delete(nil)

	if delete_err != nil {
		return fmt.Errorf("Error deleting Infoblox Record: %s", err)
	}

	return nil
}
