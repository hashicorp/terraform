package mssql

import (
	"database/sql"
	"fmt"

	"github.com/hashicorp/terraform/helper/schema"
)

func resourceMSsqlDatabase() *schema.Resource {
	return &schema.Resource{
		Create: resourceMSsqlDatabaseCreate,
		Read:   resourceMSsqlDatabaseRead,
		Delete: resourceMSsqlDatabaseDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
		},
	}
}

func resourceMSsqlDatabaseCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Client)
	conn, err := client.Connect()
	if err != nil {
		return err
	}
	defer conn.Close()

	dbName := d.Get("name").(string)

	_, err = conn.Query("CREATE DATABASE [" + dbName + "]")
	if err != nil {
		return fmt.Errorf("Error creating MSSQL database %s: %s", dbName, err)
	}

	d.SetId(dbName)

	return nil
}

func resourceMSsqlDatabaseDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Client)
	conn, err := client.Connect()
	if err != nil {
		return err
	}
	defer conn.Close()

	dbName := d.Get("name").(string)

	_, err = conn.Query("DROP DATABASE [" + dbName + "]")
	if err != nil {
		return err
	}

	d.SetId("")

	return nil
}

func resourceMSsqlDatabaseRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Client)
	conn, err := client.Connect()
	if err != nil {
		return err
	}
	defer conn.Close()

	dbName := d.Get("name").(string)

	result, err := conn.Query("SELECT db_id(?1)", dbName)
	defer result.Close()

	if err != nil {
		d.SetId("")
		return nil
	}

	for result.Next() {
		var s sql.NullString
		err := result.Scan(&s)
		if err != nil {
			d.SetId("")
			return nil
		}

		// Check result
		if s.Valid {
			d.SetId(dbName)
		} else {
			d.SetId("")
		}
		return nil
	}

	d.SetId("")
	return nil
}
