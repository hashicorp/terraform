package icinga2

import (
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/lrsmith/go-icinga2-api/iapi"
)

func resourceIcinga2CheckCommand() *schema.Resource {

	return &schema.Resource{
		Create: resourceIcinga2CheckCommandCreate,
		Read:   resourceIcinga2CheckCommandRead,
		Delete: resourceIcinga2CheckCommandDelete,
		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: "Name",
				ForceNew:    true,
			},
			"command": &schema.Schema{
				//Type:     schema.TypeList,
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
				//Elem:     &schema.Schema{Type: schema.TypeString},
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

func resourceIcinga2CheckCommandCreate(d *schema.ResourceData, meta interface{}) error {

	client := meta.(*iapi.Server)

	name := d.Get("name").(string)
	command := d.Get("command").(string)

	arguments := make(map[string]string)
	iterator := d.Get("arguments").(map[string]interface{})
	for key, value := range iterator {
		arguments[key] = value.(string)
	}

	checkcommands, err := client.CreateCheckCommand(name, command, arguments)
	if err != nil {
		return err
	}

	for _, checkcommand := range checkcommands {
		if checkcommand.Name == name {
			d.SetId(name)
		}
	}

	return nil
}

func resourceIcinga2CheckCommandRead(d *schema.ResourceData, meta interface{}) error {

	client := meta.(*iapi.Server)

	name := d.Get("name").(string)

	checkcommands, err := client.GetCheckCommand(name)
	if err != nil {
		return err
	}

	for _, checkcommand := range checkcommands {
		if checkcommand.Name == name {
			d.SetId(name)
		}
	}

	return nil
}

func resourceIcinga2CheckCommandUpdate(d *schema.ResourceData, meta interface{}) error {

	return nil
}

func resourceIcinga2CheckCommandDelete(d *schema.ResourceData, meta interface{}) error {

	client := meta.(*iapi.Server)

	name := d.Get("name").(string)

	return client.DeleteCheckCommand(name)

}
