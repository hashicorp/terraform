package icinga2

import (
	"fmt"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/lrsmith/go-icinga2-api/iapi"
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
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
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

	client := meta.(*iapi.Server)

	name := d.Get("name").(string)
	command := d.Get("command").(string)

	arguments := make(map[string]string)
	iterator := d.Get("arguments").(map[string]interface{})

	for key, value := range iterator {
		arguments[key] = value.(string)
	}

	checkcommands, err := client.CreateCheckcommand(name, command, arguments)
	if err != nil {
		return err
	}

	found := false
	for _, checkcommand := range checkcommands {
		if checkcommand.Name == name {
			d.SetId(name)
			found = true
		}
	}

	if !found {
		return fmt.Errorf("Failed to create Checkcommand %s : %s", name, err)
	}

	return nil
}

func resourceIcinga2CheckcommandRead(d *schema.ResourceData, meta interface{}) error {

	client := meta.(*iapi.Server)

	name := d.Get("name").(string)

	checkcommands, err := client.GetCheckcommand(name)
	if err != nil {
		return err
	}

	found := false
	for _, checkcommand := range checkcommands {
		if checkcommand.Name == name {
			d.SetId(name)
			d.Set("command", checkcommand.Attrs.Command[0])
			d.Set("Templates", checkcommand.Attrs.Templates)
			d.Set("arguments", checkcommand.Attrs.Arguments)
			found = true
		}
	}

	if !found {
		return fmt.Errorf("Failed to Read Checkcommand %s : %s", name, err)
	}

	return nil
}

func resourceIcinga2CheckcommandDelete(d *schema.ResourceData, meta interface{}) error {

	client := meta.(*iapi.Server)

	name := d.Get("name").(string)

	err := client.DeleteCheckcommand(name)
	if err != nil {
		return fmt.Errorf("Failed to Delete Checkcommand %s : %s", name, err)
	}

	return nil
}
