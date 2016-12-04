package icinga2

import (
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/lrsmith/go-icinga2-api/iapi"
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

	client := meta.(*iapi.Server)

	groupName := d.Get("name").(string)
	displayName := d.Get("display_name").(string)

	hostgroups, err := client.CreateHostgroup(groupName, displayName)
	if err != nil {
		return err
	}

	for _, hostgroup := range hostgroups {
		if hostgroup.Name == groupName {
			d.SetId(groupName)
		}
	}

	return nil

}

func resourceIcinga2HostGroupRead(d *schema.ResourceData, meta interface{}) error {

	client := meta.(*iapi.Server)
	groupName := d.Get("name").(string)

	_, err := client.GetHostgroup(groupName)
	if err != nil {
		return err
	}

	d.SetId(groupName)
	return nil

}

func resourceIcinga2HostGroupUpdate(d *schema.ResourceData, meta interface{}) error {
	return nil
}

func resourceIcinga2HostGroupDelete(d *schema.ResourceData, meta interface{}) error {

	client := meta.(*iapi.Server)
	groupName := d.Get("name").(string)

	err := client.DeleteHostgroup(groupName)
	if err != nil {
		return err
	}

	return nil

}
