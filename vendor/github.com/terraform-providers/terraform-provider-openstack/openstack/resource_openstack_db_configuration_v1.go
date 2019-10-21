package openstack

import (
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/db/v1/configurations"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceDatabaseConfigurationV1() *schema.Resource {
	return &schema.Resource{
		Create: resourceDatabaseConfigurationV1Create,
		Read:   resourceDatabaseConfigurationV1Read,
		Delete: resourceDatabaseConfigurationV1Delete,

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(10 * time.Minute),
			Delete: schema.DefaultTimeout(10 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			"region": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				DefaultFunc: schema.EnvDefaultFunc("OS_REGION_NAME", ""),
			},
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"description": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"datastore": {
				Type:     schema.TypeList,
				Required: true,
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"version": {
							Type:     schema.TypeString,
							Required: true,
							ForceNew: true,
						},
						"type": {
							Type:     schema.TypeString,
							Required: true,
							ForceNew: true,
						},
					},
				},
				MaxItems: 1,
			},
			"configuration": {
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:     schema.TypeString,
							Required: true,
							ForceNew: true,
						},
						"value": {
							Type:     schema.TypeString,
							Required: true,
							ForceNew: true,
						},
					},
				},
			},
		},
	}
}

func resourceDatabaseConfigurationV1Create(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	databaseV1Client, err := config.databaseV1Client(GetRegion(d, config))
	if err != nil {
		return fmt.Errorf("Error creating cloud database client: %s", err)
	}

	var datastore configurations.DatastoreOpts
	if p, ok := d.GetOk("datastore"); ok {
		pV := (p.([]interface{}))[0].(map[string]interface{})

		datastore = configurations.DatastoreOpts{
			Version: pV["version"].(string),
			Type:    pV["type"].(string),
		}
	}

	createOpts := &configurations.CreateOpts{
		Name:        d.Get("name").(string),
		Description: d.Get("description").(string),
	}

	createOpts.Datastore = &datastore

	values := make(map[string]interface{})
	if p, ok := d.GetOk("configuration"); ok {

		listSlice, _ := p.([]interface{})
		for _, d := range listSlice {
			if z, ok := d.(map[string]interface{}); ok {
				name := z["name"].(string)
				value := z["value"].(interface{})

				// check if value can be converted into int
				if valueInt, err := strconv.Atoi(value.(string)); err == nil {
					value = valueInt
				}

				values[name] = value
			}
		}
	}

	createOpts.Values = values

	log.Printf("[DEBUG] Create Options: %#v", createOpts)
	cgroup, err := configurations.Create(databaseV1Client, createOpts).Extract()

	if err != nil {
		return fmt.Errorf("Error creating cloud database configuration: %s", err)
	}
	log.Printf("[INFO] configuration ID: %s", cgroup.ID)

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"BUILD"},
		Target:     []string{"ACTIVE"},
		Refresh:    DatabaseConfigurationV1StateRefreshFunc(databaseV1Client, cgroup.ID),
		Timeout:    d.Timeout(schema.TimeoutCreate),
		Delay:      10 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	_, err = stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf(
			"Error waiting for configuration (%s) to become ready: %s",
			cgroup.ID, err)
	}

	// Store the ID now
	d.SetId(cgroup.ID)

	return resourceDatabaseConfigurationV1Read(d, meta)
}

func resourceDatabaseConfigurationV1Read(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	databaseV1Client, err := config.databaseV1Client(GetRegion(d, config))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack cloud database client: %s", err)
	}

	cgroup, err := configurations.Get(databaseV1Client, d.Id()).Extract()
	if err != nil {
		return CheckDeleted(d, err, "configuration")
	}

	log.Printf("[DEBUG] Retrieved configuration %s: %+v", d.Id(), cgroup)

	d.Set("name", cgroup.Name)
	d.Set("description", cgroup.Description)
	d.Set("region", GetRegion(d, config))

	return nil
}

func resourceDatabaseConfigurationV1Delete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	databaseV1Client, err := config.databaseV1Client(GetRegion(d, config))
	if err != nil {
		return fmt.Errorf("Error creating RS cloud instance client: %s", err)
	}

	log.Printf("[DEBUG] Deleting cloud database configuration %s", d.Id())
	err = configurations.Delete(databaseV1Client, d.Id()).ExtractErr()
	if err != nil {
		return fmt.Errorf("Error deleting cloud configuration: %s", err)
	}

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"ACTIVE", "SHUTOFF"},
		Target:     []string{"DELETED"},
		Refresh:    DatabaseConfigurationV1StateRefreshFunc(databaseV1Client, d.Id()),
		Timeout:    d.Timeout(schema.TimeoutDelete),
		Delay:      10 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	_, err = stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf(
			"Error waiting for configuration (%s) to delete: %s",
			d.Id(), err)
	}

	d.SetId("")
	return nil
}

// DatabaseConfigurationV1StateRefreshFunc returns a resource.StateRefreshFunc that is used to watch
// an cloud database instance.
func DatabaseConfigurationV1StateRefreshFunc(client *gophercloud.ServiceClient, cgroupID string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		i, err := configurations.Get(client, cgroupID).Extract()
		if err != nil {
			if _, ok := err.(gophercloud.ErrDefault404); ok {
				return i, "DELETED", nil
			}
			return nil, "", err
		}

		return i, "ACTIVE", nil
	}
}
