package icinga2

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/schema"
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

	config := meta.(*Config)
	hostname := d.Get("hostname").(string)
	address := d.Get("address").(string)
	checkCommand := d.Get("check_command").(string)

	attrs := make(map[string]interface{})
	vars := make(map[string]interface{})

	endpoint := fmt.Sprintf("v1/objects/hosts/%s", hostname)

	log.Printf("[DEBUG] Entering resourceIcinga2HostCreate : %s\n", hostname)

	vars = d.Get("vars").(map[string]interface{})

	for key, value := range vars {
		newkey := fmt.Sprintf("vars.%s", key)
		attrs[newkey] = value
	}
	attrs["address"] = address
	attrs["check_command"] = checkCommand

	attrsJSON, _ := json.Marshal(attrs)

	templates := d.Get("templates").([]interface{})
	templatesJSON, _ := json.Marshal(templates)

	theJSON := fmt.Sprintf("{ \"templates\": %s,  \"attrs\": %s }", templatesJSON, attrsJSON)

	var jsonStr = []byte(theJSON)

	httpCode, httpBody, _ := config.Client("PUT", endpoint, jsonStr)

	switch httpCode {
	case 200:
		d.SetId(hostname)
		log.Printf("[DEBUG] Exiting resourceIcinga2HostCreate : %s\n", hostname)
		return nil
	case 500:
		d.SetId(hostname)
		log.Printf("[DEBUG] Exiting resourceIcinga2HostCreate : %s\n", hostname)
		return nil
	default:
		r := httpBody.(map[string]interface{})["results"].([]interface{})[0].(map[string]interface{})
		log.Printf("[DEBUG] Exiting resourceIcinga2HostCreate : %s\n", hostname)
		return fmt.Errorf("[CREATE] %d : %s : %s", httpCode, endpoint, r["errors"])

	}

}

func resourceIcinga2HostRead(d *schema.ResourceData, meta interface{}) error {

	config := meta.(*Config)
	hostname := d.Get("hostname").(string)

	endpoint := fmt.Sprintf("v1/objects/hosts/%s", hostname)

	log.Printf("[DEBUG] Entering resourceIcinga2HostRead : %s\n", hostname)

	httpCode, httpBody, _ := config.Client("GET", endpoint, nil)

	switch httpCode {
	case 200:
		attrs := httpBody.(map[string]interface{})["results"].([]interface{})[0].(map[string]interface{})["attrs"].(map[string]interface{})
		if templates, ok := httpBody.(map[string]interface{})["results"].([]interface{})[0].(map[string]interface{})["templates"].([]interface{}); ok {
			d.Set("templates", templates)
		}
		d.Set("hostname", attrs["name"])
		d.Set("check_command", attrs["check_command"])
		d.Set("address", attrs["address"])
		d.Set("vars", attrs["vars"])
		d.SetId(hostname)
		log.Printf("[DEBUG] Exiting resourceIcinga2HostRead : %s\n", hostname)
		return nil
	case 404:
		d.SetId("")
		log.Printf("[DEBUG] Exiting resourceIcinga2HostRead : %s\n", hostname)
		return nil
	default:
		log.Printf("[DEBUG] Exiting resourceIcinga2HostRead : %s\n", hostname)
		return fmt.Errorf("[READ] Unexpected HTTP code : %d", httpCode)

	}

}

func resourceIcinga2HostUpdate(d *schema.ResourceData, meta interface{}) error {

	return nil
}

func resourceIcinga2HostDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	hostname := d.Get("hostname").(string)

	log.Printf("[DEBUG] Entering resourceIcinga2HostDelete : %s\n", hostname)

	endpoint := fmt.Sprintf("v1/objects/hosts/%s?cascade=1", hostname)
	httpCode, _, _ := config.Client("DELETE", endpoint, nil)

	switch httpCode {
	case 200:
		d.SetId("")
		log.Printf("[DEBUG] Exiting resourceIcinga2HostDelete : %s\n", hostname)
		return nil
	case 404:
		d.SetId("")
		log.Printf("[DEBUG] Exiting resourceIcinga2HostDelete : %s\n", hostname)
		return nil
	default:
		log.Printf("[DEBUG] Exiting resourceIcinga2HostDelete : %s\n", hostname)
		return fmt.Errorf("%d", httpCode)

	}

}
