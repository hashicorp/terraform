package rackspace

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/schema"
	osSecGroups "github.com/rackspace/gophercloud/openstack/networking/v2/extensions/security/groups"
	rsSecGroups "github.com/rackspace/gophercloud/rackspace/networking/v2/security/groups"
)

func resourceNetworkingSecGroup() *schema.Resource {
	return &schema.Resource{
		Create: resourceNetworkingSecGroupCreate,
		Read:   resourceNetworkingSecGroupRead,
		Delete: resourceNetworkingSecGroupDelete,

		Schema: map[string]*schema.Schema{
			"region": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				DefaultFunc: envDefaultFunc("RS_REGION_NAME"),
			},
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: false,
			},
			"description": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: false,
			},
		},
	}
}

func resourceNetworkingSecGroupCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	networkingClient, err := config.networkingClient(d.Get("region").(string))
	if err != nil {
		return fmt.Errorf("Error creating Rackspace networking client: %s", err)
	}

	createOpts := osSecGroups.CreateOpts{
		Name:        d.Get("name").(string),
		Description: d.Get("description").(string),
	}

	log.Printf("[DEBUG] Create Options: %#v", createOpts)
	sg, err := rsSecGroups.Create(networkingClient, createOpts).Extract()
	if err != nil {
		return fmt.Errorf("Error creating Rackspace security group: %s", err)
	}

	d.SetId(sg.ID)

	return resourceNetworkingSecGroupRead(d, meta)
}

func resourceNetworkingSecGroupRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	networkingClient, err := config.networkingClient(d.Get("region").(string))
	if err != nil {
		return fmt.Errorf("Error creating Rackspace networking client: %s", err)
	}

	sg, err := rsSecGroups.Get(networkingClient, d.Id()).Extract()
	if err != nil {
		return CheckDeleted(d, err, "security group")
	}

	d.Set("name", sg.Name)
	d.Set("description", sg.Description)

	return nil
}

func resourceNetworkingSecGroupDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	networkingClient, err := config.networkingClient(d.Get("region").(string))
	if err != nil {
		return fmt.Errorf("Error creating Rackspace networking client: %s", err)
	}

	err = rsSecGroups.Delete(networkingClient, d.Id()).ExtractErr()
	if err != nil {
		return fmt.Errorf("Error deleting Rackspace security group: %s", err)
	}
	d.SetId("")
	return nil
}
