package azure

import (
	"fmt"
	"log"

	"github.com/Azure/azure-sdk-for-go/management/sql"
	"github.com/hashicorp/terraform/helper/schema"
)

// resourceAsmSqlDatabaseServer returns the *schema.Resource associated
// to a database server on Azure.
func resourceAsmSqlDatabaseServer() *schema.Resource {
	return &schema.Resource{
		Create: resourceAsmSqlDatabaseServerCreate,
		Read:   resourceAsmSqlDatabaseServerRead,
		Delete: resourceAsmSqlDatabaseServerDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
				ForceNew: true,
			},
			"location": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"username": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"password": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"version": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "12.0",
				ForceNew: true,
			},
			"url": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

// resourceAsmSqlDatabaseServerCreate does all the necessary API calls to
// create an SQL database server off Azure.
func resourceAsmSqlDatabaseServerCreate(d *schema.ResourceData, meta interface{}) error {
	sqlClient := meta.(*AzureClient).asmClient.sqlClient

	log.Println("[INFO] Began constructing SQL Server creation request.")
	params := sql.DatabaseServerCreateParams{
		Location:                   d.Get("location").(string),
		AdministratorLogin:         d.Get("username").(string),
		AdministratorLoginPassword: d.Get("password").(string),
		Version:                    d.Get("version").(string),
	}

	log.Println("[INFO] Issuing SQL Server creation request to Azure.")
	name, err := sqlClient.CreateServer(params)
	if err != nil {
		return fmt.Errorf("Error creating SQL Server on Azure: %s", err)
	}

	d.Set("name", name)

	d.SetId(name)
	return resourceAsmSqlDatabaseServerRead(d, meta)
}

// resourceAsmSqlDatabaseServerRead does all the necessary API calls to
// read the state of the SQL database server off Azure.
func resourceAsmSqlDatabaseServerRead(d *schema.ResourceData, meta interface{}) error {
	sqlClient := meta.(*AzureClient).asmClient.sqlClient

	log.Println("[INFO] Sending SQL Servers list query to Azure.")
	srvList, err := sqlClient.ListServers()
	if err != nil {
		return fmt.Errorf("Error issuing SQL Servers list query to Azure: %s", err)
	}

	// search for our particular server:
	name := d.Get("name")
	for _, srv := range srvList.DatabaseServers {
		if srv.Name == name {
			d.Set("url", srv.FullyQualifiedDomainName)
			d.Set("state", srv.State)
			return nil
		}
	}

	// if reached here; it means out server doesn't exist, so we must untrack it:
	d.SetId("")
	return nil
}

// resourceAsmSqlDatabaseServerDelete does all the necessary API calls to
// delete the SQL database server off Azure.
func resourceAsmSqlDatabaseServerDelete(d *schema.ResourceData, meta interface{}) error {
	sqlClient := meta.(*AzureClient).asmClient.sqlClient

	log.Println("[INFO] Sending SQL Server deletion request to Azure.")
	name := d.Get("name").(string)
	err := sqlClient.DeleteServer(name)
	if err != nil {
		return fmt.Errorf("Error while issuing SQL Server deletion request to Azure: %s", err)
	}

	return nil
}
