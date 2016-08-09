package azurerm

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/Azure/azure-sdk-for-go/arm/resources/resources"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceArmResourceGroup() *schema.Resource {
	return &schema.Resource{
		Create: resourceArmResourceGroupCreate,
		Read:   resourceArmResourceGroupRead,
		Update: resourceArmResourceGroupCreate,
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
			"location": &schema.Schema{
				Type:      schema.TypeString,
				Required:  true,
				ForceNew:  true,
				StateFunc: azureRMNormalizeLocation,
			},

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

func resourceArmResourceGroupCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ArmClient).resourceGroupClient

	name := d.Get("name").(string)
	location := d.Get("location").(string)
	tags := d.Get("tags").(map[string]interface{})

	group := resources.ResourceGroup{
		Name:     &name,
		Location: &location,
		Tags:     expandTags(tags),
	}

	_, err := client.CreateOrUpdate(name, group)
	if err != nil {
		return err
	}

	read, err := client.Get(name)
	if err != nil {
		return err
	}
	if read.ID == nil {
		return fmt.Errorf("Cannot read Resource Group %s ID", name)
	}

	d.SetId(*read.ID)

	// NOTE(pmcatominey): Comment left over from riviera based code
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
	client := meta.(*ArmClient).resourceGroupClient

	id, err := parseAzureResourceID(d.Id())
	if err != nil {
		return err
	}
	name := id.ResourceGroup

	resp, err := client.Get(name)
	if resp.StatusCode == http.StatusNotFound {
		d.SetId("")
		return nil
	}
	if err != nil {
		return fmt.Errorf("Error making Read request on Resource Group %s: %s", name, err)
	}

	d.Set("name", resp.Name)
	d.Set("location", resp.Location)
	flattenAndSetTags(d, resp.Tags)

	return nil
}

func resourceArmResourceGroupExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	client := meta.(*ArmClient).resourceGroupClient

	id, err := parseAzureResourceID(d.Id())
	if err != nil {
		return false, err
	}

	resp, err := client.Get(id.ResourceGroup)
	if resp.StatusCode == http.StatusNotFound {
		return false, nil
	}

	if err != nil {
		return false, fmt.Errorf("Error reading Resource Group: %s", err)
	}

	return true, nil
}

func resourceArmResourceGroupDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ArmClient).resourceGroupClient

	id, err := parseAzureResourceID(d.Id())
	if err != nil {
		return err
	}

	_, err = client.Delete(id.ResourceGroup, make(chan struct{}))

	return err

}
