package azure

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/Azure/azure-sdk-for-go/management/sql"
	"github.com/hashicorp/terraform/helper/schema"
)

// resourceAzureSqlDatabaseService returns the *schema.Resource
// associated to an SQL Database Service on Azure.
func resourceAzureSqlDatabaseService() *schema.Resource {
	return &schema.Resource{
		Create: resourceAzureSqlDatabaseServiceCreate,
		Read:   resourceAzureSqlDatabaseServiceRead,
		Update: resourceAzureSqlDatabaseServiceUpdate,
		Exists: resourceAzureSqlDatabaseServiceExists,
		Delete: resourceAzureSqlDatabaseServiceDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"database_server_name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"collation": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},
			"edition": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"max_size_bytes": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"service_level_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
		},
	}
}

// resourceAzureSqlDatabaseServiceCreate does all the necessary API calls to
// create an SQL Database Service on Azure.
func resourceAzureSqlDatabaseServiceCreate(d *schema.ResourceData, meta interface{}) error {
	sqlClient := meta.(*Client).sqlClient

	log.Println("[INFO] Creating Azure SQL Database service creation request.")
	name := d.Get("name").(string)
	serverName := d.Get("database_server_name").(string)
	params := sql.DatabaseCreateParams{
		Name:               name,
		Edition:            d.Get("edition").(string),
		CollationName:      d.Get("collation").(string),
		ServiceObjectiveID: d.Get("service_level_id").(string),
	}

	if maxSize, ok := d.GetOk("max_size_bytes"); ok {
		val, err := strconv.ParseInt(maxSize.(string), 10, 64)
		if err != nil {
			return fmt.Errorf("Provided max_size_bytes is not an integer: %s", err)
		}
		params.MaxSizeBytes = val
	}

	log.Println("[INFO] Sending SQL Database Service creation request to Azure.")
	err := sqlClient.CreateDatabase(serverName, params)
	if err != nil {
		return fmt.Errorf("Error issuing Azure SQL Database Service creation: %s", err)
	}

	log.Println("[INFO] Beginning wait for Azure SQL Database Service creation.")
	err = sqlClient.WaitForDatabaseCreation(serverName, name, nil)
	if err != nil {
		return fmt.Errorf("Error whilst waiting for Azure SQL Database Service creation: %s", err)
	}

	d.SetId(name)

	return resourceAzureSqlDatabaseServiceRead(d, meta)
}

// resourceAzureSqlDatabaseServiceRead does all the necessary API calls to
// read the state of the SQL Database Service off Azure.
func resourceAzureSqlDatabaseServiceRead(d *schema.ResourceData, meta interface{}) error {
	sqlClient := meta.(*Client).sqlClient

	log.Println("[INFO] Issuing Azure SQL Database Services list operation.")
	serverName := d.Get("database_server_name").(string)
	dbs, err := sqlClient.ListDatabases(serverName)
	if err != nil {
		return fmt.Errorf("Error whilst listing Database Services off Azure: %s", err)
	}

	// search for our database:
	var found bool
	name := d.Get("name").(string)
	for _, db := range dbs.ServiceResources {
		if db.Name == name {
			found = true
			d.Set("edition", db.Edition)
			d.Set("collation", db.CollationName)
			d.Set("max_size_bytes", strconv.FormatInt(db.MaxSizeBytes, 10))
			d.Set("service_level_id", db.ServiceObjectiveID)
			break
		}
	}

	// if not found; we must untrack the resource:
	if !found {
		d.SetId("")
	}

	return nil
}

// resourceAzureSqlDatabaseServiceUpdate does all the necessary API calls to
// update the state of the SQL Database Service off Azure.
func resourceAzureSqlDatabaseServiceUpdate(d *schema.ResourceData, meta interface{}) error {
	azureClient := meta.(*Client)
	mgmtClient := azureClient.mgmtClient
	sqlClient := azureClient.sqlClient
	serverName := d.Get("database_server_name").(string)

	// changes to the name must occur separately from changes to the attributes:
	if d.HasChange("name") {
		oldv, newv := d.GetChange("name")

		// issue the update request:
		log.Println("[INFO] Issuing Azure Database Service name change.")
		reqID, err := sqlClient.UpdateDatabase(serverName, oldv.(string),
			sql.ServiceResourceUpdateParams{
				Name: newv.(string),
			})

		// wait for the update to occur:
		log.Println("[INFO] Waiting for Azure SQL Database Service name change.")
		err = mgmtClient.WaitForOperation(reqID, nil)
		if err != nil {
			return fmt.Errorf("Error waiting for Azure SQL Database Service name update: %s", err)
		}

		// set the new name as the ID:
		d.SetId(newv.(string))
	}

	name := d.Get("name").(string)
	cedition := d.HasChange("edition")
	cmaxsize := d.HasChange("max_size_bytes")
	clevel := d.HasChange("service_level_id")
	if cedition || cmaxsize || clevel {
		updateParams := sql.ServiceResourceUpdateParams{
			// we still have to stick the name in here for good measure:
			Name: name,
		}

		// build the update request:
		if cedition {
			updateParams.Edition = d.Get("edition").(string)
		}
		if maxSize, ok := d.GetOk("max_size_bytes"); cmaxsize && ok && maxSize.(string) != "" {
			val, err := strconv.ParseInt(maxSize.(string), 10, 64)
			if err != nil {
				return fmt.Errorf("Provided max_size_bytes is not an integer: %s", err)
			}
			updateParams.MaxSizeBytes = val
		}
		if clevel {
			updateParams.ServiceObjectiveID = d.Get("service_level_id").(string)
		}

		// issue the update:
		log.Println("[INFO] Issuing Azure Database Service parameter update.")
		reqID, err := sqlClient.UpdateDatabase(serverName, name, updateParams)
		if err != nil {
			return fmt.Errorf("Failed issuing Azure SQL Service parameter update: %s", err)
		}

		log.Println("[INFO] Waiting for Azure SQL Database Service parameter update.")
		err = mgmtClient.WaitForOperation(reqID, nil)
		if err != nil {
			return fmt.Errorf("Error waiting for Azure SQL Database Service parameter update: %s", err)
		}
	}

	return nil
}

// resourceAzureSqlDatabaseServiceExists does all the necessary API calls to
// check for the existence of the SQL Database Service off Azure.
func resourceAzureSqlDatabaseServiceExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	sqlClient := meta.(*Client).sqlClient

	log.Println("[INFO] Issuing Azure SQL Database Service get request.")
	name := d.Get("name").(string)
	serverName := d.Get("database_server_name").(string)
	_, err := sqlClient.GetDatabase(serverName, name)
	if err != nil {
		if strings.Contains(err.Error(), "does not exist") {
			d.SetId("")
			return false, nil
		} else {
			return false, fmt.Errorf("Error whilst getting Azure SQL Database Service info: %s", err)
		}
	}

	return true, nil
}

// resourceAzureSqlDatabaseServiceDelete does all the necessary API calls to
// delete the SQL Database Service off Azure.
func resourceAzureSqlDatabaseServiceDelete(d *schema.ResourceData, meta interface{}) error {
	sqlClient := meta.(*Client).sqlClient

	log.Println("[INFO] Issuing Azure SQL Database deletion request.")
	name := d.Get("name").(string)
	serverName := d.Get("database_server_name").(string)
	return sqlClient.DeleteDatabase(serverName, name)
}
