package openstack

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/db/v1/databases"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceDatabaseDatabaseV1() *schema.Resource {
	return &schema.Resource{
		Create: resourceDatabaseDatabaseV1Create,
		Read:   resourceDatabaseDatabaseV1Read,
		Delete: resourceDatabaseDatabaseV1Delete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(10 * time.Minute),
			Delete: schema.DefaultTimeout(10 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			"region": {
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				DefaultFunc: schema.EnvDefaultFunc("OS_REGION_NAME", ""),
			},
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"instance_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
		},
	}
}

func resourceDatabaseDatabaseV1Create(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	databaseV1Client, err := config.databaseV1Client(GetRegion(d, config))
	if err != nil {
		return fmt.Errorf("Error creating cloud database client: %s", err)
	}

	dbName := d.Get("name").(string)
	instanceID := d.Get("instance_id").(string)

	var dbs databases.BatchCreateOpts
	dbs = append(dbs, databases.CreateOpts{
		Name: dbName,
	})

	exists, err := DatabaseDatabaseV1State(databaseV1Client, instanceID, dbName)
	if err != nil {
		return fmt.Errorf("Error checking database status: %s", err)
	}

	if exists {
		return fmt.Errorf("Database %s exists on instance %s", dbName, instanceID)
	}

	err = databases.Create(databaseV1Client, instanceID, dbs).ExtractErr()
	if err != nil {
		return err
	}

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"BUILD"},
		Target:     []string{"ACTIVE"},
		Refresh:    DatabaseDatabaseV1StateRefreshFunc(databaseV1Client, instanceID, dbName),
		Timeout:    d.Timeout(schema.TimeoutCreate),
		Delay:      10 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	_, err = stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf(
			"Error waiting for database to become ready: %s", err)
	}

	// Store the ID now
	d.SetId(fmt.Sprintf("%s/%s", instanceID, dbName))

	return resourceDatabaseDatabaseV1Read(d, meta)
}

func resourceDatabaseDatabaseV1Read(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	databaseV1Client, err := config.databaseV1Client(GetRegion(d, config))
	if err != nil {
		return fmt.Errorf("Error creating database client: %s", err)
	}

	dbID := strings.SplitN(d.Id(), "/", 2)
	if len(dbID) != 2 {
		return fmt.Errorf("Invalid openstack_db_database_v1 ID format")
	}

	instanceID := dbID[0]
	dbName := dbID[1]

	exists, err := DatabaseDatabaseV1State(databaseV1Client, instanceID, dbName)
	if err != nil {
		return fmt.Errorf("Error checking database status: %s", err)
	}

	if !exists {
		return fmt.Errorf("database %s was not found", err)
	}

	log.Printf("[DEBUG] Retrieved database %s", dbName)

	d.Set("name", dbName)
	d.Set("instance_id", instanceID)

	return nil
}

func resourceDatabaseDatabaseV1Delete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	databaseV1Client, err := config.databaseV1Client(GetRegion(d, config))
	if err != nil {
		return fmt.Errorf("Error creating cloud database client: %s", err)
	}

	dbID := strings.SplitN(d.Id(), "/", 2)
	if len(dbID) != 2 {
		return fmt.Errorf("Invalid openstack_db_database_v1 ID: %s", d.Id())
	}

	instanceID := dbID[0]
	dbName := dbID[1]

	exists, err := DatabaseDatabaseV1State(databaseV1Client, instanceID, dbName)
	if err != nil {
		return fmt.Errorf("Error checking database status: %s", err)
	}

	if !exists {
		return nil
	}

	err = databases.Delete(databaseV1Client, instanceID, dbName).ExtractErr()
	if err != nil {
		return fmt.Errorf("Error deleting database %s: %s", dbName, err)
	}

	return nil
}

// DatabaseDatabaseV1StateRefreshFunc returns a resource.StateRefreshFunc
// that is used to watch a database.
func DatabaseDatabaseV1StateRefreshFunc(client *gophercloud.ServiceClient, instanceID string, dbName string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		pages, err := databases.List(client, instanceID).AllPages()
		if err != nil {
			return nil, "", fmt.Errorf("Unable to retrieve databases: %s", err)
		}

		allDatabases, err := databases.ExtractDBs(pages)
		if err != nil {
			return nil, "", fmt.Errorf("Unable to extract databases: %s", err)
		}

		for _, v := range allDatabases {
			if v.Name == dbName {
				return v, "ACTIVE", nil
			}
		}

		return nil, "BUILD", nil
	}
}

func DatabaseDatabaseV1State(client *gophercloud.ServiceClient, instanceID string, dbName string) (exists bool, err error) {
	exists = false
	err = nil

	pages, err := databases.List(client, instanceID).AllPages()
	if err != nil {
		return
	}

	allDatabases, err := databases.ExtractDBs(pages)
	if err != nil {
		return
	}

	for _, v := range allDatabases {
		if v.Name == dbName {
			exists = true
			return
		}
	}

	return false, err
}
