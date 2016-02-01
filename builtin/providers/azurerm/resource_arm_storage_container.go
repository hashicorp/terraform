package azurerm

import (
	"fmt"
	"log"
	"strings"

	"github.com/Azure/azure-sdk-for-go/storage"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceArmStorageContainer() *schema.Resource {
	return &schema.Resource{
		Create: resourceArmStorageContainerCreate,
		Read:   resourceArmStorageContainerRead,
		Exists: resourceArmStorageContainerExists,
		Delete: resourceArmStorageContainerDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"resource_group_name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"storage_account_name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"container_access_type": &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				ForceNew:     true,
				Default:      "private",
				ValidateFunc: validateArmStorageContainerAccessType,
			},
			"properties": &schema.Schema{
				Type:     schema.TypeMap,
				Computed: true,
			},
		},
	}
}

func validateArmStorageContainerAccessType(v interface{}, k string) (ws []string, errors []error) {
	value := strings.ToLower(v.(string))
	validTypes := map[string]struct{}{
		"private":   struct{}{},
		"blob":      struct{}{},
		"container": struct{}{},
	}

	if _, ok := validTypes[value]; !ok {
		errors = append(errors, fmt.Errorf("Storage container access type %q is invalid, must be %q, %q or %q", value, "private", "blob", "page"))
	}
	return
}

func resourceArmStorageContainerCreate(d *schema.ResourceData, meta interface{}) error {
	armClient := meta.(*ArmClient)

	resourceGroupName := d.Get("resource_group_name").(string)
	storageAccountName := d.Get("storage_account_name").(string)

	blobClient, err := armClient.getBlobStorageClientForStorageAccount(resourceGroupName, storageAccountName)
	if err != nil {
		return err
	}

	name := d.Get("name").(string)

	var accessType storage.ContainerAccessType
	if d.Get("container_access_type").(string) == "private" {
		accessType = storage.ContainerAccessType("")
	} else {
		accessType = storage.ContainerAccessType(d.Get("container_access_type").(string))
	}

	log.Printf("[INFO] Creating container %q in storage account %q.", name, storageAccountName)
	_, err = blobClient.CreateContainerIfNotExists(name, accessType)
	if err != nil {
		return fmt.Errorf("Error creating container %q in storage account %q: %s", name, storageAccountName, err)
	}

	d.SetId(name)
	return resourceArmStorageContainerRead(d, meta)
}

// resourceAzureStorageContainerRead does all the necessary API calls to
// read the status of the storage container off Azure.
func resourceArmStorageContainerRead(d *schema.ResourceData, meta interface{}) error {
	armClient := meta.(*ArmClient)

	resourceGroupName := d.Get("resource_group_name").(string)
	storageAccountName := d.Get("storage_account_name").(string)

	blobClient, err := armClient.getBlobStorageClientForStorageAccount(resourceGroupName, storageAccountName)
	if err != nil {
		return err
	}

	name := d.Get("name").(string)
	containers, err := blobClient.ListContainers(storage.ListContainersParameters{
		Prefix:  name,
		Timeout: 90,
	})
	if err != nil {
		return fmt.Errorf("Failed to retrieve storage containers in account %q: %s", name, err)
	}

	var found bool
	for _, cont := range containers.Containers {
		if cont.Name == name {
			found = true

			props := make(map[string]interface{})
			props["last_modified"] = cont.Properties.LastModified
			props["lease_status"] = cont.Properties.LeaseStatus
			props["lease_state"] = cont.Properties.LeaseState
			props["lease_duration"] = cont.Properties.LeaseDuration

			d.Set("properties", props)
		}
	}

	if !found {
		log.Printf("[INFO] Storage container %q does not exist in account %q, removing from state...", name, storageAccountName)
		d.SetId("")
	}

	return nil
}

func resourceArmStorageContainerExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	armClient := meta.(*ArmClient)

	resourceGroupName := d.Get("resource_group_name").(string)
	storageAccountName := d.Get("storage_account_name").(string)

	blobClient, err := armClient.getBlobStorageClientForStorageAccount(resourceGroupName, storageAccountName)
	if err != nil {
		return false, err
	}

	name := d.Get("name").(string)

	log.Printf("[INFO] Checking existence of storage container %q in storage account %q", name, storageAccountName)
	exists, err := blobClient.ContainerExists(name)
	if err != nil {
		return false, fmt.Errorf("Error querying existence of storage container %q in storage account %q: %s", name, storageAccountName, err)
	}

	if !exists {
		log.Printf("[INFO] Storage container %q does not exist in account %q, removing from state...", name, storageAccountName)
		d.SetId("")
	}

	return exists, nil
}

// resourceAzureStorageContainerDelete does all the necessary API calls to
// delete a storage container off Azure.
func resourceArmStorageContainerDelete(d *schema.ResourceData, meta interface{}) error {
	armClient := meta.(*ArmClient)

	resourceGroupName := d.Get("resource_group_name").(string)
	storageAccountName := d.Get("storage_account_name").(string)

	blobClient, err := armClient.getBlobStorageClientForStorageAccount(resourceGroupName, storageAccountName)
	if err != nil {
		return err
	}

	name := d.Get("name").(string)

	log.Printf("[INFO] Deleting storage container %q in account %q", name, storageAccountName)
	if _, err := blobClient.DeleteContainerIfExists(name); err != nil {
		return fmt.Errorf("Error deleting storage container %q from storage account %q: %s", name, storageAccountName, err)
	}

	d.SetId("")
	return nil
}
