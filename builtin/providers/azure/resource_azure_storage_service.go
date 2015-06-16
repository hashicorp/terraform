package azure

import (
	"encoding/base64"
	"fmt"
	"log"

	"github.com/Azure/azure-sdk-for-go/management"
	"github.com/Azure/azure-sdk-for-go/management/storageservice"
	"github.com/hashicorp/terraform/helper/schema"
)

// resourceAzureStorageService returns the *schema.Resource associated
// to an Azure hosted service.
func resourceAzureStorageService() *schema.Resource {
	return &schema.Resource{
		Create: resourceAzureStorageServiceCreate,
		Read:   resourceAzureStorageServiceRead,
		Exists: resourceAzureStorageServiceExists,
		Delete: resourceAzureStorageServiceDelete,

		Schema: map[string]*schema.Schema{
			// General attributes:
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
				// TODO(aznashwan): constrain name in description
				Description: parameterDescriptions["name"],
			},
			"location": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: parameterDescriptions["location"],
			},
			"label": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Default:     "Made by Terraform.",
				Description: parameterDescriptions["label"],
			},
			"description": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Description: parameterDescriptions["description"],
			},
			// Functional attributes:
			"account_type": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: parameterDescriptions["account_type"],
			},
			"affinity_group": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Description: parameterDescriptions["affinity_group"],
			},
			"properties": &schema.Schema{
				Type:     schema.TypeMap,
				Optional: true,
				ForceNew: true,
				Elem:     schema.TypeString,
			},
			// Computed attributes:
			"url": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"primary_key": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"secondary_key": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

// resourceAzureStorageServiceCreate does all the necessary API calls to
// create a new Azure storage service.
func resourceAzureStorageServiceCreate(d *schema.ResourceData, meta interface{}) error {
	azureClient := meta.(*Client)
	mgmtClient := azureClient.mgmtClient
	storageServiceClient := azureClient.storageServiceClient

	// get all the values:
	log.Println("[INFO] Creating Azure Storage Service creation parameters.")
	name := d.Get("name").(string)
	location := d.Get("location").(string)
	accountType := storageservice.AccountType(d.Get("account_type").(string))
	affinityGroup := d.Get("affinity_group").(string)
	description := d.Get("description").(string)
	label := base64.StdEncoding.EncodeToString([]byte(d.Get("label").(string)))
	var props []storageservice.ExtendedProperty
	if given := d.Get("properties").(map[string]interface{}); len(given) > 0 {
		props = []storageservice.ExtendedProperty{}
		for k, v := range given {
			props = append(props, storageservice.ExtendedProperty{
				Name:  k,
				Value: v.(string),
			})
		}
	}

	// create parameters and send request:
	log.Println("[INFO] Sending Storage Service creation request to Azure.")
	reqID, err := storageServiceClient.CreateStorageService(
		storageservice.StorageAccountCreateParameters{
			ServiceName:   name,
			Location:      location,
			Description:   description,
			Label:         label,
			AffinityGroup: affinityGroup,
			AccountType:   accountType,
			ExtendedProperties: storageservice.ExtendedPropertyList{
				ExtendedProperty: props,
			},
		})
	if err != nil {
		return fmt.Errorf("Failed to create Azure storage service %s: %s", name, err)
	}
	err = mgmtClient.WaitForOperation(reqID, nil)
	if err != nil {
		return fmt.Errorf("Failed creating storage service %s: %s", name, err)
	}

	d.SetId(name)
	return resourceAzureStorageServiceRead(d, meta)
}

// resourceAzureStorageServiceRead does all the necessary API calls to
// read the state of the storage service off Azure.
func resourceAzureStorageServiceRead(d *schema.ResourceData, meta interface{}) error {
	storageServiceClient := meta.(*Client).storageServiceClient

	// get our storage service:
	log.Println("[INFO] Sending query about storage service to Azure.")
	name := d.Get("name").(string)
	storsvc, err := storageServiceClient.GetStorageService(name)
	if err != nil {
		if !management.IsResourceNotFoundError(err) {
			return fmt.Errorf("Failed to query about Azure about storage service: %s", err)
		} else {
			// it means that the resource has been deleted from Azure
			// in the meantime and we must remove its associated Resource.
			d.SetId("")
			return nil

		}
	}

	// read values:
	d.Set("url", storsvc.URL)
	log.Println("[INFO] Querying keys of Azure storage service.")
	keys, err := storageServiceClient.GetStorageServiceKeys(name)
	if err != nil {
		return fmt.Errorf("Failed querying keys for Azure storage service: %s", err)
	}
	d.Set("primary_key", keys.PrimaryKey)
	d.Set("secondary_key", keys.SecondaryKey)

	return nil
}

// resourceAzureStorageServiceExists does all the necessary API calls to
// check if the storage service exists on Azure.
func resourceAzureStorageServiceExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	storageServiceClient := meta.(*Client).storageServiceClient

	// get our storage service:
	log.Println("[INFO] Sending query about storage service to Azure.")
	name := d.Get("name").(string)
	_, err := storageServiceClient.GetStorageService(name)
	if err != nil {
		if !management.IsResourceNotFoundError(err) {
			return false, fmt.Errorf("Failed to query about Azure about storage service: %s", err)
		} else {
			// it means that the resource has been deleted from Azure
			// in the meantime and we must remove its associated Resource.
			d.SetId("")
			return false, nil

		}
	}

	return true, nil
}

// resourceAzureStorageServiceDelete does all the necessary API calls to
// delete the storage service off Azure.
func resourceAzureStorageServiceDelete(d *schema.ResourceData, meta interface{}) error {
	azureClient := meta.(*Client)
	mgmtClient := azureClient.mgmtClient
	storageServiceClient := azureClient.storageServiceClient

	// issue the deletion:
	name := d.Get("name").(string)
	log.Println("[INFO] Issuing delete of storage service off Azure.")
	reqID, err := storageServiceClient.DeleteStorageService(name)
	if err != nil {
		return fmt.Errorf("Error whilst issuing deletion of storage service off Azure: %s", err)
	}
	err = mgmtClient.WaitForOperation(reqID, nil)
	if err != nil {
		return fmt.Errorf("Error whilst deleting storage service off Azure: %s", err)
	}

	d.SetId("")
	return nil
}
