package openstack

import (
	"fmt"
	"log"
	"time"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/db/v1/databases"
	"github.com/gophercloud/gophercloud/openstack/db/v1/instances"
	"github.com/gophercloud/gophercloud/openstack/db/v1/users"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceDatabaseInstanceV1() *schema.Resource {
	return &schema.Resource{
		Create: resourceDatabaseInstanceV1Create,
		Read:   resourceDatabaseInstanceV1Read,
		Delete: resourceDatabaseInstanceV1Delete,
		Update: resourceDatabaseInstanceUpdate,

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(30 * time.Minute),
			Delete: schema.DefaultTimeout(30 * time.Minute),
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
			"flavor_id": {
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Computed:    true,
				DefaultFunc: schema.EnvDefaultFunc("OS_FLAVOR_ID", nil),
			},
			"size": {
				Type:     schema.TypeInt,
				Required: true,
				ForceNew: true,
			},
			"datastore": {
				Type:     schema.TypeList,
				Required: true,
				ForceNew: true,
				MaxItems: 1,
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
			},
			"network": {
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"uuid": {
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},
						"port": {
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},
						"fixed_ip_v4": {
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},
						"fixed_ip_v6": {
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},
					},
				},
			},
			"database": {
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
						"charset": {
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},
						"collate": {
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},
					},
				},
			},
			"user": {
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
						"password": {
							Type:      schema.TypeString,
							Optional:  true,
							ForceNew:  true,
							Sensitive: true,
						},
						"host": {
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},
						"databases": {
							Type:     schema.TypeSet,
							Optional: true,
							ForceNew: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},
					},
				},
			},
			"configuration_id": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: false,
				ForceNew: false,
			},
		},
	}
}

func resourceDatabaseInstanceV1Create(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	databaseV1Client, err := config.databaseV1Client(GetRegion(d, config))
	if err != nil {
		return fmt.Errorf("Error creating database client: %s", err)
	}

	createOpts := &instances.CreateOpts{
		FlavorRef: d.Get("flavor_id").(string),
		Name:      d.Get("name").(string),
		Size:      d.Get("size").(int),
	}

	var datastore instances.DatastoreOpts
	if v, ok := d.GetOk("datastore"); ok {
		if v, ok := v.([]interface{}); ok && len(v) > 0 {
			ds := v[0].(map[string]interface{})
			datastore = instances.DatastoreOpts{
				Version: ds["version"].(string),
				Type:    ds["type"].(string),
			}
			createOpts.Datastore = &datastore
		}
	}

	// networks
	var networks []instances.NetworkOpts

	if v, ok := d.GetOk("network"); ok {
		if networkList, ok := v.([]interface{}); ok {
			for _, v := range networkList {
				network := v.(map[string]interface{})
				networks = append(networks, instances.NetworkOpts{
					UUID:      network["uuid"].(string),
					Port:      network["port"].(string),
					V4FixedIP: network["fixed_ip_v4"].(string),
					V6FixedIP: network["fixed_ip_v6"].(string),
				})
			}
		}
	}

	createOpts.Networks = networks

	// databases
	var dbs databases.BatchCreateOpts

	if v, ok := d.GetOk("database"); ok {
		if databaseList, ok := v.([]interface{}); ok {
			for _, v := range databaseList {
				db := v.(map[string]interface{})
				dbs = append(dbs, databases.CreateOpts{
					Name:    db["name"].(string),
					CharSet: db["charset"].(string),
					Collate: db["collate"].(string),
				})
			}
		}
	}

	createOpts.Databases = dbs

	// users
	var UserList users.BatchCreateOpts

	if v, ok := d.GetOk("user"); ok {
		if userList, ok := v.([]interface{}); ok {
			for _, v := range userList {
				user := v.(map[string]interface{})
				UserList = append(UserList, users.CreateOpts{
					Name:      user["name"].(string),
					Password:  user["password"].(string),
					Databases: resourceDBv1GetDatabases(user["databases"]),
					Host:      user["host"].(string),
				})
			}
		}
	}

	createOpts.Users = UserList

	log.Printf("[DEBUG] Create Options: %#v", createOpts)
	instance, err := instances.Create(databaseV1Client, createOpts).Extract()
	if err != nil {
		return fmt.Errorf("Error creating database instance: %s", err)
	}
	log.Printf("[INFO] database instance ID: %s", instance.ID)

	// Wait for the instance to become available.
	log.Printf(
		"[DEBUG] Waiting for database instance (%s) to become available",
		instance.ID)

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"BUILD"},
		Target:     []string{"ACTIVE"},
		Refresh:    DatabaseInstanceV1StateRefreshFunc(databaseV1Client, instance.ID),
		Timeout:    d.Timeout(schema.TimeoutCreate),
		Delay:      10 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	_, err = stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf(
			"Error waiting for database instance (%s) to become ready: %s",
			instance.ID, err)
	}

	if configuration, ok := d.GetOk("configuration_id"); ok {
		err := instances.AttachConfigurationGroup(databaseV1Client, instance.ID, configuration.(string)).ExtractErr()
		if err != nil {
			return err
		}
		log.Printf("Attaching configuration %v to the instance %v", configuration, instance.ID)
	}

	// Store the ID now
	d.SetId(instance.ID)

	return resourceDatabaseInstanceV1Read(d, meta)
}

func resourceDatabaseInstanceV1Read(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	databaseV1Client, err := config.databaseV1Client(GetRegion(d, config))
	if err != nil {
		return fmt.Errorf("Error creating database client: %s", err)
	}

	instance, err := instances.Get(databaseV1Client, d.Id()).Extract()
	if err != nil {
		return CheckDeleted(d, err, "instance")
	}

	log.Printf("[DEBUG] Retrieved database instance %s: %+v", d.Id(), instance)

	d.Set("name", instance.Name)
	d.Set("flavor_id", instance.Flavor)
	d.Set("datastore", instance.Datastore)
	d.Set("region", GetRegion(d, config))

	return nil
}

func resourceDatabaseInstanceUpdate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	databaseV1Client, err := config.databaseV1Client(GetRegion(d, config))
	if err != nil {
		return fmt.Errorf("Error creating database client: %s", err)
	}

	if d.HasChange("configuration_id") {
		old, new := d.GetChange("configuration_id")

		err := instances.DetachConfigurationGroup(databaseV1Client, d.Id()).ExtractErr()
		if err != nil {
			return err
		}
		log.Printf("Detaching configuration %v from the instance %v", old, d.Id())

		if new != "" {
			err := instances.AttachConfigurationGroup(databaseV1Client, d.Id(), new.(string)).ExtractErr()
			if err != nil {
				return err
			}
			log.Printf("Attaching configuration %v to the instance %v", new, d.Id())
		}
	}

	return resourceDatabaseInstanceV1Read(d, meta)
}

func resourceDatabaseInstanceV1Delete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	databaseV1Client, err := config.databaseV1Client(GetRegion(d, config))
	if err != nil {
		return fmt.Errorf("Error creating database client: %s", err)
	}

	log.Printf("[DEBUG] Deleting database instance %s", d.Id())
	err = instances.Delete(databaseV1Client, d.Id()).ExtractErr()
	if err != nil {
		return fmt.Errorf("Error deleting database instance: %s", err)
	}

	// Wait for the database to delete before moving on.
	log.Printf("[DEBUG] Waiting for database instance (%s) to delete", d.Id())

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"ACTIVE", "SHUTDOWN"},
		Target:     []string{"DELETED"},
		Refresh:    DatabaseInstanceV1StateRefreshFunc(databaseV1Client, d.Id()),
		Timeout:    d.Timeout(schema.TimeoutDelete),
		Delay:      10 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	_, err = stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf(
			"Error waiting for database instance (%s) to delete: %s",
			d.Id(), err)
	}

	return nil
}

// DatabaseInstanceV1StateRefreshFunc returns a resource.StateRefreshFunc
// that is used to watch a database instance.
func DatabaseInstanceV1StateRefreshFunc(client *gophercloud.ServiceClient, instanceID string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		i, err := instances.Get(client, instanceID).Extract()
		if err != nil {
			if _, ok := err.(gophercloud.ErrDefault404); ok {
				return i, "DELETED", nil
			}
			return nil, "", err
		}

		if i.Status == "error" {
			return i, i.Status, fmt.Errorf("There was an error creating the database instance.")
		}

		return i, i.Status, nil
	}
}

func resourceDBv1GetDatabases(v interface{}) databases.BatchCreateOpts {
	var dbs databases.BatchCreateOpts

	if v, ok := v.(*schema.Set); ok {
		for _, db := range v.List() {
			dbs = append(dbs, databases.CreateOpts{
				Name: db.(string),
			})
		}
	}

	return dbs
}
