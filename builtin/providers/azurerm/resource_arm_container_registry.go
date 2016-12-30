package azurerm

import (
	"fmt"
	"log"

	"net/http"

	"github.com/Azure/azure-sdk-for-go/arm/containerregistry"
	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceArmContainerRegistry() *schema.Resource {
	return &schema.Resource{
		Create: resourceArmContainerRegistryCreate,
		Read:   resourceArmContainerRegistryRead,
		Update: resourceArmContainerRegistryCreate,
		Delete: resourceArmContainerRegistryDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"resource_group_name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"location": locationSchema(),

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

	adminUserEnabled := d.Get("admin_enabled").(bool)
	tags := d.Get("tags").(map[string]interface{})

	parameters := containerregistry.Registry{
		Location: &location,
		RegistryProperties: &containerregistry.RegistryProperties{
			AdminUserEnabled: &adminUserEnabled,
		},
		Tags: expandTags(tags),
	}

	if v, ok := d.GetOk("storage_account"); ok {
		accounts := v.(*schema.Set).List()
		account := accounts[0].(map[string]interface{})
		storageAccountName := account["name"].(string)
		storageAccountAccessKey := account["access_key"].(string)
		parameters.RegistryProperties.StorageAccount = &containerregistry.StorageAccountProperties{
			Name:      &storageAccountName,
			AccessKey: &storageAccountAccessKey,
		}
	}

	_, err := client.CreateOrUpdate(resourceGroup, name, parameters)
	if err != nil {
		return err
	}

	read, err := client.GetProperties(resourceGroup, name)
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

	resp, err := client.GetProperties(resourceGroup, name)
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

	if resp.StorageAccount != nil {
		flattenArmContainerRegistryStorageAccount(d, resp.StorageAccount)
	}

	if *resp.AdminUserEnabled {
		credsResp, err := client.GetCredentials(resourceGroup, name)
		if err != nil {
			return fmt.Errorf("Error making Read request on Azure Container Registry %s for Credentials: %s", name, err)
		}

		d.Set("admin_username", credsResp.Username)
		d.Set("admin_password", credsResp.Password)
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
		return fmt.Errorf("Error issuing Azure ARM delete request of ContainerRegistry '%s': %s", name, err)
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
