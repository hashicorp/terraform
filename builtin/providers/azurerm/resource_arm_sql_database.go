package azurerm

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/jen20/riviera/azure"
	"github.com/jen20/riviera/sql"
)

func resourceArmSqlDatabase() *schema.Resource {
	return &schema.Resource{
		Create: resourceArmSqlDatabaseCreate,
		Read:   resourceArmSqlDatabaseRead,
		Update: resourceArmSqlDatabaseCreate,
		Delete: resourceArmSqlDatabaseDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"location": &schema.Schema{
				Type:      schema.TypeString,
				Required:  true,
				ForceNew:  true,
				StateFunc: azureRMNormalizeLocation,
			},

			"resource_group_name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"server_name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"create_mode": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "Default",
			},

			"source_database_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"restore_point_in_time": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"edition": &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				ValidateFunc: validateArmSqlDatabaseEdition,
			},

			"collation": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"max_size_bytes": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"requested_service_objective_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"requested_service_objective_name": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"source_database_deletion_date": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"elastic_pool_name": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"encryption": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"creation_date": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"default_secondary_location": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"tags": tagsSchema(),
		},
	}
}

func resourceArmSqlDatabaseCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ArmClient)
	rivieraClient := client.rivieraClient

	tags := d.Get("tags").(map[string]interface{})
	expandedTags := expandTags(tags)

	command := &sql.CreateOrUpdateDatabase{
		Name:              d.Get("name").(string),
		Location:          d.Get("location").(string),
		ResourceGroupName: d.Get("resource_group_name").(string),
		ServerName:        d.Get("server_name").(string),
		Tags:              *expandedTags,
		CreateMode:        azure.String(d.Get("create_mode").(string)),
	}

	if v, ok := d.GetOk("source_database_id"); ok {
		command.SourceDatabaseID = azure.String(v.(string))
	}

	if v, ok := d.GetOk("edition"); ok {
		command.Edition = azure.String(v.(string))
	}

	if v, ok := d.GetOk("collation"); ok {
		command.Collation = azure.String(v.(string))
	}

	if v, ok := d.GetOk("max_size_bytes"); ok {
		command.MaxSizeBytes = azure.String(v.(string))
	}

	if v, ok := d.GetOk("source_database_deletion_date"); ok {
		command.SourceDatabaseDeletionDate = azure.String(v.(string))
	}

	if v, ok := d.GetOk("requested_service_objective_id"); ok {
		command.RequestedServiceObjectiveID = azure.String(v.(string))
	}

	if v, ok := d.GetOk("requested_service_objective_name"); ok {
		command.RequestedServiceObjectiveName = azure.String(v.(string))
	}

	createRequest := rivieraClient.NewRequest()
	createRequest.Command = command

	createResponse, err := createRequest.Execute()
	if err != nil {
		return fmt.Errorf("Error creating SQL Database: %s", err)
	}
	if !createResponse.IsSuccessful() {
		return fmt.Errorf("Error creating SQL Database: %s", createResponse.Error)
	}

	readRequest := rivieraClient.NewRequest()
	readRequest.Command = &sql.GetDatabase{
		Name:              d.Get("name").(string),
		ResourceGroupName: d.Get("resource_group_name").(string),
		ServerName:        d.Get("server_name").(string),
	}

	readResponse, err := readRequest.Execute()
	if err != nil {
		return fmt.Errorf("Error reading SQL Database: %s", err)
	}
	if !readResponse.IsSuccessful() {
		return fmt.Errorf("Error reading SQL Database: %s", readResponse.Error)
	}

	resp := readResponse.Parsed.(*sql.GetDatabaseResponse)
	d.SetId(*resp.ID)

	return resourceArmSqlDatabaseRead(d, meta)
}

func resourceArmSqlDatabaseRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ArmClient)
	rivieraClient := client.rivieraClient

	readRequest := rivieraClient.NewRequestForURI(d.Id())
	readRequest.Command = &sql.GetDatabase{}

	readResponse, err := readRequest.Execute()
	if err != nil {
		return fmt.Errorf("Error reading SQL Database: %s", err)
	}
	if !readResponse.IsSuccessful() {
		log.Printf("[INFO] Error reading SQL Database %q - removing from state", d.Id())
		d.SetId("")
		return fmt.Errorf("Error reading SQL Database: %s", readResponse.Error)
	}

	resp := readResponse.Parsed.(*sql.GetDatabaseResponse)

	d.Set("name", resp.Name)
	d.Set("creation_date", resp.CreationDate)
	d.Set("default_secondary_location", resp.DefaultSecondaryLocation)

	return nil
}

func resourceArmSqlDatabaseDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ArmClient)
	rivieraClient := client.rivieraClient

	deleteRequest := rivieraClient.NewRequestForURI(d.Id())
	deleteRequest.Command = &sql.DeleteDatabase{}

	deleteResponse, err := deleteRequest.Execute()
	if err != nil {
		return fmt.Errorf("Error deleting SQL Database: %s", err)
	}
	if !deleteResponse.IsSuccessful() {
		return fmt.Errorf("Error deleting SQL Database: %s", deleteResponse.Error)
	}

	return nil
}

func validateArmSqlDatabaseEdition(v interface{}, k string) (ws []string, errors []error) {
	editions := map[string]bool{
		"Basic":    true,
		"Standard": true,
		"Premium":  true,
	}

	if !editions[v.(string)] {
		errors = append(errors, fmt.Errorf("SQL Database Edition can only be Basic, Standard or Premium"))
	}
	return
}
