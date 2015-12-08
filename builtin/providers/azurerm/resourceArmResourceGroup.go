package azurerm

import (
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/arm/resources"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

// resourceArmResourceGroup returns the *schema.Resource
// associated to resource group resources on ARM.
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

// validateArmResourceGroupName validates inputs to the name argument against the requirements
// documented in the ARM REST API guide: http://bit.ly/1NEXclG
func validateArmResourceGroupName(v interface{}, k string) (ws []string, es []error) {
	value := v.(string)

	if len(value) > 80 {
		es = append(es, fmt.Errorf("%q may not exceed 80 characters in length", k))
	}

	if strings.HasSuffix(value, ".") {
		es = append(es, fmt.Errorf("%q may not end with a period", k))
	}

	if matched := regexp.MustCompile(`^[\(\)\.a-zA-Z0-9_-]$`).Match([]byte(value)); !matched {
		es = append(es, fmt.Errorf("%q may only contain alphanumeric characters, dash, underscores, parentheses and periods", k))
	}

	return
}

// resourceArmResourceGroupCreate goes ahead and creates the specified ARM resource group.
func resourceArmResourceGroupCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ArmClient)
	resGroupClient := client.resourceGroupClient

	name := d.Get("name").(string)
	location := d.Get("location").(string)

	log.Printf("[INFO] Issuing Azure ARM creation request for resource group '%s'.", name)

	rg := resources.ResourceGroup{
		Name:     &name,
		Location: &location,
	}

	_, err := resGroupClient.CreateOrUpdate(name, rg)
	if err != nil {
		return fmt.Errorf("Error issuing Azure ARM create request for resource group '%s': %s", name, err)
	}

	d.SetId(*rg.Name)

	// Wait for the resource group to become available
	// TODO(jen20): Is there any need for this?
	log.Printf("[DEBUG] Waiting for Resource Group (%s) to become available", d.Id())
	stateConf := &resource.StateChangeConf{
		Pending: []string{"Accepted"},
		Target:  "Succeeded",
		Refresh: resourceGroupStateRefreshFunc(client, d.Id()),
		Timeout: 10 * time.Minute,
	}
	if _, err := stateConf.WaitForState(); err != nil {
		return fmt.Errorf("Error waiting for Resource Group (%s) to become available: %s", d.Id(), err)
	}

	return resourceArmResourceGroupRead(d, meta)
}

// resourceArmResourceGroupRead goes ahead and reads the state of the corresponding ARM resource group.
func resourceArmResourceGroupRead(d *schema.ResourceData, meta interface{}) error {
	resGroupClient := meta.(*ArmClient).resourceGroupClient

	name := d.Id()
	log.Printf("[INFO] Issuing read request to Azure ARM for resource group '%s'.", name)

	res, err := resGroupClient.Get(name)
	if err != nil {
		return fmt.Errorf("Error issuing read request to Azure ARM for resource group '%s': %s", name, err)
	}

	d.Set("name", *res.Name)
	d.Set("location", *res.Location)

	return nil
}

// resourceArmResourceGroupExists goes ahead and checks for the existence of the correspoding ARM resource group.
func resourceArmResourceGroupExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	resGroupClient := meta.(*ArmClient).resourceGroupClient

	name := d.Id()

	resp, err := resGroupClient.CheckExistence(name)
	if err != nil {
		// TODO(aznashwan): implement some error switching helpers in the SDK
		// to avoid HTTP error checks such as the below:
		if resp.StatusCode != 200 {
			return false, err
		}

		return true, nil
	}

	return true, nil
}

// resourceArmResourceGroupDelete deletes the specified ARM resource group.
func resourceArmResourceGroupDelete(d *schema.ResourceData, meta interface{}) error {
	resGroupClient := meta.(*ArmClient).resourceGroupClient

	name := d.Id()

	_, err := resGroupClient.Delete(name)
	if err != nil {
		return err
	}

	return nil
}

// resourceGroupStateRefreshFunc returns a resource.StateRefreshFunc that is used to watch
// a resource group.
func resourceGroupStateRefreshFunc(client *ArmClient, id string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		res, err := client.resourceGroupClient.Get(id)
		if err != nil {
			return nil, "", fmt.Errorf("Error issuing read request in resourceGroupStateRefreshFunc to Azure ARM for resource group '%s': %s", id, err)
		}

		return res, *res.Properties.ProvisioningState, nil
	}
}
