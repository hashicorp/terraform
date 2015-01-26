package openstack

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/rackspace/gophercloud"
	"github.com/rackspace/gophercloud/openstack"
	"github.com/rackspace/gophercloud/openstack/compute/v2/extensions/secgroups"
)

func resourceComputeSecGroup() *schema.Resource {
	return &schema.Resource{
		Create: resourceComputeSecGroupCreate,
		Read:   resourceComputeSecGroupRead,
		Update: resourceComputeSecGroupUpdate,
		Delete: resourceComputeSecGroupDelete,

		Schema: map[string]*schema.Schema{
			"region": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				DefaultFunc: envDefaultFunc("OS_REGION_NAME"),
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

func resourceComputeSecGroupCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	computeClient, err := openstack.NewComputeV2(config.osClient, gophercloud.EndpointOpts{
		Region: d.Get("region").(string),
	})
	if err != nil {
		return fmt.Errorf("Error creating OpenStack compute client: %s", err)
	}

	createOpts := secgroups.CreateOpts{
		Name:        d.Get("name").(string),
		Description: d.Get("description").(string),
	}

	sg, err := secgroups.Create(computeClient, createOpts).Extract()
	if err != nil {
		return fmt.Errorf("Error creating OpenStack security group: %s", err)
	}

	d.SetId(sg.ID)

	return resourceComputeSecGroupRead(d, meta)
}

func resourceComputeSecGroupRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	computeClient, err := openstack.NewComputeV2(config.osClient, gophercloud.EndpointOpts{
		Region: d.Get("region").(string),
	})
	if err != nil {
		return fmt.Errorf("Error creating OpenStack compute client: %s", err)
	}

	sg, err := secgroups.Get(computeClient, d.Id()).Extract()
	if err != nil {
		return fmt.Errorf("Error retrieving OpenStack security group: %s", err)
	}

	d.Set("name", sg.Name)
	d.Set("description", sg.Description)

	return nil
}

func resourceComputeSecGroupUpdate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	computeClient, err := openstack.NewComputeV2(config.osClient, gophercloud.EndpointOpts{
		Region: d.Get("region").(string),
	})
	if err != nil {
		return fmt.Errorf("Error creating OpenStack compute client: %s", err)
	}

	var updateOpts secgroups.UpdateOpts
	if d.HasChange("name") {
		updateOpts.Name = d.Get("name").(string)
	}
	if d.HasChange("description") {
		updateOpts.Description = d.Get("description").(string)
	}

	log.Printf("[DEBUG] Updating Security Group (%s) with options: %+v", d.Id(), updateOpts)

	_, err = secgroups.Update(computeClient, d.Id(), updateOpts).Extract()
	if err != nil {
		return fmt.Errorf("Error updating OpenStack security group (%s): %s", d.Id(), err)
	}

	return resourceComputeSecGroupRead(d, meta)
}

func resourceComputeSecGroupDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	computeClient, err := openstack.NewComputeV2(config.osClient, gophercloud.EndpointOpts{
		Region: d.Get("region").(string),
	})
	if err != nil {
		return fmt.Errorf("Error creating OpenStack compute client: %s", err)
	}

	err = secgroups.Delete(computeClient, d.Id()).ExtractErr()
	if err != nil {
		return fmt.Errorf("Error deleting OpenStack security group: %s", err)
	}
	d.SetId("")
	return nil
}
