package rancher

import (
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
	rancherClient "github.com/rancher/go-rancher/v2"
)

var (
	defaultProjectTemplates = map[string]string{
		"mesos":      "",
		"kubernetes": "",
		"windows":    "",
		"swarm":      "",
		"cattle":     "",
	}
)

func resourceRancherEnvironment() *schema.Resource {
	return &schema.Resource{
		Create: resourceRancherEnvironmentCreate,
		Read:   resourceRancherEnvironmentRead,
		Update: resourceRancherEnvironmentUpdate,
		Delete: resourceRancherEnvironmentDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
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
			"orchestration": &schema.Schema{
				Type:          schema.TypeString,
				Optional:      true,
				ValidateFunc:  validation.StringInSlice([]string{"cattle", "kubernetes", "mesos", "swarm", "windows"}, true),
				Computed:      true,
				ConflictsWith: []string{"project_template_id"},
			},
			"project_template_id": &schema.Schema{
				Type:          schema.TypeString,
				Optional:      true,
				Computed:      true,
				ConflictsWith: []string{"orchestration"},
			},
			"description": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
		},
	}
}

func resourceRancherEnvironmentCreate(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[INFO] Creating Environment: %s", d.Id())
	populateProjectTemplateIDs(meta.(*Config))

	client, err := meta.(*Config).GlobalClient()
	if err != nil {
		return err
	}

	name := d.Get("name").(string)
	description := d.Get("description").(string)
	orchestration := d.Get("orchestration").(string)
	projectTemplateID := d.Get("project_template_id").(string)

	projectTemplateID, err = getProjectTemplateID(orchestration, projectTemplateID)
	if err != nil {
		return err
	}

	data := map[string]interface{}{
		"name":              &name,
		"description":       &description,
		"projectTemplateId": &projectTemplateID,
	}

	var newEnv rancherClient.Project
	if err := client.Create("project", data, &newEnv); err != nil {
		return err
	}

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"active", "removed", "removing"},
		Target:     []string{"active"},
		Refresh:    EnvironmentStateRefreshFunc(client, newEnv.Id),
		Timeout:    10 * time.Minute,
		Delay:      1 * time.Second,
		MinTimeout: 3 * time.Second,
	}
	_, waitErr := stateConf.WaitForState()
	if waitErr != nil {
		return fmt.Errorf(
			"Error waiting for environment (%s) to be created: %s", newEnv.Id, waitErr)
	}

	d.SetId(newEnv.Id)
	log.Printf("[INFO] Environment ID: %s", d.Id())

	return resourceRancherEnvironmentRead(d, meta)
}

func resourceRancherEnvironmentRead(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[INFO] Refreshing Environment: %s", d.Id())
	client, err := meta.(*Config).GlobalClient()
	if err != nil {
		return err
	}

	env, err := client.Project.ById(d.Id())
	if err != nil {
		return err
	}

	if env == nil {
		log.Printf("[INFO] Environment %s not found", d.Id())
		d.SetId("")
		return nil
	}

	if removed(env.State) {
		log.Printf("[INFO] Environment %s was removed on %v", d.Id(), env.Removed)
		d.SetId("")
		return nil
	}

	log.Printf("[INFO] Environment Name: %s", env.Name)

	d.Set("description", env.Description)
	d.Set("name", env.Name)
	d.Set("orchestration", getActiveOrchestration(env))
	d.Set("project_template_id", env.ProjectTemplateId)

	return nil
}

func resourceRancherEnvironmentUpdate(d *schema.ResourceData, meta interface{}) error {
	populateProjectTemplateIDs(meta.(*Config))

	client, err := meta.(*Config).GlobalClient()
	if err != nil {
		return err
	}

	name := d.Get("name").(string)
	description := d.Get("description").(string)
	orchestration := d.Get("orchestration").(string)
	projectTemplateID := d.Get("project_template_id").(string)

	projectTemplateID, err = getProjectTemplateID(orchestration, projectTemplateID)
	if err != nil {
		return err
	}

	data := map[string]interface{}{
		"name":                &name,
		"description":         &description,
		"project_template_id": &projectTemplateID,
	}

	var newEnv rancherClient.Project
	env, err := client.Project.ById(d.Id())
	if err != nil {
		return err
	}

	if err := client.Update("project", &env.Resource, data, &newEnv); err != nil {
		return err
	}

	return resourceRancherEnvironmentRead(d, meta)
}

func resourceRancherEnvironmentDelete(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[INFO] Deleting Environment: %s", d.Id())
	id := d.Id()
	client, err := meta.(*Config).GlobalClient()
	if err != nil {
		return err
	}

	env, err := client.Project.ById(id)
	if err != nil {
		return err
	}

	if err := client.Project.Delete(env); err != nil {
		return fmt.Errorf("Error deleting Environment: %s", err)
	}

	log.Printf("[DEBUG] Waiting for environment (%s) to be removed", id)

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"active", "removed", "removing"},
		Target:     []string{"removed"},
		Refresh:    EnvironmentStateRefreshFunc(client, id),
		Timeout:    10 * time.Minute,
		Delay:      1 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	_, waitErr := stateConf.WaitForState()
	if waitErr != nil {
		return fmt.Errorf(
			"Error waiting for environment (%s) to be removed: %s", id, waitErr)
	}

	d.SetId("")
	return nil
}

func getProjectTemplateID(orchestration, templateID string) (string, error) {
	id := templateID
	if templateID == "" && orchestration == "" {
		return "", fmt.Errorf("Need either 'orchestration' or 'project_template_id'")
	}

	if templateID == "" && orchestration != "" {
		ok := false
		id, ok = defaultProjectTemplates[orchestration]
		if !ok {
			return "", fmt.Errorf("Invalid orchestration: %s", orchestration)
		}
	}

	return id, nil
}

// EnvironmentStateRefreshFunc returns a resource.StateRefreshFunc that is used to watch
// a Rancher Environment.
func EnvironmentStateRefreshFunc(client *rancherClient.RancherClient, environmentID string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		env, err := client.Project.ById(environmentID)

		if err != nil {
			return nil, "", err
		}

		return env, env.State, nil
	}
}
