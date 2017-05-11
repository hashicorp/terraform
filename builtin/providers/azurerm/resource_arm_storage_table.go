package azurerm

import (
	"fmt"
	"log"
	"regexp"

	"github.com/Azure/azure-sdk-for-go/storage"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceArmStorageTable() *schema.Resource {
	return &schema.Resource{
		Create: resourceArmStorageTableCreate,
		Read:   resourceArmStorageTableRead,
		Delete: resourceArmStorageTableDelete,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validateArmStorageTableName,
			},
			"resource_group_name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"storage_account_name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
		},
	}
}

func validateArmStorageTableName(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)
	if value == "table" {
		errors = append(errors, fmt.Errorf(
			"Table Storage %q cannot use the word `table`: %q",
			k, value))
	}
	if !regexp.MustCompile(`^[A-Za-z][A-Za-z0-9]{6,63}$`).MatchString(value) {
		errors = append(errors, fmt.Errorf(
			"Table Storage %q cannot begin with a numeric character, only alphanumeric characters are allowed and must be between 6 and 63 characters long: %q",
			k, value))
	}

	return
}

func resourceArmStorageTableCreate(d *schema.ResourceData, meta interface{}) error {
	armClient := meta.(*ArmClient)

	resourceGroupName := d.Get("resource_group_name").(string)
	storageAccountName := d.Get("storage_account_name").(string)

	tableClient, accountExists, err := armClient.getTableServiceClientForStorageAccount(resourceGroupName, storageAccountName)
	if err != nil {
		return err
	}
	if !accountExists {
		return fmt.Errorf("Storage Account %q Not Found", storageAccountName)
	}

	name := d.Get("name").(string)
	table := tableClient.GetTableReference(name)

	log.Printf("[INFO] Creating table %q in storage account %q.", name, storageAccountName)

	timeout := uint(60)
	options := &storage.TableOptions{}
	err = table.Create(timeout, storage.NoMetadata, options)
	if err != nil {
		return fmt.Errorf("Error creating table %q in storage account %q: %s", name, storageAccountName, err)
	}

	d.SetId(name)

	return resourceArmStorageTableRead(d, meta)
}

func resourceArmStorageTableRead(d *schema.ResourceData, meta interface{}) error {
	armClient := meta.(*ArmClient)

	resourceGroupName := d.Get("resource_group_name").(string)
	storageAccountName := d.Get("storage_account_name").(string)

	tableClient, accountExists, err := armClient.getTableServiceClientForStorageAccount(resourceGroupName, storageAccountName)
	if err != nil {
		return err
	}
	if !accountExists {
		log.Printf("[DEBUG] Storage account %q not found, removing table %q from state", storageAccountName, d.Id())
		d.SetId("")
		return nil
	}

	name := d.Get("name").(string)
	metaDataLevel := storage.MinimalMetadata
	options := &storage.QueryTablesOptions{}
	tables, err := tableClient.QueryTables(metaDataLevel, options)
	if err != nil {
		return fmt.Errorf("Failed to retrieve storage tables in account %q: %s", name, err)
	}

	var found bool
	for _, table := range tables.Tables {
		tableName := string(table.Name)
		if tableName == name {
			found = true
			d.Set("name", tableName)
		}
	}

	if !found {
		log.Printf("[INFO] Storage table %q does not exist in account %q, removing from state...", name, storageAccountName)
		d.SetId("")
	}

	return nil
}

func resourceArmStorageTableDelete(d *schema.ResourceData, meta interface{}) error {
	armClient := meta.(*ArmClient)

	resourceGroupName := d.Get("resource_group_name").(string)
	storageAccountName := d.Get("storage_account_name").(string)

	tableClient, accountExists, err := armClient.getTableServiceClientForStorageAccount(resourceGroupName, storageAccountName)
	if err != nil {
		return err
	}
	if !accountExists {
		log.Printf("[INFO] Storage Account %q doesn't exist so the table won't exist", storageAccountName)
		return nil
	}

	name := d.Get("name").(string)
	table := tableClient.GetTableReference(name)
	timeout := uint(60)
	options := &storage.TableOptions{}

	log.Printf("[INFO] Deleting storage table %q in account %q", name, storageAccountName)
	if err := table.Delete(timeout, options); err != nil {
		return fmt.Errorf("Error deleting storage table %q from storage account %q: %s", name, storageAccountName, err)
	}

	d.SetId("")
	return nil
}
