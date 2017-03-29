package opc

import (
	"fmt"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/oracle/terraform-provider-compute/sdk/compute"
	"log"
)

func resourceIPReservation() *schema.Resource {
	return &schema.Resource{
		Create: resourceIPReservationCreate,
		Read:   resourceIPReservationRead,
		Delete: resourceIPReservationDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"permanent": &schema.Schema{
				Type:     schema.TypeBool,
				Required: true,
				ForceNew: true,
			},

			"parentpool": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"tags": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			
			"ip": &schema.Schema{
				Type:     schema.TypeString,
				Optional: false,
				Computed: true,
			},
		},
	}
}

func resourceIPReservationCreate(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] Resource state: %#v", d.State())

	parentpool, permanent, tags := getIPReservationResourceData(d)

	log.Printf("[DEBUG] Creating ip reservation from parentpool %s with tags=%s",
		parentpool, tags)

	client := meta.(*OPCClient).IPReservations()
	info, err := client.CreateIPReservation(parentpool, permanent, tags)
	if err != nil {
		return fmt.Errorf("Error creating ip reservation from parentpool %s with tags=%s: %s",
			parentpool, tags, err)
	}

	d.SetId(info.Name)
	updateIPReservationResourceData(d, info)
	return nil
}

func updateIPReservationResourceData(d *schema.ResourceData, info *compute.IPReservationInfo) {
	d.Set("name", info.Name)
	d.Set("parentpool", info.ParentPool)
	d.Set("permanent", info.Permanent)
	d.Set("tags", info.Tags)
	d.Set("ip", info.IP)
}

func resourceIPReservationRead(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] Resource state: %#v", d.State())
	client := meta.(*OPCClient).IPReservations()
	name := d.Get("name").(string)

	log.Printf("[DEBUG] Reading state of ip reservation %s", name)
	result, err := client.GetIPReservation(name)
	if err != nil {
		// IP Reservation does not exist
		if compute.WasNotFoundError(err) {
			d.SetId("")
			return nil
		}
		return fmt.Errorf("Error reading ip reservation %s: %s", name, err)
	}

	log.Printf("[DEBUG] Read state of ip reservation %s: %#v", name, result)
	updateIPReservationResourceData(d, result)
	return nil
}

func getIPReservationResourceData(d *schema.ResourceData) (string, bool, []string) {
	tagdata := d.Get("tags").([]interface{})
	tags := make([]string, len(tagdata))
	for i, tag := range tagdata {
		tags[i] = tag.(string)
	}
	return d.Get("parentpool").(string),
		d.Get("permanent").(bool),
		tags
}

func resourceIPReservationDelete(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] Resource state: %#v", d.State())
	client := meta.(*OPCClient).IPReservations()
	name := d.Get("name").(string)

	log.Printf("[DEBUG] Deleting ip reservation %s", name)

	if err := client.DeleteIPReservation(name); err != nil {
		return fmt.Errorf("Error deleting ip reservation %s", name)
	}
	return nil
}
