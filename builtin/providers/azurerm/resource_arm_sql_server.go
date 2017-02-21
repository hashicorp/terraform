package azurerm

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/jen20/riviera/azure"
	"github.com/jen20/riviera/sql"
)

func resourceArmSqlServer() *schema.Resource {
	return &schema.Resource{
		Create: resourceArmSqlServerCreate,
		Read:   resourceArmSqlServerRead,
		Update: resourceArmSqlServerCreate,
		Delete: resourceArmSqlServerDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"location": locationSchema(),

			"resource_group_name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"version": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"administrator_login": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"administrator_login_password": &schema.Schema{
				Type:      schema.TypeString,
				Required:  true,
				Sensitive: true,
			},

			"fully_qualified_domain_name": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"tags": tagsSchema(),
		},
	}
}

func resourceArmSqlServerCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ArmClient)
	rivieraClient := client.rivieraClient

	tags := d.Get("tags").(map[string]interface{})
	expandedTags := expandTags(tags)

	createRequest := rivieraClient.NewRequest()
	createRequest.Command = &sql.CreateOrUpdateServer{
		Name:                       d.Get("name").(string),
		Location:                   d.Get("location").(string),
		ResourceGroupName:          d.Get("resource_group_name").(string),
		AdministratorLogin:         azure.String(d.Get("administrator_login").(string)),
		AdministratorLoginPassword: azure.String(d.Get("administrator_login_password").(string)),
		Version:                    azure.String(d.Get("version").(string)),
		Tags:                       *expandedTags,
	}

	createResponse, err := createRequest.Execute()
	if err != nil {
		return fmt.Errorf("Error creating SQL Server: %s", err)
	}
	if !createResponse.IsSuccessful() {
		return fmt.Errorf("Error creating SQL Server: %s", createResponse.Error)
	}

	readRequest := rivieraClient.NewRequest()
	readRequest.Command = &sql.GetServer{
		Name:              d.Get("name").(string),
		ResourceGroupName: d.Get("resource_group_name").(string),
	}

	readResponse, err := readRequest.Execute()
	if err != nil {
		return fmt.Errorf("Error reading SQL Server: %s", err)
	}
	if !readResponse.IsSuccessful() {
		return fmt.Errorf("Error reading SQL Server: %s", readResponse.Error)
	}

	resp := readResponse.Parsed.(*sql.GetServerResponse)
	d.SetId(*resp.ID)

	return resourceArmSqlServerRead(d, meta)
}

func resourceArmSqlServerRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ArmClient)
	rivieraClient := client.rivieraClient

	id, err := parseAzureResourceID(d.Id())
	if err != nil {
		return err
	}

	readRequest := rivieraClient.NewRequestForURI(d.Id())
	readRequest.Command = &sql.GetServer{}

	readResponse, err := readRequest.Execute()
	if err != nil {
		return fmt.Errorf("Error reading SQL Server: %s", err)
	}
	if !readResponse.IsSuccessful() {
		log.Printf("[INFO] Error reading SQL Server %q - removing from state", d.Id())
		d.SetId("")
		return fmt.Errorf("Error reading SQL Server: %s", readResponse.Error)
	}

	resp := readResponse.Parsed.(*sql.GetServerResponse)

	d.Set("name", id.Path["servers"])
	d.Set("resource_group_name", id.ResourceGroup)
	d.Set("location", azureRMNormalizeLocation(*resp.Location))
	d.Set("fully_qualified_domain_name", resp.FullyQualifiedDomainName)
	d.Set("administrator_login", resp.AdministratorLogin)
	d.Set("version", resp.Version)

	flattenAndSetTags(d, resp.Tags)

	return nil
}

func resourceArmSqlServerDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ArmClient)
	rivieraClient := client.rivieraClient

	deleteRequest := rivieraClient.NewRequestForURI(d.Id())
	deleteRequest.Command = &sql.DeleteServer{}

	deleteResponse, err := deleteRequest.Execute()
	if err != nil {
		return fmt.Errorf("Error deleting SQL Server: %s", err)
	}
	if !deleteResponse.IsSuccessful() {
		return fmt.Errorf("Error deleting SQL Server: %s", deleteResponse.Error)
	}

	return nil
}
