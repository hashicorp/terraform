package icinga2

import (
	"fmt"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/lrsmith/go-icinga2-api/iapi"
)

func resourceIcinga2Hostgroup() *schema.Resource {

	return &schema.Resource{
		Create: resourceIcinga2HostgroupCreate,
		Read:   resourceIcinga2HostgroupRead,
		Delete: resourceIcinga2HostgroupDelete,
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
		},
	}
}

func resourceIcinga2HostgroupCreate(d *schema.ResourceData, meta interface{}) error {

	client := meta.(*iapi.Server)

	name := d.Get("name").(string)
	displayName := d.Get("display_name").(string)

	hostgroups, err := client.CreateHostgroup(name, displayName)
	if err != nil {
		return err
	}

	found := false
	for _, hostgroup := range hostgroups {
		if hostgroup.Name == name {
			d.SetId(name)
			found = true
		}
	}

	if !found {
		return fmt.Errorf("Failed to Create Hostgroup %s : %s", name, err)
	}

	return nil
}

func resourceIcinga2HostgroupRead(d *schema.ResourceData, meta interface{}) error {

	client := meta.(*iapi.Server)
	name := d.Get("name").(string)

	hostgroups, err := client.GetHostgroup(name)
	if err != nil {
		return err
	}

	found := false
	for _, hostgroup := range hostgroups {
		if hostgroup.Name == name {
			d.SetId(name)
			d.Set("display_name", hostgroup.Attrs.DisplayName)
			found = true
		}
	}

	if !found {
		return fmt.Errorf("Failed to Read Hostgroup %s : %s", name, err)
	}

	return nil
}

func resourceIcinga2HostgroupDelete(d *schema.ResourceData, meta interface{}) error {

	client := meta.(*iapi.Server)
	name := d.Get("name").(string)

	err := client.DeleteHostgroup(name)
	if err != nil {
		return fmt.Errorf("Failed to Delete Hostgroup %s : %s", name, err)
	}

	return nil

}
