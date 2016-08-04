package azurerm

import (
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/arm/storage"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceArmStorageAccount() *schema.Resource {
	return &schema.Resource{
		Create: resourceArmStorageAccountCreate,
		Read:   resourceArmStorageAccountRead,
		Update: resourceArmStorageAccountUpdate,
		Delete: resourceArmStorageAccountDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validateArmStorageAccountName,
			},

			"resource_group_name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"location": {
				Type:      schema.TypeString,
				Required:  true,
				ForceNew:  true,
				StateFunc: azureRMNormalizeLocation,
			},

			"account_type": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validateArmStorageAccountType,
			},

			"primary_location": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"secondary_location": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"primary_blob_endpoint": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"secondary_blob_endpoint": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"primary_queue_endpoint": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"secondary_queue_endpoint": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"primary_table_endpoint": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"secondary_table_endpoint": {
				Type:     schema.TypeString,
				Computed: true,
			},

			// NOTE: The API does not appear to expose a secondary file endpoint
			"primary_file_endpoint": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"primary_access_key": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"secondary_access_key": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"tags": tagsSchema(),
		},
	}
}

func resourceArmStorageAccountCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ArmClient)
	storageClient := client.storageServiceClient

	resourceGroupName := d.Get("resource_group_name").(string)
	storageAccountName := d.Get("name").(string)
	accountType := d.Get("account_type").(string)
	location := d.Get("location").(string)
	tags := d.Get("tags").(map[string]interface{})

	sku := storage.Sku{
		Name: storage.SkuName(accountType),
	}

	opts := storage.AccountCreateParameters{
		Location: &location,
		Sku:      &sku,
		Tags:     expandTags(tags),
	}

	_, err := storageClient.Create(resourceGroupName, storageAccountName, opts, make(chan struct{}))
	if err != nil {
		return fmt.Errorf("Error creating Azure Storage Account '%s': %s", storageAccountName, err)
	}

	// The only way to get the ID back apparently is to read the resource again
	read, err := storageClient.GetProperties(resourceGroupName, storageAccountName)
	if err != nil {
		return err
	}
	if read.ID == nil {
		return fmt.Errorf("Cannot read Storage Account %s (resource group %s) ID",
			storageAccountName, resourceGroupName)
	}

	log.Printf("[DEBUG] Waiting for Storage Account (%s) to become available", storageAccountName)
	stateConf := &resource.StateChangeConf{
		Pending:    []string{"Updating", "Creating"},
		Target:     []string{"Succeeded"},
		Refresh:    storageAccountStateRefreshFunc(client, resourceGroupName, storageAccountName),
		Timeout:    30 * time.Minute,
		MinTimeout: 15 * time.Second,
	}
	if _, err := stateConf.WaitForState(); err != nil {
		return fmt.Errorf("Error waiting for Storage Account (%s) to become available: %s", storageAccountName, err)
	}

	d.SetId(*read.ID)

	return resourceArmStorageAccountRead(d, meta)
}

// resourceArmStorageAccountUpdate is unusual in the ARM API where most resources have a combined
// and idempotent operation for CreateOrUpdate. In particular updating all of the parameters
// available requires a call to Update per parameter...
func resourceArmStorageAccountUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ArmClient).storageServiceClient
	id, err := parseAzureResourceID(d.Id())
	if err != nil {
		return err
	}
	storageAccountName := id.Path["storageAccounts"]
	resourceGroupName := id.ResourceGroup

	d.Partial(true)

	if d.HasChange("account_type") {
		accountType := d.Get("account_type").(string)

		sku := storage.Sku{
			Name: storage.SkuName(accountType),
		}

		opts := storage.AccountUpdateParameters{
			Sku: &sku,
		}
		_, err := client.Update(resourceGroupName, storageAccountName, opts)
		if err != nil {
			return fmt.Errorf("Error updating Azure Storage Account type %q: %s", storageAccountName, err)
		}

		d.SetPartial("account_type")
	}

	if d.HasChange("tags") {
		tags := d.Get("tags").(map[string]interface{})

		opts := storage.AccountUpdateParameters{
			Tags: expandTags(tags),
		}
		_, err := client.Update(resourceGroupName, storageAccountName, opts)
		if err != nil {
			return fmt.Errorf("Error updating Azure Storage Account tags %q: %s", storageAccountName, err)
		}

		d.SetPartial("tags")
	}

	d.Partial(false)
	return nil
}

func resourceArmStorageAccountRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ArmClient).storageServiceClient

	id, err := parseAzureResourceID(d.Id())
	if err != nil {
		return err
	}
	name := id.Path["storageAccounts"]
	resGroup := id.ResourceGroup

	resp, err := client.GetProperties(resGroup, name)
	if err != nil {
		if resp.StatusCode == http.StatusNotFound {
			d.SetId("")
			return nil
		}

		return fmt.Errorf("Error reading the state of AzureRM Storage Account %q: %s", name, err)
	}

	keys, err := client.ListKeys(resGroup, name)
	if err != nil {
		return err
	}

	accessKeys := *keys.Keys
	d.Set("primary_access_key", accessKeys[0].Value)
	d.Set("secondary_access_key", accessKeys[1].Value)
	d.Set("location", resp.Location)
	d.Set("account_type", resp.Sku.Name)
	d.Set("primary_location", resp.Properties.PrimaryLocation)
	d.Set("secondary_location", resp.Properties.SecondaryLocation)

	if resp.Properties.PrimaryEndpoints != nil {
		d.Set("primary_blob_endpoint", resp.Properties.PrimaryEndpoints.Blob)
		d.Set("primary_queue_endpoint", resp.Properties.PrimaryEndpoints.Queue)
		d.Set("primary_table_endpoint", resp.Properties.PrimaryEndpoints.Table)
		d.Set("primary_file_endpoint", resp.Properties.PrimaryEndpoints.File)
	}

	if resp.Properties.SecondaryEndpoints != nil {
		if resp.Properties.SecondaryEndpoints.Blob != nil {
			d.Set("secondary_blob_endpoint", resp.Properties.SecondaryEndpoints.Blob)
		} else {
			d.Set("secondary_blob_endpoint", "")
		}
		if resp.Properties.SecondaryEndpoints.Queue != nil {
			d.Set("secondary_queue_endpoint", resp.Properties.SecondaryEndpoints.Queue)
		} else {
			d.Set("secondary_queue_endpoint", "")
		}
		if resp.Properties.SecondaryEndpoints.Table != nil {
			d.Set("secondary_table_endpoint", resp.Properties.SecondaryEndpoints.Table)
		} else {
			d.Set("secondary_table_endpoint", "")
		}
	}

	d.Set("name", resp.Name)

	flattenAndSetTags(d, resp.Tags)

	return nil
}

func resourceArmStorageAccountDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ArmClient).storageServiceClient

	id, err := parseAzureResourceID(d.Id())
	if err != nil {
		return err
	}
	name := id.Path["storageAccounts"]
	resGroup := id.ResourceGroup

	_, err = client.Delete(resGroup, name)
	if err != nil {
		return fmt.Errorf("Error issuing AzureRM delete request for storage account %q: %s", name, err)
	}

	return nil
}

func validateArmStorageAccountName(v interface{}, k string) (ws []string, es []error) {
	input := v.(string)

	if !regexp.MustCompile(`\A([a-z0-9]{3,24})\z`).MatchString(input) {
		es = append(es, fmt.Errorf("name can only consist of lowercase letters and numbers, and must be between 3 and 24 characters long"))
	}

	return
}

func validateArmStorageAccountType(v interface{}, k string) (ws []string, es []error) {
	validAccountTypes := []string{"standard_lrs", "standard_zrs",
		"standard_grs", "standard_ragrs", "premium_lrs"}

	input := strings.ToLower(v.(string))

	for _, valid := range validAccountTypes {
		if valid == input {
			return
		}
	}

	es = append(es, fmt.Errorf("Invalid storage account type %q", input))
	return
}

func storageAccountStateRefreshFunc(client *ArmClient, resourceGroupName string, storageAccountName string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		res, err := client.storageServiceClient.GetProperties(resourceGroupName, storageAccountName)
		if err != nil {
			return nil, "", fmt.Errorf("Error issuing read request in storageAccountStateRefreshFunc to Azure ARM for Storage Account '%s' (RG: '%s'): %s", storageAccountName, resourceGroupName, err)
		}

		return res, string(res.Properties.ProvisioningState), nil
	}
}
