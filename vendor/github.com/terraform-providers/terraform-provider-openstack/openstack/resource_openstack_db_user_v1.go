package openstack

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/db/v1/databases"
	"github.com/gophercloud/gophercloud/openstack/db/v1/users"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceDatabaseUserV1() *schema.Resource {
	return &schema.Resource{
		Create: resourceDatabaseUserV1Create,
		Read:   resourceDatabaseUserV1Read,
		Delete: resourceDatabaseUserV1Delete,

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
			"instance_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"password": {
				Type:      schema.TypeString,
				Required:  true,
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
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},
		},
	}
}

func resourceDatabaseUserV1Create(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	databaseV1Client, err := config.databaseV1Client(GetRegion(d, config))
	if err != nil {
		return fmt.Errorf("Error creating cloud database client: %s", err)
	}

	userName := d.Get("name").(string)
	rawDatabases := d.Get("databases").(*schema.Set).List()
	instanceID := d.Get("instance_id").(string)

	var dbs databases.BatchCreateOpts
	for _, db := range rawDatabases {
		dbs = append(dbs, databases.CreateOpts{
			Name: db.(string),
		})
	}

	var usersList users.BatchCreateOpts
	usersList = append(usersList, users.CreateOpts{
		Name:      userName,
		Password:  d.Get("password").(string),
		Host:      d.Get("host").(string),
		Databases: dbs,
	})

	users.Create(databaseV1Client, instanceID, usersList)

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"BUILD"},
		Target:     []string{"ACTIVE"},
		Refresh:    DatabaseUserV1StateRefreshFunc(databaseV1Client, instanceID, userName),
		Timeout:    d.Timeout(schema.TimeoutCreate),
		Delay:      10 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	_, err = stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf(
			"Error waiting for user (%s) to be created", err)
	}

	// Store the ID now
	d.SetId(fmt.Sprintf("%s/%s", instanceID, userName))

	return resourceDatabaseUserV1Read(d, meta)
}

func resourceDatabaseUserV1Read(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	databaseV1Client, err := config.databaseV1Client(GetRegion(d, config))
	if err != nil {
		return fmt.Errorf("Error creating cloud database client: %s", err)
	}

	userID := strings.SplitN(d.Id(), "/", 2)
	if len(userID) != 2 {
		return fmt.Errorf("Invalid openstack_db_user_v1 ID format")
	}

	instanceID := userID[0]
	userName := userID[1]

	exists, userObj, err := DatabaseUserV1State(databaseV1Client, instanceID, userName)
	if err != nil {
		return fmt.Errorf("Error checking user status: %s", err)
	}

	if !exists {
		return fmt.Errorf("User %s was not found: %s", userName, err)
	}

	log.Printf("[DEBUG] Retrieved user %s", userName)

	d.Set("name", userName)

	var databases []string
	for _, dbName := range userObj.Databases {
		databases = append(databases, dbName.Name)
	}

	if err := d.Set("databases", databases); err != nil {
		return fmt.Errorf("Unable to set databases: %s", err)
	}

	return nil
}

func resourceDatabaseUserV1Delete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	databaseV1Client, err := config.databaseV1Client(GetRegion(d, config))
	if err != nil {
		return fmt.Errorf("Error creating cloud database client: %s", err)
	}

	userID := strings.SplitN(d.Id(), "/", 2)
	if len(userID) != 2 {
		return fmt.Errorf("Invalid openstack_db_user_v1 ID format")
	}

	instanceID := userID[0]
	userName := userID[1]

	exists, _, err := DatabaseUserV1State(databaseV1Client, instanceID, userName)
	if err != nil {
		return fmt.Errorf("Error checking user status: %s", err)
	}

	if !exists {
		log.Printf("User %s was not found on instance %s", userName, instanceID)
		return nil
	}

	log.Printf("[DEBUG] Retrieved user %s", userName)

	users.Delete(databaseV1Client, instanceID, userName)

	d.SetId("")
	return nil
}

// DatabaseUserV1StateRefreshFunc returns a resource.StateRefreshFunc that is used to watch db user.
func DatabaseUserV1StateRefreshFunc(client *gophercloud.ServiceClient, instanceID string, userName string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {

		pages, err := users.List(client, instanceID).AllPages()
		if err != nil {
			return nil, "", fmt.Errorf("Unable to retrieve users, pages: %s", err)
		}

		allUsers, err := users.ExtractUsers(pages)
		if err != nil {
			return nil, "", fmt.Errorf("Unable to retrieve users, extract: %s", err)
		}

		for _, v := range allUsers {
			if v.Name == userName {
				return v, "ACTIVE", nil
			}
		}

		return nil, "BUILD", nil
	}
}

// DatabaseUserV1State is used to check whether user exists on particular database instance
func DatabaseUserV1State(client *gophercloud.ServiceClient, instanceID string, userName string) (exists bool, userObj users.User, err error) {
	exists = false
	err = nil

	pages, err := users.List(client, instanceID).AllPages()
	if err != nil {
		return
	}

	allUsers, err := users.ExtractUsers(pages)
	if err != nil {
		return
	}

	for _, v := range allUsers {
		if v.Name == userName {
			exists = true
			userObj = v
			return
		}
	}

	return false, userObj, err
}
