package google

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/schema"

	"google.golang.org/api/googleapi"
	"google.golang.org/api/sqladmin/v1beta4"
)

func resourceSqlDatabase() *schema.Resource {
	return &schema.Resource{
		Create: resourceSqlDatabaseCreate,
		Read:   resourceSqlDatabaseRead,
		Delete: resourceSqlDatabaseDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"instance": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"project": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"self_link": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceSqlDatabaseCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	project, err := getProject(d, config)
	if err != nil {
		return err
	}

	database_name := d.Get("name").(string)
	instance_name := d.Get("instance").(string)

	db := &sqladmin.Database{
		Name:     database_name,
		Instance: instance_name,
	}

	op, err := config.clientSqlAdmin.Databases.Insert(project, instance_name,
		db).Do()

	if err != nil {
		return fmt.Errorf("Error, failed to insert "+
			"database %s into instance %s: %s", database_name,
			instance_name, err)
	}

	err = sqladminOperationWait(config, op, "Insert Database")

	if err != nil {
		return fmt.Errorf("Error, failure waiting for insertion of %s "+
			"into %s: %s", database_name, instance_name, err)
	}

	return resourceSqlDatabaseRead(d, meta)
}

func resourceSqlDatabaseRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	project, err := getProject(d, config)
	if err != nil {
		return err
	}

	database_name := d.Get("name").(string)
	instance_name := d.Get("instance").(string)

	db, err := config.clientSqlAdmin.Databases.Get(project, instance_name,
		database_name).Do()

	if err != nil {
		if gerr, ok := err.(*googleapi.Error); ok && gerr.Code == 404 {
			log.Printf("[WARN] Removing SQL Database %q because it's gone", d.Get("name").(string))
			// The resource doesn't exist anymore
			d.SetId("")

			return nil
		}

		return fmt.Errorf("Error, failed to get"+
			"database %s in instance %s: %s", database_name,
			instance_name, err)
	}

	d.Set("self_link", db.SelfLink)
	d.SetId(instance_name + ":" + database_name)

	return nil
}

func resourceSqlDatabaseDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	project, err := getProject(d, config)
	if err != nil {
		return err
	}

	database_name := d.Get("name").(string)
	instance_name := d.Get("instance").(string)

	op, err := config.clientSqlAdmin.Databases.Delete(project, instance_name,
		database_name).Do()

	if err != nil {
		return fmt.Errorf("Error, failed to delete"+
			"database %s in instance %s: %s", database_name,
			instance_name, err)
	}

	err = sqladminOperationWait(config, op, "Delete Database")

	if err != nil {
		return fmt.Errorf("Error, failure waiting for deletion of %s "+
			"in %s: %s", database_name, instance_name, err)
	}

	return nil
}
