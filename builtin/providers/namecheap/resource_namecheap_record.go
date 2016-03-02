package namecheap

import (
	"fmt"
	"strconv"
	"sync"

	"github.com/HX-Rd/namecheap"
	"github.com/hashicorp/terraform/helper/schema"
)

var mutex = &sync.Mutex{}

const ncDefaultTTL int = 1800
const ncDefaultMXPref int = 10

func resourceNameCheapRecord() *schema.Resource {
	return &schema.Resource{
		Create: resourceNameCheapRecordCreate,
		Update: resourceNameCheapRecordUpdate,
		Read:   resourceNameCheapRecordRead,
		Delete: resourceNameCheapRecordDelete,

		Schema: map[string]*schema.Schema{
			"domain": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"hostname": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"recordType": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"address": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"mxPref": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				Default:  10,
			},
			"ttl": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				Default:  1800,
			},
		},
	}
}

func resourceNameCheapRecordCreate(d *schema.ResourceData, meta interface{}) error {
	mutex.Lock()
	client := meta.(*namecheap.Client)
	record := namecheap.Record{
		HostName:   d.Get("hostname").(string),
		RecordType: d.Get("recordType").(string),
		Address:    d.Get("address").(string),
		MXPref:     d.Get("mxPref").(int),
		TTL:        d.Get("ttl").(int),
	}

	_, err := client.AddRecord(d.Get("domain").(string), &record)

	if err != nil {
		mutex.Unlock()
		return fmt.Errorf("Failed to create namecheap Record: %s", err)
	}
	hashId := client.CreateHash(&record)
	d.SetId(strconv.Itoa(hashId))

	mutex.Unlock()
	return resourceNameCheapRecordRead(d, meta)
}

func resourceNameCheapRecordUpdate(d *schema.ResourceData, meta interface{}) error {
	mutex.Lock()
	client := meta.(*namecheap.Client)
	domain := d.Get("domain").(string)
	hashId, err := strconv.Atoi(d.Id())
	if err != nil {
		mutex.Unlock()
		return fmt.Errorf("Failed to parse id: %s", err)
	}
	record := namecheap.Record{
		HostName:   d.Get("hostname").(string),
		RecordType: d.Get("recordType").(string),
		Address:    d.Get("address").(string),
		MXPref:     d.Get("mxPref").(int),
		TTL:        d.Get("ttl").(int),
	}
	err = client.UpdateRecord(domain, hashId, &record)
	if err != nil {
		mutex.Unlock()
		return fmt.Errorf("Failed to update namecheap record: %s", err)
	}
	newHashId := client.CreateHash(&record)
	d.SetId(strconv.Itoa(newHashId))
	mutex.Unlock()
	return resourceNameCheapRecordRead(d, meta)
}

func resourceNameCheapRecordRead(d *schema.ResourceData, meta interface{}) error {
	mutex.Lock()

	client := meta.(*namecheap.Client)
	domain := d.Get("domain").(string)
	hashId, err := strconv.Atoi(d.Id())
	if err != nil {
		mutex.Unlock()
		return fmt.Errorf("Failed to parse id: %s", err)
	}

	record, err := client.ReadRecord(domain, hashId)
	if err != nil {
		mutex.Unlock()
		return fmt.Errorf("Couldn't find namecheap record: %s", err)
	}
	d.Set("hostname", record.HostName)
	d.Set("recordType", record.RecordType)
	d.Set("address", record.Address)
	d.Set("mxPref", record.MXPref)
	d.Set("ttl", record.TTL)
	mutex.Unlock()
	return nil
}

func resourceNameCheapRecordDelete(d *schema.ResourceData, meta interface{}) error {
	mutex.Lock()
	client := meta.(*namecheap.Client)
	domain := d.Get("domain").(string)
	hashId, err := strconv.Atoi(d.Id())
	if err != nil {
		mutex.Unlock()
		return fmt.Errorf("Failed to parse id: %s", err)
	}
	err = client.DeleteRecord(domain, hashId)

	if err != nil {
		mutex.Unlock()
		return fmt.Errorf("Failed to delete namecheap record: %s", err)
	}

	mutex.Unlock()
	return nil
}
