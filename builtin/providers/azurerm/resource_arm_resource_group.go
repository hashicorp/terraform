package azurerm

import (
	"fmt"
	"log"
	"regexp"
	"strings"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/jen20/riviera/azure"
)

func resourceArmResourceGroup() *schema.Resource {
	return &schema.Resource{
		Create: resourceArmResourceGroupCreate,
		Read:   resourceArmResourceGroupRead,
		Update: resourceArmResourceGroupUpdate,
		Exists: resourceArmResourceGroupExists,
		Delete: resourceArmResourceGroupDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validateArmResourceGroupName,
			},

			"location": locationSchema(),

			"tags": tagsSchema(),
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

func resourceArmResourceGroupUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ArmClient)
	rivieraClient := client.rivieraClient

	if !d.HasChange("tags") {
		return nil
	}

	name := d.Get("name").(string)
	newTags := d.Get("tags").(map[string]interface{})

	updateRequest := rivieraClient.NewRequestForURI(d.Id())
	updateRequest.Command = &azure.UpdateResourceGroup{
		Name: name,
		Tags: *expandTags(newTags),
	}

	updateResponse, err := updateRequest.Execute()
	if err != nil {
		return fmt.Errorf("Error updating resource group: %s", err)
	}
	if !updateResponse.IsSuccessful() {
		return fmt.Errorf("Error updating resource group: %s", updateResponse.Error)
	}

	return resourceArmResourceGroupRead(d, meta)
}

func resourceArmResourceGroupCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ArmClient)
	rivieraClient := client.rivieraClient

	createRequest := rivieraClient.NewRequest()
	createRequest.Command = &azure.CreateResourceGroup{
		Name:     d.Get("name").(string),
		Location: d.Get("location").(string),
		Tags:     *expandTags(d.Get("tags").(map[string]interface{})),
	}

	createResponse, err := createRequest.Execute()
	if err != nil {
		return fmt.Errorf("Error creating resource group: %s", err)
	}
	if !createResponse.IsSuccessful() {
		return fmt.Errorf("Error creating resource group: %s", createResponse.Error)
	}

	resp := createResponse.Parsed.(*azure.CreateResourceGroupResponse)
	d.SetId(*resp.ID)

	// TODO(jen20): Decide whether we need this or not and migrate to use @stack72's work if so
	// log.Printf("[DEBUG] Waiting for Resource Group (%s) to become available", name)
	// stateConf := &resource.StateChangeConf{
	// 	Pending: []string{"Accepted"},
	// 	Target:  []string{"Succeeded"},
	// 	Refresh: resourceGroupStateRefreshFunc(client, name),
	// 	Timeout: 10 * time.Minute,
	// }
	// if _, err := stateConf.WaitForState(); err != nil {
	// 	return fmt.Errorf("Error waiting for Resource Group (%s) to become available: %s", name, err)
	// }

	return resourceArmResourceGroupRead(d, meta)
}

func resourceArmResourceGroupRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ArmClient)
	rivieraClient := client.rivieraClient

	readRequest := rivieraClient.NewRequestForURI(d.Id())
	readRequest.Command = &azure.GetResourceGroup{}

	readResponse, err := readRequest.Execute()
	if err != nil {
		return fmt.Errorf("Error reading resource group: %s", err)
	}
	if !readResponse.IsSuccessful() {
		log.Printf("[INFO] Error reading resource group %q - removing from state", d.Id())
		d.SetId("")
		return fmt.Errorf("Error reading resource group: %s", readResponse.Error)
	}

	resp := readResponse.Parsed.(*azure.GetResourceGroupResponse)

	d.Set("name", resp.Name)
	d.Set("location", resp.Location)
	flattenAndSetTags(d, resp.Tags)

	return nil
}

func resourceArmResourceGroupExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	client := meta.(*ArmClient)
	rivieraClient := client.rivieraClient

	readRequest := rivieraClient.NewRequestForURI(d.Id())
	readRequest.Command = &azure.GetResourceGroup{}

	readResponse, err := readRequest.Execute()
	if err != nil {
		return false, fmt.Errorf("Error reading resource group: %s", err)
	}
	if readResponse.IsSuccessful() {
		return true, nil
	}

	return false, nil
}

func resourceArmResourceGroupDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ArmClient)
	rivieraClient := client.rivieraClient

	deleteRequest := rivieraClient.NewRequestForURI(d.Id())
	deleteRequest.Command = &azure.DeleteResourceGroup{}

	deleteResponse, err := deleteRequest.Execute()
	if err != nil {
		return fmt.Errorf("Error deleting resource group: %s", err)
	}
	if !deleteResponse.IsSuccessful() {
		return fmt.Errorf("Error deleting resource group: %s", deleteResponse.Error)
	}

	return nil

}
