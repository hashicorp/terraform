package rancher

import (
	"fmt"
	"log"
	"reflect"
	"strings"
	"time"

	compose "github.com/docker/libcompose/config"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
	rancherClient "github.com/rancher/go-rancher/client"
)

func resourceRancherStack() *schema.Resource {
	return &schema.Resource{
		Create: resourceRancherStackCreate,
		Read:   resourceRancherStackRead,
		Update: resourceRancherStackUpdate,
		Delete: resourceRancherStackDelete,
		Importer: &schema.ResourceImporter{
			State: resourceRancherStackImport,
		},

		Schema: map[string]*schema.Schema{
			"id": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"description": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"environment_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"docker_compose": {
				Type:             schema.TypeString,
				Optional:         true,
				DiffSuppressFunc: suppressComposeDiff,
			},
			"rancher_compose": {
				Type:             schema.TypeString,
				Optional:         true,
				DiffSuppressFunc: suppressComposeDiff,
			},
			"environment": {
				Type:     schema.TypeMap,
				Optional: true,
			},
			"catalog_id": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"scope": {
				Type:         schema.TypeString,
				Default:      "user",
				Optional:     true,
				ValidateFunc: validation.StringInSlice([]string{"user", "system"}, true),
			},
			"start_on_create": {
				Type:     schema.TypeBool,
				Optional: true,
				Computed: true,
			},
			"finish_upgrade": {
				Type:     schema.TypeBool,
				Optional: true,
			},
			"rendered_docker_compose": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"rendered_rancher_compose": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceRancherStackCreate(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[INFO] Creating Stack: %s", d.Id())
	client, err := meta.(*Config).EnvironmentClient(d.Get("environment_id").(string))
	if err != nil {
		return err
	}

	data, err := makeStackData(d, meta)
	if err != nil {
		return err
	}

	var newStack rancherClient.Environment
	if err := client.Create("environment", data, &newStack); err != nil {
		return err
	}

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"activating", "active", "removed", "removing"},
		Target:     []string{"active"},
		Refresh:    StackStateRefreshFunc(client, newStack.Id),
		Timeout:    10 * time.Minute,
		Delay:      1 * time.Second,
		MinTimeout: 3 * time.Second,
	}
	_, waitErr := stateConf.WaitForState()
	if waitErr != nil {
		return fmt.Errorf(
			"Error waiting for stack (%s) to be created: %s", newStack.Id, waitErr)
	}

	d.SetId(newStack.Id)
	log.Printf("[INFO] Stack ID: %s", d.Id())

	return resourceRancherStackRead(d, meta)
}

func resourceRancherStackRead(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[INFO] Refreshing Stack: %s", d.Id())
	client, err := meta.(*Config).EnvironmentClient(d.Get("environment_id").(string))
	if err != nil {
		return err
	}

	stack, err := client.Environment.ById(d.Id())
	if err != nil {
		return err
	}

	if stack == nil {
		log.Printf("[INFO] Stack %s not found", d.Id())
		d.SetId("")
		return nil
	}

	if removed(stack.State) {
		log.Printf("[INFO] Stack %s was removed on %v", d.Id(), stack.Removed)
		d.SetId("")
		return nil
	}

	config, err := client.Environment.ActionExportconfig(stack, &rancherClient.ComposeConfigInput{})
	if err != nil {
		return err
	}

	log.Printf("[INFO] Stack Name: %s", stack.Name)

	d.Set("description", stack.Description)
	d.Set("name", stack.Name)
	dockerCompose := strings.Replace(config.DockerComposeConfig, "\r", "", -1)
	rancherCompose := strings.Replace(config.RancherComposeConfig, "\r", "", -1)

	catalogID := d.Get("catalog_id")
	if catalogID == "" {
		d.Set("docker_compose", dockerCompose)
		d.Set("rancher_compose", rancherCompose)
	} else {
		d.Set("docker_compose", "")
		d.Set("rancher_compose", "")
	}
	d.Set("rendered_docker_compose", dockerCompose)
	d.Set("rendered_rancher_compose", rancherCompose)
	d.Set("environment_id", stack.AccountId)
	d.Set("environment", stack.Environment)

	if stack.ExternalId == "" {
		d.Set("scope", "user")
		d.Set("catalog_id", "")
	} else {
		trimmedID := strings.TrimPrefix(stack.ExternalId, "system-")
		if trimmedID == stack.ExternalId {
			d.Set("scope", "user")
		} else {
			d.Set("scope", "system")
		}
		d.Set("catalog_id", strings.TrimPrefix(trimmedID, "catalog://"))
	}

	d.Set("start_on_create", stack.StartOnCreate)
	d.Set("finish_upgrade", d.Get("finish_upgrade").(bool))

	return nil
}

func resourceRancherStackUpdate(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[INFO] Updating Stack: %s", d.Id())
	client, err := meta.(*Config).EnvironmentClient(d.Get("environment_id").(string))
	if err != nil {
		return err
	}
	d.Partial(true)

	data, err := makeStackData(d, meta)
	if err != nil {
		return err
	}

	stack, err := client.Environment.ById(d.Id())
	if err != nil {
		return err
	}

	var newStack rancherClient.Environment
	if err = client.Update("environment", &stack.Resource, data, &newStack); err != nil {
		return err
	}

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"active", "active-updating"},
		Target:     []string{"active"},
		Refresh:    StackStateRefreshFunc(client, newStack.Id),
		Timeout:    10 * time.Minute,
		Delay:      1 * time.Second,
		MinTimeout: 3 * time.Second,
	}
	s, waitErr := stateConf.WaitForState()
	stack = s.(*rancherClient.Environment)
	if waitErr != nil {
		return fmt.Errorf(
			"Error waiting for stack (%s) to be updated: %s", stack.Id, waitErr)
	}

	d.SetPartial("name")
	d.SetPartial("description")
	d.SetPartial("scope")

	if d.HasChange("docker_compose") ||
		d.HasChange("rancher_compose") ||
		d.HasChange("environment") ||
		d.HasChange("catalog_id") {

		envMap := make(map[string]interface{})
		for key, value := range *data["environment"].(*map[string]string) {
			envValue := value
			envMap[key] = &envValue
		}
		stack, err = client.Environment.ActionUpgrade(stack, &rancherClient.EnvironmentUpgrade{
			DockerCompose:  *data["dockerCompose"].(*string),
			RancherCompose: *data["rancherCompose"].(*string),
			Environment:    envMap,
			ExternalId:     *data["externalId"].(*string),
		})
		if err != nil {
			return err
		}

		stateConf := &resource.StateChangeConf{
			Pending:    []string{"active", "upgrading", "upgraded"},
			Target:     []string{"upgraded"},
			Refresh:    StackStateRefreshFunc(client, stack.Id),
			Timeout:    10 * time.Minute,
			Delay:      1 * time.Second,
			MinTimeout: 3 * time.Second,
		}
		s, waitErr := stateConf.WaitForState()
		if waitErr != nil {
			return fmt.Errorf(
				"Error waiting for stack (%s) to be upgraded: %s", stack.Id, waitErr)
		}
		stack = s.(*rancherClient.Environment)

		if d.Get("finish_upgrade").(bool) {
			stack, err = client.Environment.ActionFinishupgrade(stack)
			if err != nil {
				return err
			}

			stateConf = &resource.StateChangeConf{
				Pending:    []string{"active", "upgraded", "finishing-upgrade"},
				Target:     []string{"active"},
				Refresh:    StackStateRefreshFunc(client, stack.Id),
				Timeout:    10 * time.Minute,
				Delay:      1 * time.Second,
				MinTimeout: 3 * time.Second,
			}
			_, waitErr = stateConf.WaitForState()
			if waitErr != nil {
				return fmt.Errorf(
					"Error waiting for stack (%s) to be upgraded: %s", stack.Id, waitErr)
			}
		}

		d.SetPartial("rendered_docker_compose")
		d.SetPartial("rendered_rancher_compose")
		d.SetPartial("docker_compose")
		d.SetPartial("rancher_compose")
		d.SetPartial("environment")
		d.SetPartial("catalog_id")
	}

	d.Partial(false)

	return resourceRancherStackRead(d, meta)
}

func resourceRancherStackDelete(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[INFO] Deleting Stack: %s", d.Id())
	id := d.Id()
	client, err := meta.(*Config).EnvironmentClient(d.Get("environment_id").(string))
	if err != nil {
		return err
	}

	stack, err := client.Environment.ById(id)
	if err != nil {
		return err
	}

	if err := client.Environment.Delete(stack); err != nil {
		return fmt.Errorf("Error deleting Stack: %s", err)
	}

	log.Printf("[DEBUG] Waiting for stack (%s) to be removed", id)

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"active", "removed", "removing"},
		Target:     []string{"removed"},
		Refresh:    StackStateRefreshFunc(client, id),
		Timeout:    10 * time.Minute,
		Delay:      1 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	_, waitErr := stateConf.WaitForState()
	if waitErr != nil {
		return fmt.Errorf(
			"Error waiting for stack (%s) to be removed: %s", id, waitErr)
	}

	d.SetId("")
	return nil
}

func resourceRancherStackImport(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	envID, resourceID := splitID(d.Id())
	d.SetId(resourceID)
	if envID != "" {
		d.Set("environment_id", envID)
	} else {
		client, err := meta.(*Config).GlobalClient()
		if err != nil {
			return []*schema.ResourceData{}, err
		}
		stack, err := client.Environment.ById(d.Id())
		if err != nil {
			return []*schema.ResourceData{}, err
		}
		d.Set("environment_id", stack.AccountId)
	}
	return []*schema.ResourceData{d}, nil
}

// StackStateRefreshFunc returns a resource.StateRefreshFunc that is used to watch
// a Rancher Stack.
func StackStateRefreshFunc(client *rancherClient.RancherClient, stackID string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		stack, err := client.Environment.ById(stackID)

		if err != nil {
			return nil, "", err
		}

		return stack, stack.State, nil
	}
}

func environmentFromMap(m map[string]interface{}) map[string]string {
	result := make(map[string]string)
	for k, v := range m {
		result[k] = v.(string)
	}
	return result
}

func makeStackData(d *schema.ResourceData, meta interface{}) (data map[string]interface{}, err error) {
	name := d.Get("name").(string)
	description := d.Get("description").(string)

	var externalID string
	var dockerCompose string
	var rancherCompose string
	var environment map[string]string
	if c, ok := d.GetOk("catalog_id"); ok {
		if scope, ok := d.GetOk("scope"); ok && scope.(string) == "system" {
			externalID = "system-"
		}
		catalogID := c.(string)
		externalID += "catalog://" + catalogID

		catalogClient, err := meta.(*Config).CatalogClient()
		if err != nil {
			return data, err
		}
		template, err := catalogClient.Template.ById(catalogID)
		if err != nil {
			return data, fmt.Errorf("Failed to get catalog template: %s", err)
		}

		if template == nil {
			return data, fmt.Errorf("Unknown catalog template %s", catalogID)
		}

		dockerCompose = template.Files["docker-compose.yml"].(string)
		rancherCompose = template.Files["rancher-compose.yml"].(string)
	}

	if c, ok := d.GetOk("docker_compose"); ok {
		dockerCompose = c.(string)
	}
	if c, ok := d.GetOk("rancher_compose"); ok {
		rancherCompose = c.(string)
	}
	environment = environmentFromMap(d.Get("environment").(map[string]interface{}))

	startOnCreate := d.Get("start_on_create")

	data = map[string]interface{}{
		"name":           &name,
		"description":    &description,
		"dockerCompose":  &dockerCompose,
		"rancherCompose": &rancherCompose,
		"environment":    &environment,
		"externalId":     &externalID,
		"startOnCreate":  &startOnCreate,
	}

	return data, nil
}

func suppressComposeDiff(k, old, new string, d *schema.ResourceData) bool {
	cOld, err := compose.CreateConfig([]byte(old))
	if err != nil {
		// TODO: log?
		return false
	}

	cNew, err := compose.CreateConfig([]byte(new))
	if err != nil {
		// TODO: log?
		return false
	}

	return reflect.DeepEqual(cOld, cNew)
}
