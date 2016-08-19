package icinga2

import (
	"fmt"

	"github.com/hashicorp/terraform/helper/schema"
)

func resourceIcinga2HostGroup() *schema.Resource {

	return &schema.Resource{
		Create: resourceIcinga2HostGroupCreate,
		Read:   resourceIcinga2HostGroupRead,
		Delete: resourceIcinga2HostGroupDelete,
		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: "name",
				ForceNew:    true,
			},
			"display_name": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: "Display name of Host Group",
				ForceNew:    true,
			},
			"groups": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
		},
	}
}

func resourceIcinga2HostGroupCreate(d *schema.ResourceData, meta interface{}) error {

	config := meta.(*Config)
	name := d.Get("name").(string)
	DisplayName := d.Get("display_name").(string)
	//groups := d.Get("groups").([]interface{})

	endpoint := fmt.Sprintf("v1/objects/hostgroups/%s", name)
	jsonData := []byte(fmt.Sprintf("{ \"attrs\": { \"display_name\":  \"%s\" } }", DisplayName))

	httpCode, httpBody, _ := config.Client("PUT", endpoint, jsonData)

	switch httpCode {
	case 200:
		d.SetId(name)
		return nil
	case 500:
		d.SetId(name)
		return nil
	default:
		r := httpBody.(map[string]interface{})["results"].([]interface{})[0].(map[string]interface{})
		return fmt.Errorf("[CREATE HOSTGROUP] %d : %s", httpCode, r["errors"])

	}

}

func resourceIcinga2HostGroupRead(d *schema.ResourceData, meta interface{}) error {

	config := meta.(*Config)
	name := d.Get("name").(string)

	endpoint := fmt.Sprintf("v1/objects/hostgroups/%s", name)

	httpCode, httpBody, _ := config.Client("GET", endpoint, nil)

	switch httpCode {
	case 200:
		attrs := httpBody.(map[string]interface{})["results"].([]interface{})[0].(map[string]interface{})["attrs"].(map[string]interface{})
		d.Set("name", attrs["name"])
		d.Set("display_name", attrs["display_name"])
		return nil
	case 404:
		d.SetId("")
		return nil
	default:
		return fmt.Errorf("[READ HOSTGROUPS ] Unexpected HTTP code : %d", httpCode)
	}

}

func resourceIcinga2HostGroupDelete(d *schema.ResourceData, meta interface{}) error {

	config := meta.(*Config)
	name := d.Get("name").(string)

	endpoint := fmt.Sprintf("v1/objects/hostgroups/%s", name)
	httpCode, _, _ := config.Client("DELETE", endpoint, nil)

	switch httpCode {
	case 200:
		d.SetId("")
		return nil
	case 404:
		d.SetId("")
		return nil
	default:
		return fmt.Errorf("%d", httpCode)

	}

}
