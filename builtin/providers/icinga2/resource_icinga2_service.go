package icinga2

import (
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/lrsmith/go-icinga2-api/iapi"
)

func resourceIcinga2Service() *schema.Resource {

	return &schema.Resource{
		Create: resourceIcinga2ServiceCreate,
		Read:   resourceIcinga2ServiceRead,
		Delete: resourceIcinga2ServiceDelete,
		Schema: map[string]*schema.Schema{
			"servicename": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: "ServiceName",
				ForceNew:    true,
			},
			"hostname": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: "Hostname",
				ForceNew:    true,
			},
			"checkcommand": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: "CheckCommand",
				ForceNew:    true,
			},
		},
	}
}

func resourceIcinga2ServiceCreate(d *schema.ResourceData, meta interface{}) error {

	client := meta.(*iapi.Server)

	hostname := d.Get("hostname").(string)
	servicename := d.Get("servicename").(string)
	checkcommand := d.Get("checkcommand").(string)

	// Call CreateService with normalized data
	services, err := client.CreateService(servicename, hostname, checkcommand)
	if err != nil {
		return err
	}

	for _, service := range services {
		if service.Name == hostname+"!"+servicename {
			d.SetId(hostname + "!" + servicename)
		}
	}

	return nil

}

func resourceIcinga2ServiceRead(d *schema.ResourceData, meta interface{}) error {

	client := meta.(*iapi.Server)

	hostname := d.Get("hostname").(string)
	servicename := d.Get("servicename").(string)

	services, err := client.GetService(servicename, hostname)
	if err != nil {
		return err
	}

	for _, service := range services {
		if service.Name == hostname+"!"+servicename {
			d.SetId(hostname + "!" + servicename)
		}
	}

	return nil
}

func resourceIcinga2ServiceUpdate(d *schema.ResourceData, meta interface{}) error {

	return nil

}

func resourceIcinga2ServiceDelete(d *schema.ResourceData, meta interface{}) error {

	client := meta.(*iapi.Server)

	hostname := d.Get("hostname").(string)
	servicename := d.Get("servicename").(string)

	return client.DeleteService(servicename, hostname)

}
