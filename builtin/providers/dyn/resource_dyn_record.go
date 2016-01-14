package dyn

import (
	"fmt"
	"log"
	"sync"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/nesv/go-dynect/dynect"
)

var mutex = &sync.Mutex{}

func resourceDynRecord() *schema.Resource {
	return &schema.Resource{
		Create: resourceDynRecordCreate,
		Read:   resourceDynRecordRead,
		Update: resourceDynRecordUpdate,
		Delete: resourceDynRecordDelete,

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

			"fqdn": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"type": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"value": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"ttl": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "0", // 0 means use zone default
			},
		},
	}
}

func resourceDynRecordCreate(d *schema.ResourceData, meta interface{}) error {
	mutex.Lock()

	client := meta.(*dynect.ConvenientClient)

	record := &dynect.Record{
		Name:  d.Get("name").(string),
		Zone:  d.Get("zone").(string),
		Type:  d.Get("type").(string),
		TTL:   d.Get("ttl").(string),
		Value: d.Get("value").(string),
	}
	log.Printf("[DEBUG] Dyn record create configuration: %#v", record)

	// create the record
	err := client.CreateRecord(record)
	if err != nil {
		mutex.Unlock()
		return fmt.Errorf("Failed to create Dyn record: %s", err)
	}

	// publish the zone
	err = client.PublishZone(record.Zone)
	if err != nil {
		mutex.Unlock()
		return fmt.Errorf("Failed to publish Dyn zone: %s", err)
	}

	// get the record ID
	err = client.GetRecordID(record)
	if err != nil {
		mutex.Unlock()
		return fmt.Errorf("%s", err)
	}
	d.SetId(record.ID)

	mutex.Unlock()
	return resourceDynRecordRead(d, meta)
}

func resourceDynRecordRead(d *schema.ResourceData, meta interface{}) error {
	mutex.Lock()
	defer mutex.Unlock()

	client := meta.(*dynect.ConvenientClient)

	record := &dynect.Record{
		ID:   d.Id(),
		Name: d.Get("name").(string),
		Zone: d.Get("zone").(string),
		TTL:  d.Get("ttl").(string),
		FQDN: d.Get("fqdn").(string),
		Type: d.Get("type").(string),
	}

	err := client.GetRecord(record)
	if err != nil {
		return fmt.Errorf("Couldn't find Dyn record: %s", err)
	}

	d.Set("zone", record.Zone)
	d.Set("fqdn", record.FQDN)
	d.Set("name", record.Name)
	d.Set("type", record.Type)
	d.Set("ttl", record.TTL)
	d.Set("value", record.Value)

	return nil
}

func resourceDynRecordUpdate(d *schema.ResourceData, meta interface{}) error {
	mutex.Lock()

	client := meta.(*dynect.ConvenientClient)

	record := &dynect.Record{
		Name:  d.Get("name").(string),
		Zone:  d.Get("zone").(string),
		TTL:   d.Get("ttl").(string),
		Type:  d.Get("type").(string),
		Value: d.Get("value").(string),
	}
	log.Printf("[DEBUG] Dyn record update configuration: %#v", record)

	// update the record
	err := client.UpdateRecord(record)
	if err != nil {
		mutex.Unlock()
		return fmt.Errorf("Failed to update Dyn record: %s", err)
	}

	// publish the zone
	err = client.PublishZone(record.Zone)
	if err != nil {
		mutex.Unlock()
		return fmt.Errorf("Failed to publish Dyn zone: %s", err)
	}

	// get the record ID
	err = client.GetRecordID(record)
	if err != nil {
		mutex.Unlock()
		return fmt.Errorf("%s", err)
	}
	d.SetId(record.ID)

	mutex.Unlock()
	return resourceDynRecordRead(d, meta)
}

func resourceDynRecordDelete(d *schema.ResourceData, meta interface{}) error {
	mutex.Lock()
	defer mutex.Unlock()

	client := meta.(*dynect.ConvenientClient)

	record := &dynect.Record{
		ID:   d.Id(),
		Name: d.Get("name").(string),
		Zone: d.Get("zone").(string),
		FQDN: d.Get("fqdn").(string),
		Type: d.Get("type").(string),
	}

	log.Printf("[INFO] Deleting Dyn record: %s, %s", record.FQDN, record.ID)

	// delete the record
	err := client.DeleteRecord(record)
	if err != nil {
		return fmt.Errorf("Failed to delete Dyn record: %s", err)
	}

	// publish the zone
	err = client.PublishZone(record.Zone)
	if err != nil {
		return fmt.Errorf("Failed to publish Dyn zone: %s", err)
	}

	return nil
}
