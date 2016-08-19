package icinga2

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/schema"
)

func resourceIcinga2Checkcommand() *schema.Resource {

	return &schema.Resource{
		Create: resourceIcinga2CheckcommandCreate,
		Read:   resourceIcinga2CheckcommandRead,
		Delete: resourceIcinga2CheckcommandDelete,
		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: "Name",
				ForceNew:    true,
			},
			"command": &schema.Schema{
				Type:     schema.TypeList,
				Required: true,
				ForceNew: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"templates": &schema.Schema{
				Type:     schema.TypeList,
				Required: true,
				ForceNew: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"arguments": &schema.Schema{
				Type:     schema.TypeMap,
				Optional: true,
				ForceNew: true,
			},
		},
	}
}

func resourceIcinga2CheckcommandCreate(d *schema.ResourceData, meta interface{}) error {

	config := meta.(*Config)
	name := d.Get("name").(string)

	endpoint := fmt.Sprintf("v1/objects/checkcommands/%s", name)

	log.Printf("[DEBUG] Entering resourceIcinga2CheckcommandCreate : %s\n", name)

	commandJSON, _ := json.Marshal(d.Get("command").([]interface{}))
	templateJSON, _ := json.Marshal(d.Get("templates").([]interface{}))
	argsJSON, _ := json.Marshal(d.Get("arguments").(map[string]interface{}))

	theJSON := fmt.Sprintf("{ \"templates\": %s, \"attrs\": { \"command\": %s, \"arguments\" : %s } }",
		templateJSON, commandJSON, argsJSON)

	jsonStr := []byte(theJSON)
	httpCode, httpBody, _ := config.Client("PUT", endpoint, jsonStr)

	switch httpCode {
	case 200:
		d.SetId(name)
		log.Printf("[DEBUG] Exiting resourceIcinga2CheckcommandCreate : %s\n", name)
		return nil
	case 500:
		d.SetId(name)
		log.Printf("[DEBUG] Exiting resourceIcinga2CheckcommandCreate : %s\n", name)
		return nil
	default:
		r := httpBody.(map[string]interface{})["results"].([]interface{})[0].(map[string]interface{})
		log.Printf("[DEBUG] Exiting resourceIcinga2CheckcommandCreate : %s\n", name)
		return fmt.Errorf("[CREATE] %d : %s : %s", httpCode, endpoint, r["errors"])
	}

}

func resourceIcinga2CheckcommandRead(d *schema.ResourceData, meta interface{}) error {

	config := meta.(*Config)
	name := d.Get("name").(string)

	endpoint := fmt.Sprintf("v1/objects/checkcommands/%s", name)

	log.Printf("[DEBUG] Entering resourceIcinga2CheckcommandRead : %s\n", name)

	httpCode, httpBody, _ := config.Client("GET", endpoint, nil)

	switch httpCode {
	case 200:
		attrs := httpBody.(map[string]interface{})["results"].([]interface{})[0].(map[string]interface{})["attrs"].(map[string]interface{})
		if templates, ok := httpBody.(map[string]interface{})["results"].([]interface{})[0].(map[string]interface{})["templates"].([]interface{}); ok {
			d.Set("templates", templates)
		}
		// Add code to build the arguments map
		d.Set("name", attrs["name"])
		d.Set("command", attrs["command"])
		d.Set("arguments", attrs["arguments"])
		d.SetId(name)
		return nil
	case 404:
		d.SetId("")
		log.Printf("[DEBUG] Exiting resourceIcinga2CheckcommandRead : %s", name)
		return nil
	default:
		log.Printf("[DEBUG] Exiting resourceIcinga2CheckcommandRead : %s", name)
		return fmt.Errorf("[READ] Unexpected HTTP Code : %d", httpCode)
	}
}

func resourceIcinga2CheckcommandUpdate(d *schema.ResourceData, meta interface{}) error {

	return nil
}

func resourceIcinga2CheckcommandDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	name := d.Get("name").(string)

	log.Printf("[DEBUG] Entering resourceIcinga2CheckcommandDelete : %s\n", name)

	endpoint := fmt.Sprintf("v1/objects/hosts/%s?cascade=1", name)
	httpCode, _, _ := config.Client("DELETE", endpoint, nil)

	switch httpCode {
	case 200:
		d.SetId("")
		log.Printf("[DEBUG] Exiting resourceIcinga2CheckcommandDelete : %s\n", name)
		return nil
	case 404:
		d.SetId("")
		log.Printf("[DEBUG] Exiting resourceIcinga2CheckcommandDelete : %s\n", name)
		return nil
	default:
		log.Printf("[DEBUG] Exiting resourceIcinga2CheckcommandDelete : %s\n", name)
		return fmt.Errorf("%d", httpCode)

	}

}
