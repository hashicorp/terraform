package azurerm

import (
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/arm/resources"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceArmResourceGroup() *schema.Resource {
	return &schema.Resource{
		Create: resourceArmResourceGroupCreate,
		Read:   resourceArmResourceGroupRead,
		Exists: resourceArmResourceGroupExists,
		Delete: resourceArmResourceGroupDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validateArmResourceGroupName,
			},
			"location": &schema.Schema{
				Type:      schema.TypeString,
				Required:  true,
				ForceNew:  true,
				StateFunc: azureRMNormalizeLocation,
			},
		},
	}
}

func validateArmResourceGroupName(v interface{}, k string) (ws []string, es []error) {
	value := v.(string)

	if len(value) > 80 {
		es = append(es, fmt.Errorf("%q may not exceed 80 characters in length", k))
	}

	if strings.HasSuffix(value, ".") {
		es = append(es, fmt.Errorf("%q may not end with a period", k))
	}

	if matched := regexp.MustCompile(`[\(\)\.a-zA-Z0-9_-]`).Match([]byte(value)); !matched {
		es = append(es, fmt.Errorf("%q may only contain alphanumeric characters, dash, underscores, parentheses and periods", k))
	}

	return
}

func resourceArmResourceGroupCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ArmClient)
	resGroupClient := client.resourceGroupClient

	name := d.Get("name").(string)
	location := d.Get("location").(string)

	rg := resources.ResourceGroup{
		Name:     &name,
		Location: &location,
	}

	resp, err := resGroupClient.CreateOrUpdate(name, rg)
	if err != nil {
		return fmt.Errorf("Error issuing Azure ARM create request for resource group '%s': %s", name, err)
	}

	d.SetId(*resp.ID)

	log.Printf("[DEBUG] Waiting for Resource Group (%s) to become available", name)
	stateConf := &resource.StateChangeConf{
		Pending: []string{"Accepted"},
		Target:  "Succeeded",
		Refresh: resourceGroupStateRefreshFunc(client, name),
		Timeout: 10 * time.Minute,
	}
	if _, err := stateConf.WaitForState(); err != nil {
		return fmt.Errorf("Error waiting for Resource Group (%s) to become available: %s", name, err)
	}

	return resourceArmResourceGroupRead(d, meta)
}

func resourceArmResourceGroupRead(d *schema.ResourceData, meta interface{}) error {
	resGroupClient := meta.(*ArmClient).resourceGroupClient

	id, err := parseAzureResourceID(d.Id())
	if err != nil {
		return err
	}
	name := id.ResourceGroup

	res, err := resGroupClient.Get(name)
	if err != nil {
		if res.StatusCode == http.StatusNotFound {
			d.SetId("")
			return nil
		}
		return fmt.Errorf("Error issuing read request to Azure ARM for resource group '%s': %s", name, err)
	}

	d.Set("name", res.Name)
	d.Set("location", res.Location)

	return nil
}

func resourceArmResourceGroupExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	resGroupClient := meta.(*ArmClient).resourceGroupClient

	id, err := parseAzureResourceID(d.Id())
	if err != nil {
		return false, err
	}
	name := id.ResourceGroup

	resp, err := resGroupClient.CheckExistence(name)
	if err != nil {
		if resp.StatusCode != 200 {
			return false, err
		}

		return true, nil
	}

	return true, nil
}

func resourceArmResourceGroupDelete(d *schema.ResourceData, meta interface{}) error {
	resGroupClient := meta.(*ArmClient).resourceGroupClient

	id, err := parseAzureResourceID(d.Id())
	if err != nil {
		return err
	}
	name := id.ResourceGroup

	_, err = resGroupClient.Delete(name)
	if err != nil {
		return err
	}

	return nil
}

func resourceGroupStateRefreshFunc(client *ArmClient, id string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		res, err := client.resourceGroupClient.Get(id)
		if err != nil {
			return nil, "", fmt.Errorf("Error issuing read request in resourceGroupStateRefreshFunc to Azure ARM for resource group '%s': %s", id, err)
		}

		return res, *res.Properties.ProvisioningState, nil
	}
}
