package rancher

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
	rancherClient "github.com/rancher/go-rancher/client"
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
				Type:         schema.TypeString,
				Default:      "cattle",
				Optional:     true,
				ValidateFunc: validation.StringInSlice([]string{"cattle", "kubernetes", "mesos", "swarm"}, true),
			},
			"description": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"member": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"external_id_type": {
							Type:     schema.TypeString,
							Required: true,
						},
						"external_id": {
							Type:     schema.TypeString,
							Required: true,
						},
						"role": {
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},
		},
	}
}

func resourceRancherEnvironmentCreate(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[INFO] Creating Environment: %s", d.Id())
	client, err := meta.(*Config).GlobalClient()
	if err != nil {
		return err
	}

	name := d.Get("name").(string)
	description := d.Get("description").(string)
	orchestration := d.Get("orchestration").(string)

	data := map[string]interface{}{
		"name":        &name,
		"description": &description,
	}

	setOrchestrationFields(orchestration, data)

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

	// Add members
	if v, ok := d.GetOk("member"); ok {
		envClient, err := meta.(*Config).EnvironmentClient(d.Id())
		if err != nil {
			return err
		}
		members := v.([]interface{})
		_, err = envClient.Project.ActionSetmembers(&newEnv, &rancherClient.SetProjectMembersInput{
			Members: members,
		})
		if err != nil {
			return err
		}
	}

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

	envClient, err := meta.(*Config).EnvironmentClient(d.Id())
	if err != nil {
		return err
	}

	members, _ := envClient.ProjectMember.List(NewListOpts())

	d.Set("member", normalizeMembers(members.Data))
	return nil
}

func resourceRancherEnvironmentUpdate(d *schema.ResourceData, meta interface{}) error {
	client, err := meta.(*Config).GlobalClient()
	if err != nil {
		return err
	}

	name := d.Get("name").(string)
	description := d.Get("description").(string)
	orchestration := d.Get("orchestration").(string)

	data := map[string]interface{}{
		"name":        &name,
		"description": &description,
	}

	setOrchestrationFields(orchestration, data)

	var newEnv rancherClient.Project
	env, err := client.Project.ById(d.Id())
	if err != nil {
		return err
	}

	if err := client.Update("project", &env.Resource, data, &newEnv); err != nil {
		return err
	}

	// Update members
	envClient, err := meta.(*Config).EnvironmentClient(d.Id())
	if err != nil {
		return err
	}
	members := d.Get("member").(*schema.Set).List()
	_, err = envClient.Project.ActionSetmembers(&newEnv, &rancherClient.SetProjectMembersInput{
		Members: makeProjectMembers(members),
	})
	if err != nil {
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

func setOrchestrationFields(orchestration string, data map[string]interface{}) {
	orch := strings.ToLower(orchestration)

	data["swarm"] = false
	data["kubernetes"] = false
	data["mesos"] = false

	if orch == "k8s" {
		orch = "kubernetes"
	}

	data[orch] = true
}

func normalizeMembers(in []rancherClient.ProjectMember) (out []interface{}) {
	for _, m := range in {
		mm := map[string]string{
			"external_id_type": m.ExternalIdType,
			"external_id":      m.ExternalId,
			"role":             m.Role,
		}
		out = append(out, mm)
	}
	return
}

func makeProjectMembers(in []interface{}) (out []interface{}) {
	for _, m := range in {
		mMap := m.(map[string]interface{})
		mm := rancherClient.ProjectMember{
			ExternalIdType: mMap["external_id_type"].(string),
			ExternalId:     mMap["external_id"].(string),
			Role:           mMap["role"].(string),
		}
		out = append(out, mm)
	}
	return
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
