package icinga2

import (
	"fmt"

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
	hosts, err := client.CreateHost(hostname, address, checkCommand, vars)
	if err != nil {
		return err
	}

	found := false
	for _, host := range hosts {
		if host.Name == hostname {
			d.SetId(hostname)
			found = true
		}
	}

	if !found {
		return fmt.Errorf("Failed to Create Host %s : %s", hostname, err)
	}

	return nil
}

func resourceIcinga2HostRead(d *schema.ResourceData, meta interface{}) error {

	client := meta.(*iapi.Server)

	hostname := d.Get("hostname").(string)

	hosts, err := client.GetHost(hostname)
	if err != nil {
		return err
	}

	found := false
	for _, host := range hosts {
		if host.Name == hostname {
			d.SetId(hostname)
			d.Set("hostname", host.Name)
			d.Set("address", host.Attrs.Address)
			d.Set("check_command", host.Attrs.CheckCommand)
			d.Set("vars", host.Attrs.Vars)
			found = true
		}
	}

	if !found {
		return fmt.Errorf("Failed to Read Host %s : %s", hostname, err)
	}

	return nil
}

func resourceIcinga2HostDelete(d *schema.ResourceData, meta interface{}) error {

	client := meta.(*iapi.Server)
	hostname := d.Get("hostname").(string)

	err := client.DeleteHost(hostname)
	if err != nil {
		return fmt.Errorf("Failed to Delete Host %s : %s", hostname, err)
	}

	return nil

}
