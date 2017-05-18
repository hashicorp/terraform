package openstack

import (
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/gophercloud/gophercloud/openstack/dns/v2/zones"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceDNSZoneV2() *schema.Resource {
	return &schema.Resource{
		Create: resourceDNSZoneV2Create,
		Read:   resourceDNSZoneV2Read,
		Update: resourceDNSZoneV2Update,
		Delete: resourceDNSZoneV2Delete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(10 * time.Minute),
			Delete: schema.DefaultTimeout(10 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			"region": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				DefaultFunc: schema.EnvDefaultFunc("OS_REGION_NAME", ""),
			},
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"email": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: false,
			},
			"type": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"attributes": &schema.Schema{
				Type:     schema.TypeMap,
				Optional: true,
				ForceNew: true,
			},
			"ttl": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				ForceNew: false,
			},
			"description": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: false,
			},
			"masters": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: false,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"value_specs": &schema.Schema{
				Type:     schema.TypeMap,
				Optional: true,
				ForceNew: true,
			},
		},
	}
}

func resourceDNSZoneV2Create(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	dnsClient, err := config.dnsV2Client(GetRegion(d))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack DNS client: %s", err)
	}

	mastersraw := d.Get("masters").(*schema.Set).List()
	masters := make([]string, len(mastersraw))
	for i, masterraw := range mastersraw {
		masters[i] = masterraw.(string)
	}

	attrsraw := d.Get("attributes").(map[string]interface{})
	attrs := make(map[string]string, len(attrsraw))
	for k, v := range attrsraw {
		attrs[k] = v.(string)
	}

	createOpts := ZoneCreateOpts{
		zones.CreateOpts{
			Name:        d.Get("name").(string),
			Type:        d.Get("type").(string),
			Attributes:  attrs,
			TTL:         d.Get("ttl").(int),
			Email:       d.Get("email").(string),
			Description: d.Get("description").(string),
			Masters:     masters,
		},
		MapValueSpecs(d),
	}

	log.Printf("[DEBUG] Create Options: %#v", createOpts)
	n, err := zones.Create(dnsClient, createOpts).Extract()
	if err != nil {
		return fmt.Errorf("Error creating OpenStack DNS zone: %s", err)
	}
	log.Printf("[INFO] Zone ID: %s", n.ID)

	d.SetId(n.ID)

	return resourceDNSZoneV2Read(d, meta)
}

func resourceDNSZoneV2Read(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	dnsClient, err := config.dnsV2Client(GetRegion(d))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack DNS client: %s", err)
	}

	n, err := zones.Get(dnsClient, d.Id()).Extract()
	if err != nil {
		return CheckDeleted(d, err, "zone")
	}

	log.Printf("[DEBUG] Retrieved Zone %s: %+v", d.Id(), n)

	d.Set("name", n.Name)
	d.Set("email", n.Email)
	d.Set("description", n.Description)
	d.Set("ttl", strconv.Itoa(n.TTL))
	d.Set("type", n.Type)
	d.Set("attributes", n.Attributes)
	d.Set("masters", n.Masters)
	d.Set("region", GetRegion(d))

	return nil
}

func resourceDNSZoneV2Update(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	dnsClient, err := config.dnsV2Client(GetRegion(d))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack DNS client: %s", err)
	}

	var updateOpts zones.UpdateOpts
	if d.HasChange("email") {
		updateOpts.Email = d.Get("email").(string)
	}
	if d.HasChange("ttl") {
		updateOpts.TTL = d.Get("ttl").(int)
	}
	if d.HasChange("masters") {
		updateOpts.Masters = d.Get("masters").([]string)
	}
	if d.HasChange("description") {
		updateOpts.Description = d.Get("description").(string)
	}

	log.Printf("[DEBUG] Updating Zone %s with options: %+v", d.Id(), updateOpts)

	_, err = zones.Update(dnsClient, d.Id(), updateOpts).Extract()
	if err != nil {
		return fmt.Errorf("Error updating OpenStack DNS Zone: %s", err)
	}

	return resourceDNSZoneV2Read(d, meta)
}

func resourceDNSZoneV2Delete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	dnsClient, err := config.dnsV2Client(GetRegion(d))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack DNS client: %s", err)
	}

	_, err = zones.Delete(dnsClient, d.Id()).Extract()
	if err != nil {
		return fmt.Errorf("Error deleting OpenStack DNS Zone: %s", err)
	}

	d.SetId("")
	return nil
}
