package cloudflare

import (
	"fmt"
	"log"
	"strings"
	"time"

	"golang.org/x/net/context"

	"github.com/crackcomm/cloudflare"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceCloudFlareRecord() *schema.Resource {
	return &schema.Resource{
		Create: resourceCloudFlareRecordCreate,
		Read:   resourceCloudFlareRecordRead,
		Update: resourceCloudFlareRecordUpdate,
		Delete: resourceCloudFlareRecordDelete,

		Schema: map[string]*schema.Schema{
			"domain": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
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

			"value": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"ttl": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				Computed: true,
			},

			"priority": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
			},

			"proxied": &schema.Schema{
				Default:  false,
				Optional: true,
				Type:     schema.TypeBool,
			},

			"zone_id": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceCloudFlareRecordCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*cloudflare.Client)

	newRecord := &cloudflare.Record{
		Content:  d.Get("value").(string),
		Name:     d.Get("name").(string),
		Proxied:  d.Get("proxied").(bool),
		Type:     d.Get("type").(string),
		ZoneName: d.Get("domain").(string),
	}

	if priority, ok := d.GetOk("priority"); ok {
		newRecord.Priority = priority.(int)
	}

	if ttl, ok := d.GetOk("ttl"); ok {
		newRecord.TTL = ttl.(int)
	}

	zone, err := retrieveZone(client, newRecord.ZoneName)
	if err != nil {
		return err
	}

	d.Set("zone_id", zone.ID)
	newRecord.ZoneID = zone.ID

	log.Printf("[DEBUG] CloudFlare Record create configuration: %#v", newRecord)

	ctx, _ := context.WithDeadline(context.Background(), time.Now().Add(time.Second*30))

	err = client.Records.Create(ctx, newRecord)
	if err != nil {
		return fmt.Errorf("Failed to create record: %s", err)
	}

	d.SetId(newRecord.ID)

	log.Printf("[INFO] CloudFlare Record ID: %s", d.Id())

	return resourceCloudFlareRecordRead(d, meta)
}

func resourceCloudFlareRecordRead(d *schema.ResourceData, meta interface{}) error {
	var (
		client = meta.(*cloudflare.Client)
		domain = d.Get("domain").(string)
		rName  = strings.Join([]string{d.Get("name").(string), domain}, ".")
	)

	zone, err := retrieveZone(client, domain)
	if err != nil {
		return err
	}

	record, err := retrieveRecord(client, zone, rName)
	if err != nil {
		return err
	}

	d.SetId(record.ID)
	d.Set("hostname", record.Name)
	d.Set("type", record.Type)
	d.Set("value", record.Content)
	d.Set("ttl", record.TTL)
	d.Set("priority", record.Priority)
	d.Set("proxied", record.Proxied)
	d.Set("zone_id", zone.ID)

	return nil
}

func resourceCloudFlareRecordUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*cloudflare.Client)

	updateRecord := &cloudflare.Record{
		Content:  d.Get("value").(string),
		ID:       d.Id(),
		Name:     d.Get("name").(string),
		Proxied:  false,
		Type:     d.Get("type").(string),
		ZoneName: d.Get("domain").(string),
	}

	if priority, ok := d.GetOk("priority"); ok {
		updateRecord.Priority = priority.(int)
	}

	if proxied, ok := d.GetOk("proxied"); ok {
		updateRecord.Proxied = proxied.(bool)
	}

	if ttl, ok := d.GetOk("ttl"); ok {
		updateRecord.TTL = ttl.(int)
	}

	zone, err := retrieveZone(client, updateRecord.ZoneName)
	if err != nil {
		return err
	}

	updateRecord.ZoneID = zone.ID

	log.Printf("[DEBUG] CloudFlare Record update configuration: %#v", updateRecord)

	ctx, _ := context.WithDeadline(context.Background(), time.Now().Add(time.Second*30))

	err = client.Records.Patch(ctx, updateRecord)
	if err != nil {
		return fmt.Errorf("Failed to update CloudFlare Record: %s", err)
	}

	return resourceCloudFlareRecordRead(d, meta)
}

func resourceCloudFlareRecordDelete(d *schema.ResourceData, meta interface{}) error {
	var (
		client = meta.(*cloudflare.Client)
		domain = d.Get("domain").(string)
		rName  = strings.Join([]string{d.Get("name").(string), domain}, ".")
	)

	zone, err := retrieveZone(client, domain)
	if err != nil {
		return err
	}

	record, err := retrieveRecord(client, zone, rName)
	if err != nil {
		return err
	}

	log.Printf("[INFO] Deleting CloudFlare Record: %s, %s", domain, d.Id())

	ctx, _ := context.WithDeadline(context.Background(), time.Now().Add(time.Second*30))

	err = client.Records.Delete(ctx, zone.ID, record.ID)
	if err != nil {
		return fmt.Errorf("Error deleting CloudFlare Record: %s", err)
	}

	return nil
}

func retrieveRecord(
	client *cloudflare.Client,
	zone *cloudflare.Zone,
	name string,
) (*cloudflare.Record, error) {
	ctx, _ := context.WithDeadline(context.Background(), time.Now().Add(time.Second*30))

	rs, err := client.Records.List(ctx, zone.ID)
	if err != nil {
		return nil, fmt.Errorf("Unable to retrieve records for (%s): %s", zone.Name, err)
	}

	var record *cloudflare.Record

	for _, r := range rs {
		if r.Name == name {
			record = r
		}
	}
	if record == nil {
		return nil, fmt.Errorf("Unable to find Cloudflare record %s", name)
	}

	return record, nil
}

func retrieveZone(client *cloudflare.Client, domain string) (*cloudflare.Zone, error) {
	ctx, _ := context.WithDeadline(context.Background(), time.Now().Add(time.Second*30))

	zs, err := client.Zones.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("Failed to fetch zone for %s: %s", domain, err)
	}

	var zone *cloudflare.Zone

	for _, z := range zs {
		if z.Name == domain {
			zone = z
		}
	}

	if zone == nil {
		return nil, fmt.Errorf("Failed to find zone for: %s", domain)
	}

	return zone, nil
}
