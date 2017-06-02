package azurerm

import (
	"fmt"
	"log"
	"regexp"

	"github.com/Azure/azure-sdk-for-go/storage"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceArmStorageShare() *schema.Resource {
	return &schema.Resource{
		Create: resourceArmStorageShareCreate,
		Read:   resourceArmStorageShareRead,
		Exists: resourceArmStorageShareExists,
		Delete: resourceArmStorageShareDelete,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validateArmStorageShareName,
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
			"quota": {
				Type:     schema.TypeInt,
				Optional: true,
				ForceNew: true,
				Default:  0,
			},
			"url": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}
func resourceArmStorageShareCreate(d *schema.ResourceData, meta interface{}) error {
	armClient := meta.(*ArmClient)

	resourceGroupName := d.Get("resource_group_name").(string)
	storageAccountName := d.Get("storage_account_name").(string)

	fileClient, accountExists, err := armClient.getFileServiceClientForStorageAccount(resourceGroupName, storageAccountName)
	if err != nil {
		return err
	}
	if !accountExists {
		return fmt.Errorf("Storage Account %q Not Found", storageAccountName)
	}

	name := d.Get("name").(string)
	metaData := make(map[string]string) // TODO: support MetaData
	options := &storage.FileRequestOptions{}

	log.Printf("[INFO] Creating share %q in storage account %q", name, storageAccountName)
	reference := fileClient.GetShareReference(name)
	err = reference.Create(options)

	log.Printf("[INFO] Setting share %q metadata in storage account %q", name, storageAccountName)
	reference.Metadata = metaData
	reference.SetMetadata(options)

	log.Printf("[INFO] Setting share %q properties in storage account %q", name, storageAccountName)
	reference.Properties = storage.ShareProperties{
		Quota: d.Get("quota").(int),
	}
	reference.SetProperties(options)

	d.SetId(name)
	return resourceArmStorageShareRead(d, meta)
}

func resourceArmStorageShareRead(d *schema.ResourceData, meta interface{}) error {
	armClient := meta.(*ArmClient)

	resourceGroupName := d.Get("resource_group_name").(string)
	storageAccountName := d.Get("storage_account_name").(string)

	fileClient, accountExists, err := armClient.getFileServiceClientForStorageAccount(resourceGroupName, storageAccountName)
	if err != nil {
		return err
	}
	if !accountExists {
		log.Printf("[DEBUG] Storage account %q not found, removing file %q from state", storageAccountName, d.Id())
		d.SetId("")
		return nil
	}

	exists, err := resourceArmStorageShareExists(d, meta)
	if err != nil {
		return err
	}

	if !exists {
		// Exists already removed this from state
		return nil
	}

	name := d.Get("name").(string)

	reference := fileClient.GetShareReference(name)
	url := reference.URL()
	if url == "" {
		log.Printf("[INFO] URL for %q is empty", name)
	}
	d.Set("url", url)

	return nil
}

func resourceArmStorageShareExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	armClient := meta.(*ArmClient)

	resourceGroupName := d.Get("resource_group_name").(string)
	storageAccountName := d.Get("storage_account_name").(string)

	fileClient, accountExists, err := armClient.getFileServiceClientForStorageAccount(resourceGroupName, storageAccountName)
	if err != nil {
		return false, err
	}
	if !accountExists {
		log.Printf("[DEBUG] Storage account %q not found, removing share %q from state", storageAccountName, d.Id())
		d.SetId("")
		return false, nil
	}

	name := d.Get("name").(string)

	log.Printf("[INFO] Checking for existence of share %q.", name)
	reference := fileClient.GetShareReference(name)
	exists, err := reference.Exists()
	if err != nil {
		return false, fmt.Errorf("Error testing existence of share %q: %s", name, err)
	}

	if !exists {
		log.Printf("[INFO] Share %q no longer exists, removing from state...", name)
		d.SetId("")
	}

	return exists, nil
}

func resourceArmStorageShareDelete(d *schema.ResourceData, meta interface{}) error {
	armClient := meta.(*ArmClient)

	resourceGroupName := d.Get("resource_group_name").(string)
	storageAccountName := d.Get("storage_account_name").(string)

	fileClient, accountExists, err := armClient.getFileServiceClientForStorageAccount(resourceGroupName, storageAccountName)
	if err != nil {
		return err
	}
	if !accountExists {
		log.Printf("[INFO]Storage Account %q doesn't exist so the file won't exist", storageAccountName)
		return nil
	}

	name := d.Get("name").(string)

	reference := fileClient.GetShareReference(name)
	options := &storage.FileRequestOptions{}

	if _, err = reference.DeleteIfExists(options); err != nil {
		return fmt.Errorf("Error deleting storage file %q: %s", name, err)
	}

	d.SetId("")
	return nil
}

//Following the naming convention as laid out in the docs https://msdn.microsoft.com/library/azure/dn167011.aspx
func validateArmStorageShareName(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)
	if !regexp.MustCompile(`^[0-9a-z-]+$`).MatchString(value) {
		errors = append(errors, fmt.Errorf(
			"only lowercase alphanumeric characters and hyphens allowed in %q: %q",
			k, value))
	}
	if len(value) < 3 || len(value) > 63 {
		errors = append(errors, fmt.Errorf(
			"%q must be between 3 and 63 characters: %q", k, value))
	}
	if regexp.MustCompile(`^-`).MatchString(value) {
		errors = append(errors, fmt.Errorf(
			"%q cannot begin with a hyphen: %q", k, value))
	}
	if regexp.MustCompile(`[-]{2,}`).MatchString(value) {
		errors = append(errors, fmt.Errorf(
			"%q does not allow consecutive hyphens: %q", k, value))
	}
	return
}
