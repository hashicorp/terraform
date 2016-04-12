package cobbler

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/schema"
	cobbler "github.com/jtopjian/cobblerclient"
)

func resourceKickstartFile() *schema.Resource {
	return &schema.Resource{
		Create: resourceKickstartFileCreate,
		Read:   resourceKickstartFileRead,
		Update: resourceKickstartFileUpdate,
		Delete: resourceKickstartFileDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"body": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
		},
	}
}

func resourceKickstartFileCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	ks := cobbler.KickstartFile{
		Name: d.Get("name").(string),
		Body: d.Get("body").(string),
	}

	log.Printf("[DEBUG] Cobbler KickstartFile: Create Options: %#v", ks)

	if err := config.cobblerClient.CreateKickstartFile(ks); err != nil {
		return fmt.Errorf("Cobbler KickstartFile: Error Creating: %s", err)
	}

	d.SetId(ks.Name)

	return resourceKickstartFileRead(d, meta)
}

func resourceKickstartFileRead(d *schema.ResourceData, meta interface{}) error {
	// Since all attributes are required and not computed,
	// there's no reason to read.
	return nil
}

func resourceKickstartFileUpdate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	ks := cobbler.KickstartFile{
		Name: d.Id(),
		Body: d.Get("body").(string),
	}

	log.Printf("[DEBUG] Cobbler KickstartFile: Updating Kickstart (%s) with options: %+v", d.Id(), ks)

	if err := config.cobblerClient.CreateKickstartFile(ks); err != nil {
		return fmt.Errorf("Cobbler KickstartFile: Error Updating (%s): %s", d.Id(), err)
	}

	return resourceKickstartFileRead(d, meta)
}

func resourceKickstartFileDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	if err := config.cobblerClient.DeleteKickstartFile(d.Id()); err != nil {
		return fmt.Errorf("Cobbler KickstartFile: Error Deleting (%s): %s", d.Id(), err)
	}

	return nil
}
