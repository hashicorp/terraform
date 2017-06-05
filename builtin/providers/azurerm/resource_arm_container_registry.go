package azurerm

import (
	"fmt"
	"log"

	"net/http"

	"regexp"

	"github.com/Azure/azure-sdk-for-go/arm/containerregistry"
	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
	"github.com/jen20/riviera/azure"
)

func resourceArmContainerRegistry() *schema.Resource {
	return &schema.Resource{
		Create: resourceArmContainerRegistryCreate,
		Read:   resourceArmContainerRegistryRead,
		Update: resourceArmContainerRegistryUpdate,
		Delete: resourceArmContainerRegistryDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},
		MigrateState:  resourceAzureRMContainerRegistryMigrateState,
		SchemaVersion: 1,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validateAzureRMContainerRegistryName,
			},

			"resource_group_name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"location": locationSchema(),

			"sku": {
				Type:             schema.TypeString,
				Optional:         true,
				ForceNew:         true,
				Default:          string(containerregistry.Basic),
				DiffSuppressFunc: ignoreCaseDiffSuppressFunc,
				ValidateFunc: validation.StringInSlice([]string{
					string(containerregistry.Basic),
				}, true),
			},

			"admin_enabled": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},

			"storage_account": {
				Type:     schema.TypeSet,
				Required: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:     schema.TypeString,
							Required: true,
						},

						"access_key": {
							Type:      schema.TypeString,
							Required:  true,
							Sensitive: true,
						},
					},
				},
			},

			"login_server": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"admin_username": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"admin_password": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"tags": tagsSchema(),
		},
	}
}

func resourceArmContainerRegistryCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ArmClient).containerRegistryClient
	log.Printf("[INFO] preparing arguments for AzureRM Container Registry creation.")

	resourceGroup := d.Get("resource_group_name").(string)
	name := d.Get("name").(string)
	location := d.Get("location").(string)
	sku := d.Get("sku").(string)

	adminUserEnabled := d.Get("admin_enabled").(bool)
	tags := d.Get("tags").(map[string]interface{})

	parameters := containerregistry.RegistryCreateParameters{
		Location: &location,
		Sku: &containerregistry.Sku{
			Name: &sku,
			Tier: containerregistry.SkuTier(sku),
		},
		RegistryPropertiesCreateParameters: &containerregistry.RegistryPropertiesCreateParameters{
			AdminUserEnabled: &adminUserEnabled,
		},
		Tags: expandTags(tags),
	}

	accounts := d.Get("storage_account").(*schema.Set).List()
	account := accounts[0].(map[string]interface{})
	storageAccountName := account["name"].(string)
	storageAccountAccessKey := account["access_key"].(string)
	parameters.RegistryPropertiesCreateParameters.StorageAccount = &containerregistry.StorageAccountParameters{
		Name:      azure.String(storageAccountName),
		AccessKey: azure.String(storageAccountAccessKey),
	}

	_, error := client.Create(resourceGroup, name, parameters, make(<-chan struct{}))
	err := <-error
	if err != nil {
		return err
	}

	read, err := client.Get(resourceGroup, name)
	if err != nil {
		return err
	}

	if read.ID == nil {
		return fmt.Errorf("Cannot read Container Registry %s (resource group %s) ID", name, resourceGroup)
	}

	d.SetId(*read.ID)

	return resourceArmContainerRegistryRead(d, meta)
}

func resourceArmContainerRegistryUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ArmClient).containerRegistryClient
	log.Printf("[INFO] preparing arguments for AzureRM Container Registry update.")

	resourceGroup := d.Get("resource_group_name").(string)
	name := d.Get("name").(string)

	accounts := d.Get("storage_account").(*schema.Set).List()
	account := accounts[0].(map[string]interface{})
	storageAccountName := account["name"].(string)
	storageAccountAccessKey := account["access_key"].(string)

	adminUserEnabled := d.Get("admin_enabled").(bool)
	tags := d.Get("tags").(map[string]interface{})

	parameters := containerregistry.RegistryUpdateParameters{
		RegistryPropertiesUpdateParameters: &containerregistry.RegistryPropertiesUpdateParameters{
			AdminUserEnabled: &adminUserEnabled,
			StorageAccount: &containerregistry.StorageAccountParameters{
				Name:      azure.String(storageAccountName),
				AccessKey: azure.String(storageAccountAccessKey),
			},
		},
		Tags: expandTags(tags),
	}

	_, err := client.Update(resourceGroup, name, parameters)
	if err != nil {
		return err
	}

	read, err := client.Get(resourceGroup, name)
	if err != nil {
		return err
	}

	if read.ID == nil {
		return fmt.Errorf("Cannot read Container Registry %s (resource group %s) ID", name, resourceGroup)
	}

	d.SetId(*read.ID)

	return resourceArmContainerRegistryRead(d, meta)
}

func resourceArmContainerRegistryRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ArmClient).containerRegistryClient

	id, err := parseAzureResourceID(d.Id())
	if err != nil {
		return err
	}
	resourceGroup := id.ResourceGroup
	name := id.Path["registries"]

	resp, err := client.Get(resourceGroup, name)
	if err != nil {
		return fmt.Errorf("Error making Read request on Azure Container Registry %s: %s", name, err)
	}
	if resp.StatusCode == http.StatusNotFound {
		d.SetId("")
		return nil
	}

	d.Set("name", resp.Name)
	d.Set("resource_group_name", resourceGroup)
	d.Set("location", azureRMNormalizeLocation(*resp.Location))
	d.Set("admin_enabled", resp.AdminUserEnabled)
	d.Set("login_server", resp.LoginServer)

	if resp.Sku != nil {
		d.Set("sku", string(resp.Sku.Tier))
	}

	if resp.StorageAccount != nil {
		flattenArmContainerRegistryStorageAccount(d, resp.StorageAccount)
	}

	if *resp.AdminUserEnabled {
		credsResp, err := client.ListCredentials(resourceGroup, name)
		if err != nil {
			return fmt.Errorf("Error making Read request on Azure Container Registry %s for Credentials: %s", name, err)
		}

		d.Set("admin_username", credsResp.Username)
		for _, v := range *credsResp.Passwords {
			d.Set("admin_password", v.Value)
			break
		}
	} else {
		d.Set("admin_username", "")
		d.Set("admin_password", "")
	}

	flattenAndSetTags(d, resp.Tags)

	return nil
}

func resourceArmContainerRegistryDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ArmClient).containerRegistryClient

	id, err := parseAzureResourceID(d.Id())
	if err != nil {
		return err
	}
	resourceGroup := id.ResourceGroup
	name := id.Path["registries"]

	resp, err := client.Delete(resourceGroup, name)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Error issuing Azure ARM delete request of Container Registry '%s': %s", name, err)
	}

	return nil
}

func flattenArmContainerRegistryStorageAccount(d *schema.ResourceData, properties *containerregistry.StorageAccountProperties) {
	storageAccounts := schema.Set{
		F: resourceAzureRMContainerRegistryStorageAccountHash,
	}

	storageAccount := map[string]interface{}{}
	storageAccount["name"] = properties.Name
	storageAccounts.Add(storageAccount)

	d.Set("storage_account", &storageAccounts)
}

func resourceAzureRMContainerRegistryStorageAccountHash(v interface{}) int {
	m := v.(map[string]interface{})
	name := m["name"].(*string)
	return hashcode.String(*name)
}

func validateAzureRMContainerRegistryName(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)
	if !regexp.MustCompile(`^[a-zA-Z0-9]+$`).MatchString(value) {
		errors = append(errors, fmt.Errorf(
			"alpha numeric characters only are allowed in %q: %q", k, value))
	}

	if 5 > len(value) {
		errors = append(errors, fmt.Errorf("%q cannot be less than 5 characters: %q", k, value))
	}

	if len(value) >= 50 {
		errors = append(errors, fmt.Errorf("%q cannot be longer than 50 characters: %q %d", k, value, len(value)))
	}

	return
}
