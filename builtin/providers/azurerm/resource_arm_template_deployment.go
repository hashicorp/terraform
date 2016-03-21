package azurerm

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/Azure/azure-sdk-for-go/arm/resources/resources"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceArmTemplateDeployment() *schema.Resource {
	return &schema.Resource{
		Create: resourceArmTemplateDeploymentCreate,
		Read:   resourceArmTemplateDeploymentRead,
		Update: resourceArmTemplateDeploymentCreate,
		Delete: resourceArmTemplateDeploymentDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"resource_group_name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"template_body": &schema.Schema{
				Type:      schema.TypeString,
				Optional:  true,
				Computed:  true,
				StateFunc: normalizeJson,
			},

			"parameters": &schema.Schema{
				Type:     schema.TypeMap,
				Optional: true,
			},

			"deployment_mode": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
		},
	}
}

func resourceArmTemplateDeploymentCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ArmClient)
	deployClient := client.deploymentsClient

	name := d.Get("name").(string)
	resGroup := d.Get("resource_group_name").(string)
	deployment_mode := d.Get("deployment_mode").(string)

	log.Printf("[INFO] preparing arguments for Azure ARM Virtual Machine creation.")
	properties := resources.DeploymentProperties{
		Mode: resources.DeploymentMode(deployment_mode),
	}

	if v, ok := d.GetOk("parameters"); ok {
		params := v.(map[string]interface{})
		properties.Parameters = &params
	}

	if v, ok := d.GetOk("template_body"); ok {
		template, err := expandTemplateBody(v.(string))
		if err != nil {
			return err
		}

		properties.Template = &template
	}

	deployment := resources.Deployment{
		Properties: &properties,
	}
	resp, err := deployClient.CreateOrUpdate(resGroup, name, deployment)
	if err != nil {
		return nil
	}

	d.SetId(*resp.ID)

	log.Printf("[DEBUG] Waiting for Template Deploymnet (%s) to become available", name)
	stateConf := &resource.StateChangeConf{
		Pending: []string{"Creating", "Updating", "Accepted", "Running"},
		Target:  []string{"Succeeded"},
		Refresh: templateDeploymentStateRefreshFunc(client, resGroup, name),
		Timeout: 10 * time.Minute,
	}
	if _, err := stateConf.WaitForState(); err != nil {
		return fmt.Errorf("Error waiting for Template Deployment (%s) to become available: %s", name, err)
	}

	return resourceArmTemplateDeploymentRead(d, meta)
}

func resourceArmTemplateDeploymentRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ArmClient)
	deployClient := client.deploymentsClient

	id, err := parseAzureResourceID(d.Id())
	if err != nil {
		return err
	}
	resGroup := id.ResourceGroup
	name := id.Path["deployments"]
	if name == "" {
		name = id.Path["Deployments"]
	}

	resp, err := deployClient.Get(resGroup, name)
	if resp.StatusCode == http.StatusNotFound {
		d.SetId("")
		return nil
	}
	if err != nil {
		return fmt.Errorf("Error making Read request on Azure Template Deployment %s: %s", name, err)
	}

	return nil
}

func resourceArmTemplateDeploymentDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ArmClient)
	deployClient := client.deploymentsClient

	id, err := parseAzureResourceID(d.Id())
	if err != nil {
		return err
	}
	resGroup := id.ResourceGroup
	name := id.Path["deployments"]
	if name == "" {
		name = id.Path["Deployments"]
	}

	_, err = deployClient.Delete(resGroup, name)
	return nil
}

func expandTemplateBody(template string) (map[string]interface{}, error) {
	var templateBody map[string]interface{}
	err := json.Unmarshal([]byte(template), &templateBody)
	if err != nil {
		return nil, fmt.Errorf("dont be a dumb fuck")
	}
	return templateBody, nil
}

func normalizeJson(jsonString interface{}) string {
	if jsonString == nil || jsonString == "" {
		return ""
	}
	var j interface{}
	err := json.Unmarshal([]byte(jsonString.(string)), &j)
	if err != nil {
		return fmt.Sprintf("Error parsing JSON: %s", err)
	}
	b, _ := json.Marshal(j)
	return string(b[:])
}

func templateDeploymentStateRefreshFunc(client *ArmClient, resourceGroupName string, name string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		res, err := client.deploymentsClient.Get(resourceGroupName, name)
		if err != nil {
			return nil, "", fmt.Errorf("Error issuing read request in templateDeploymentStateRefreshFunc to Azure ARM for Template Deployment '%s' (RG: '%s'): %s", name, resourceGroupName, err)
		}

		return res, *res.Properties.ProvisioningState, nil
	}
}
