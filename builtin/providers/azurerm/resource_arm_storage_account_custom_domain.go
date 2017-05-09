package azurerm

import (
	"fmt"
	"net/http"

	"github.com/Azure/azure-sdk-for-go/arm/storage"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceArmStorageAccountCustomDomain() *schema.Resource {
	return &schema.Resource{
		Create: resourceArmStorageAccountCustomDomainCreate,
		Read:   resourceArmStorageAccountCustomDomainRead,
		Update: resourceArmStorageAccountCustomDomainCreate,
		Delete: resourceArmStorageAccountCustomDomainDelete,

		Schema: map[string]*schema.Schema{
			"storage_account_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"custom_domain_name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"use_subdomain": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
			},
		},
	}
}

func resourceArmStorageAccountCustomDomainCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ArmClient).storageServiceClient

	storageAccountId := d.Get("storage_account_id").(string)
	thisResourceId := fmt.Sprintf("%s::custom_domain", storageAccountId)
	customDomainName := d.Get("custom_domain_name").(string)
	useSubdomain := d.Get("use_subdomain").(bool)

	id, err := parseAzureResourceID(storageAccountId)
	if err != nil {
		return err
	}
	storageAccountName := id.Path["storageAccounts"]
	storageAccountResourceGroup := id.ResourceGroup

	opts := storage.AccountUpdateParameters{
		Properties: &storage.AccountPropertiesUpdateParameters{
			CustomDomain: &storage.CustomDomain{
				Name:         &customDomainName,
				UseSubDomain: &useSubdomain,
			},
		},
	}

	accResp, err := client.Update(storageAccountResourceGroup, storageAccountName, opts)
	if err != nil {
		return fmt.Errorf("Error updating Azure Storage Account custom domain %q: %s", storageAccountName, err)
	}
	_, err = pollIndefinitelyAsNeeded(client.Client, accResp.Response.Response, http.StatusOK)
	if err != nil {
		return fmt.Errorf("Error updating Azure Storage Account custom domain %q: %s", storageAccountName, err)
	}

	d.SetId(thisResourceId)

	return resourceArmStorageAccountRead(d, meta)
}

func resourceArmStorageAccountCustomDomainRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ArmClient).storageServiceClient

	id, err := parseAzureResourceID(d.Id())
	if err != nil {
		return err
	}
	name := id.Path["storageAccounts"]
	resGroup := id.ResourceGroup

	resp, err := client.GetProperties(resGroup, name)
	if err != nil {
		if resp.StatusCode == http.StatusNoContent {
			d.SetId("")
			return nil
		}

		return fmt.Errorf("Error reading the state of AzureRM Storage Account Custom Domain %q: %s", name, err)
	}

	if resp.Properties.CustomDomain == nil {
		d.SetId("")
	} else {
		d.Set("custom_domain_name", resp.Properties.CustomDomain.Name)
		d.Set("use_subdomain", resp.Properties.CustomDomain.UseSubDomain)
	}

	return nil
}

func resourceArmStorageAccountCustomDomainDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ArmClient).storageServiceClient

	id, err := parseAzureResourceID(d.Id())
	if err != nil {
		return err
	}
	storageAccountName := id.Path["storageAccounts"]
	storageAccountResourceGroup := id.ResourceGroup

	empty := ""
	opts := storage.AccountUpdateParameters{
		Properties: &storage.AccountPropertiesUpdateParameters{
			CustomDomain: &storage.CustomDomain{
				Name: &empty,
			},
		},
	}

	accResp, err := client.Update(storageAccountResourceGroup, storageAccountName, opts)
	if err != nil {
		return fmt.Errorf("Error updating Azure Storage Account custom domain %q: %s", storageAccountName, err)
	}
	_, err = pollIndefinitelyAsNeeded(client.Client, accResp.Response.Response, http.StatusOK)
	if err != nil {
		return fmt.Errorf("Error updating Azure Storage Account custom domain %q: %s", storageAccountName, err)
	}

	d.SetId("")

	return nil
}
