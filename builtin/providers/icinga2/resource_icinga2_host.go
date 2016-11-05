package icinga2

import (
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/lrsmith/go-icinga2-api/iapi"
)

func resourceIcinga2Host() *schema.Resource {

	return &schema.Resource{
		Create: resourceIcinga2HostCreate,
		Read:   resourceIcinga2HostRead,
		Delete: resourceIcinga2HostDelete,
		Schema: map[string]*schema.Schema{
			"hostname": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: "Hostname",
				ForceNew:    true,
			},
			"address": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"check_command": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"templates": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"vars": &schema.Schema{
				Type:     schema.TypeMap,
				Optional: true,
				ForceNew: true,
			},
		},
	}
}

func resourceIcinga2HostCreate(d *schema.ResourceData, meta interface{}) error {

	client := meta.(*iapi.Server)

	hostname := d.Get("hostname").(string)
	address := d.Get("address").(string)
	checkCommand := d.Get("check_command").(string)

	vars := make(map[string]string)

	// Normalize from map[string]interface{} to map[string]string
	iterator := d.Get("vars").(map[string]interface{})
	for key, value := range iterator {
		vars[key] = value.(string)
	}

	// Call CreateHost with normalized data
	err := client.CreateHost(hostname, address, checkCommand, vars)
	if err != nil {
		return err
	}

	d.SetId(hostname)
	return nil

}

func resourceIcinga2HostRead(d *schema.ResourceData, meta interface{}) error {

	client := meta.(*iapi.Server)

	hostname := d.Get("hostname").(string)

	_, err := client.GetHost(hostname)
	if err != nil {
		return err
	}

	return nil
}

func resourceIcinga2HostUpdate(d *schema.ResourceData, meta interface{}) error {

	return nil

}

func resourceIcinga2HostDelete(d *schema.ResourceData, meta interface{}) error {

	client := meta.(*iapi.Server)
	hostname := d.Get("hostname").(string)

	err := client.DeleteHost(hostname)
	if err != nil {
		return err
	}

	return nil

}
