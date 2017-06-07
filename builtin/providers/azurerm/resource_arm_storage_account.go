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
	"github.com/hashicorp/terraform/helper/validation"
)

// The KeySource of storage.Encryption appears to require this value
// for Encryption services to work
var storageAccountEncryptionSource = "Microsoft.Storage"

const blobStorageAccountDefaultAccessTier = "Hot"

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
				Type:             schema.TypeString,
				Required:         true,
				ForceNew:         true,
				DiffSuppressFunc: resourceAzurermResourceGroupNameDiffSuppress,
			},

			"location": locationSchema(),

			"account_kind": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				ValidateFunc: validation.StringInSlice([]string{
					string(storage.Storage),
					string(storage.BlobStorage),
				}, true),
				Default: string(storage.Storage),
			},

			"account_type": {
				Type:             schema.TypeString,
				Required:         true,
				ValidateFunc:     validateArmStorageAccountType,
				DiffSuppressFunc: ignoreCaseDiffSuppressFunc,
			},

			// Only valid for BlobStorage accounts, defaults to "Hot" in create function
			"access_tier": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ValidateFunc: validation.StringInSlice([]string{
					string(storage.Cool),
					string(storage.Hot),
				}, true),
			},

			"enable_blob_encryption": {
				Type:     schema.TypeBool,
				Optional: true,
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
	accountKind := d.Get("account_kind").(string)
	accountType := d.Get("account_type").(string)

	location := d.Get("location").(string)
	tags := d.Get("tags").(map[string]interface{})
	enableBlobEncryption := d.Get("enable_blob_encryption").(bool)

	sku := storage.Sku{
		Name: storage.SkuName(accountType),
	}

	opts := storage.AccountCreateParameters{
		Location: &location,
		Sku:      &sku,
		Tags:     expandTags(tags),
		Kind:     storage.Kind(accountKind),
		AccountPropertiesCreateParameters: &storage.AccountPropertiesCreateParameters{
			Encryption: &storage.Encryption{
				Services: &storage.EncryptionServices{
					Blob: &storage.EncryptionService{
						Enabled: &enableBlobEncryption,
					},
				},
				KeySource: &storageAccountEncryptionSource,
			},
		},
	}

	// AccessTier is only valid for BlobStorage accounts
	if accountKind == string(storage.BlobStorage) {
		accessTier, ok := d.GetOk("access_tier")
		if !ok {
			// default to "Hot"
			accessTier = blobStorageAccountDefaultAccessTier
		}

		opts.AccountPropertiesCreateParameters.AccessTier = storage.AccessTier(accessTier.(string))
	}

	// Create
	_, createError := storageClient.Create(resourceGroupName, storageAccountName, opts, make(chan struct{}))
	createErr := <-createError

	// The only way to get the ID back apparently is to read the resource again
	read, err := storageClient.GetProperties(resourceGroupName, storageAccountName)

	// Set the ID right away if we have one
	if err == nil && read.ID != nil {
		log.Printf("[INFO] storage account %q ID: %q", storageAccountName, *read.ID)
		d.SetId(*read.ID)
	}

	// If we had a create error earlier then we return with that error now.
	// We do this later here so that we can grab the ID above is possible.
	if createErr != nil {
		return fmt.Errorf(
			"Error creating Azure Storage Account '%s': %s",
			storageAccountName, createErr)
	}

	// Check the read error now that we know it would exist without a create err
	if err != nil {
		return err
	}

	// If we got no ID then the resource group doesn't yet exist
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

	if d.HasChange("access_tier") {
		accessTier := d.Get("access_tier").(string)

		opts := storage.AccountUpdateParameters{
			AccountPropertiesUpdateParameters: &storage.AccountPropertiesUpdateParameters{
				AccessTier: storage.AccessTier(accessTier),
			},
		}
		_, err := client.Update(resourceGroupName, storageAccountName, opts)
		if err != nil {
			return fmt.Errorf("Error updating Azure Storage Account access_tier %q: %s", storageAccountName, err)
		}

		d.SetPartial("access_tier")
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

	if d.HasChange("enable_blob_encryption") {
		enableBlobEncryption := d.Get("enable_blob_encryption").(bool)

		opts := storage.AccountUpdateParameters{
			AccountPropertiesUpdateParameters: &storage.AccountPropertiesUpdateParameters{
				Encryption: &storage.Encryption{
					Services: &storage.EncryptionServices{
						Blob: &storage.EncryptionService{
							Enabled: &enableBlobEncryption,
						},
					},
					KeySource: &storageAccountEncryptionSource,
				},
			},
		}
		_, err := client.Update(resourceGroupName, storageAccountName, opts)
		if err != nil {
			return fmt.Errorf("Error updating Azure Storage Account enable_blob_encryption %q: %s", storageAccountName, err)
		}

		d.SetPartial("enable_blob_encryption")
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
	d.Set("resource_group_name", resGroup)
	d.Set("primary_access_key", accessKeys[0].Value)
	d.Set("secondary_access_key", accessKeys[1].Value)
	d.Set("location", resp.Location)
	d.Set("account_kind", resp.Kind)
	d.Set("account_type", resp.Sku.Name)
	d.Set("primary_location", resp.AccountProperties.PrimaryLocation)
	d.Set("secondary_location", resp.AccountProperties.SecondaryLocation)

	if resp.AccountProperties.AccessTier != "" {
		d.Set("access_tier", resp.AccountProperties.AccessTier)
	}

	if resp.AccountProperties.PrimaryEndpoints != nil {
		d.Set("primary_blob_endpoint", resp.AccountProperties.PrimaryEndpoints.Blob)
		d.Set("primary_queue_endpoint", resp.AccountProperties.PrimaryEndpoints.Queue)
		d.Set("primary_table_endpoint", resp.AccountProperties.PrimaryEndpoints.Table)
		d.Set("primary_file_endpoint", resp.AccountProperties.PrimaryEndpoints.File)
	}

	if resp.AccountProperties.SecondaryEndpoints != nil {
		if resp.AccountProperties.SecondaryEndpoints.Blob != nil {
			d.Set("secondary_blob_endpoint", resp.AccountProperties.SecondaryEndpoints.Blob)
		} else {
			d.Set("secondary_blob_endpoint", "")
		}
		if resp.AccountProperties.SecondaryEndpoints.Queue != nil {
			d.Set("secondary_queue_endpoint", resp.AccountProperties.SecondaryEndpoints.Queue)
		} else {
			d.Set("secondary_queue_endpoint", "")
		}
		if resp.AccountProperties.SecondaryEndpoints.Table != nil {
			d.Set("secondary_table_endpoint", resp.AccountProperties.SecondaryEndpoints.Table)
		} else {
			d.Set("secondary_table_endpoint", "")
		}
	}

	if resp.AccountProperties.Encryption != nil {
		if resp.AccountProperties.Encryption.Services.Blob != nil {
			d.Set("enable_blob_encryption", resp.AccountProperties.Encryption.Services.Blob.Enabled)
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

		return res, string(res.AccountProperties.ProvisioningState), nil
	}
}
