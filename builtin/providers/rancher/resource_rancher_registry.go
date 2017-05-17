package rancher

import (
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	rancherClient "github.com/rancher/go-rancher/client"
)

func resourceRancherRegistry() *schema.Resource {
	return &schema.Resource{
		Create: resourceRancherRegistryCreate,
		Read:   resourceRancherRegistryRead,
		Update: resourceRancherRegistryUpdate,
		Delete: resourceRancherRegistryDelete,
		Importer: &schema.ResourceImporter{
			State: resourceRancherRegistryImport,
		},

		Schema: map[string]*schema.Schema{
			"id": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"server_address": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"description": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"environment_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
		},
	}
}

func resourceRancherRegistryCreate(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[INFO] Creating Registry: %s", d.Id())
	client, err := meta.(*Config).EnvironmentClient(d.Get("environment_id").(string))
	if err != nil {
		return err
	}

	name := d.Get("name").(string)
	description := d.Get("description").(string)
	serverAddress := d.Get("server_address").(string)

	registry := rancherClient.Registry{
		Name:          name,
		Description:   description,
		ServerAddress: serverAddress,
	}
	newRegistry, err := client.Registry.Create(&registry)
	if err != nil {
		return err
	}

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"active", "removed", "removing"},
		Target:     []string{"active"},
		Refresh:    RegistryStateRefreshFunc(client, newRegistry.Id),
		Timeout:    10 * time.Minute,
		Delay:      1 * time.Second,
		MinTimeout: 3 * time.Second,
	}
	_, waitErr := stateConf.WaitForState()
	if waitErr != nil {
		return fmt.Errorf(
			"Error waiting for registry (%s) to be created: %s", newRegistry.Id, waitErr)
	}

	d.SetId(newRegistry.Id)
	log.Printf("[INFO] Registry ID: %s", d.Id())

	return resourceRancherRegistryRead(d, meta)
}

func resourceRancherRegistryRead(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[INFO] Refreshing Registry: %s", d.Id())
	client, err := meta.(*Config).EnvironmentClient(d.Get("environment_id").(string))
	if err != nil {
		return err
	}

	registry, err := client.Registry.ById(d.Id())
	if err != nil {
		return err
	}

	if registry == nil {
		log.Printf("[INFO] Registry %s not found", d.Id())
		d.SetId("")
		return nil
	}

	if removed(registry.State) {
		log.Printf("[INFO] Registry %s was removed on %v", d.Id(), registry.Removed)
		d.SetId("")
		return nil
	}

	log.Printf("[INFO] Registry Name: %s", registry.Name)

	d.Set("description", registry.Description)
	d.Set("name", registry.Name)
	d.Set("server_address", registry.ServerAddress)
	d.Set("environment_id", registry.AccountId)

	return nil
}

func resourceRancherRegistryUpdate(d *schema.ResourceData, meta interface{}) error {
	client, err := meta.(*Config).EnvironmentClient(d.Get("environment_id").(string))
	if err != nil {
		return err
	}

	registry, err := client.Registry.ById(d.Id())
	if err != nil {
		return err
	}

	name := d.Get("name").(string)
	description := d.Get("description").(string)

	registry.Name = name
	registry.Description = description
	client.Registry.Update(registry, &registry)

	return resourceRancherRegistryRead(d, meta)
}

func resourceRancherRegistryDelete(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[INFO] Deleting Registry: %s", d.Id())
	id := d.Id()
	client, err := meta.(*Config).EnvironmentClient(d.Get("environment_id").(string))
	if err != nil {
		return err
	}

	reg, err := client.Registry.ById(id)
	if err != nil {
		return err
	}

	// Step 1: Deactivate
	if _, e := client.Registry.ActionDeactivate(reg); e != nil {
		return fmt.Errorf("Error deactivating Registry: %s", err)
	}

	log.Printf("[DEBUG] Waiting for registry (%s) to be deactivated", id)

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"active", "inactive", "deactivating"},
		Target:     []string{"inactive"},
		Refresh:    RegistryStateRefreshFunc(client, id),
		Timeout:    10 * time.Minute,
		Delay:      1 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	_, waitErr := stateConf.WaitForState()
	if waitErr != nil {
		return fmt.Errorf(
			"Error waiting for registry (%s) to be deactivated: %s", id, waitErr)
	}

	// Update resource to reflect its state
	reg, err = client.Registry.ById(id)
	if err != nil {
		return fmt.Errorf("Failed to refresh state of deactivated registry (%s): %s", id, err)
	}

	// Step 2: Remove
	if _, err := client.Registry.ActionRemove(reg); err != nil {
		return fmt.Errorf("Error removing Registry: %s", err)
	}

	log.Printf("[DEBUG] Waiting for registry (%s) to be removed", id)

	stateConf = &resource.StateChangeConf{
		Pending:    []string{"inactive", "removed", "removing"},
		Target:     []string{"removed"},
		Refresh:    RegistryStateRefreshFunc(client, id),
		Timeout:    10 * time.Minute,
		Delay:      1 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	_, waitErr = stateConf.WaitForState()
	if waitErr != nil {
		return fmt.Errorf(
			"Error waiting for registry (%s) to be removed: %s", id, waitErr)
	}

	d.SetId("")
	return nil
}

func resourceRancherRegistryImport(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	envID, resourceID := splitID(d.Id())
	d.SetId(resourceID)
	if envID != "" {
		d.Set("environment_id", envID)
	} else {
		client, err := meta.(*Config).GlobalClient()
		if err != nil {
			return []*schema.ResourceData{}, err
		}
		registry, err := client.Registry.ById(d.Id())
		if err != nil {
			return []*schema.ResourceData{}, err
		}
		d.Set("environment_id", registry.AccountId)
	}
	return []*schema.ResourceData{d}, nil
}

// RegistryStateRefreshFunc returns a resource.StateRefreshFunc that is used to watch
// a Rancher Environment.
func RegistryStateRefreshFunc(client *rancherClient.RancherClient, registryID string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		env, err := client.Registry.ById(registryID)

		if err != nil {
			return nil, "", err
		}

		return env, env.State, nil
	}
}
