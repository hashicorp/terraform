package rancher

import (
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	rancherClient "github.com/rancher/go-rancher/client"
)

func resourceRancherRegistrationToken() *schema.Resource {
	return &schema.Resource{
		Create: resourceRancherRegistrationTokenCreate,
		Read:   resourceRancherRegistrationTokenRead,
		Delete: resourceRancherRegistrationTokenDelete,
		Importer: &schema.ResourceImporter{
			State: resourceRancherRegistrationTokenImport,
		},

		Schema: map[string]*schema.Schema{
			"id": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"description": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"environment_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"token": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"registration_url": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"command": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"image": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceRancherRegistrationTokenCreate(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[INFO] Creating RegistrationToken: %s", d.Id())
	client, err := meta.(*Config).EnvironmentClient(d.Get("environment_id").(string))
	if err != nil {
		return err
	}

	name := d.Get("name").(string)
	description := d.Get("description").(string)

	data := map[string]interface{}{
		"name":        &name,
		"description": &description,
	}

	var newRegT rancherClient.RegistrationToken
	if err := client.Create("registrationToken", data, &newRegT); err != nil {
		return err
	}

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"active", "removed", "removing"},
		Target:     []string{"active"},
		Refresh:    RegistrationTokenStateRefreshFunc(client, newRegT.Id),
		Timeout:    10 * time.Minute,
		Delay:      1 * time.Second,
		MinTimeout: 3 * time.Second,
	}
	_, waitErr := stateConf.WaitForState()
	if waitErr != nil {
		return fmt.Errorf(
			"Error waiting for registration token (%s) to be created: %s", newRegT.Id, waitErr)
	}

	d.SetId(newRegT.Id)
	log.Printf("[INFO] RegistrationToken ID: %s", d.Id())

	return resourceRancherRegistrationTokenRead(d, meta)
}

func resourceRancherRegistrationTokenRead(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[INFO] Refreshing RegistrationToken: %s", d.Id())
	client, err := meta.(*Config).EnvironmentClient(d.Get("environment_id").(string))
	if err != nil {
		return err
	}
	// client := meta.(*Config)

	regT, err := client.RegistrationToken.ById(d.Id())
	if err != nil {
		return err
	}

	if regT == nil {
		log.Printf("[INFO] RegistrationToken %s not found", d.Id())
		d.SetId("")
		return nil
	}

	if removed(regT.State) {
		log.Printf("[INFO] Registration Token %s was removed on %v", d.Id(), regT.Removed)
		d.SetId("")
		return nil
	}

	log.Printf("[INFO] RegistrationToken Name: %s", regT.Name)

	d.Set("description", regT.Description)
	d.Set("name", regT.Name)
	d.Set("token", regT.Token)
	d.Set("registration_url", regT.RegistrationUrl)
	d.Set("environment_id", regT.AccountId)
	d.Set("command", regT.Command)
	d.Set("image", regT.Image)

	return nil
}

func resourceRancherRegistrationTokenDelete(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[INFO] Deleting RegistrationToken: %s", d.Id())
	id := d.Id()
	client, err := meta.(*Config).EnvironmentClient(d.Get("environment_id").(string))
	if err != nil {
		return err
	}

	regT, err := client.RegistrationToken.ById(id)
	if err != nil {
		return err
	}

	// Step 1: Deactivate
	if _, e := client.RegistrationToken.ActionDeactivate(regT); e != nil {
		return fmt.Errorf("Error deactivating RegistrationToken: %s", err)
	}

	log.Printf("[DEBUG] Waiting for registration token (%s) to be deactivated", id)

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"active", "inactive", "deactivating"},
		Target:     []string{"inactive"},
		Refresh:    RegistrationTokenStateRefreshFunc(client, id),
		Timeout:    10 * time.Minute,
		Delay:      1 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	_, waitErr := stateConf.WaitForState()
	if waitErr != nil {
		return fmt.Errorf(
			"Error waiting for registration token (%s) to be deactivated: %s", id, waitErr)
	}

	// Update resource to reflect its state
	regT, err = client.RegistrationToken.ById(id)
	if err != nil {
		return fmt.Errorf("Failed to refresh state of deactivated registration token (%s): %s", id, err)
	}

	// Step 2: Remove
	if _, err := client.RegistrationToken.ActionRemove(regT); err != nil {
		return fmt.Errorf("Error removing RegistrationToken: %s", err)
	}

	log.Printf("[DEBUG] Waiting for registration token (%s) to be removed", id)

	stateConf = &resource.StateChangeConf{
		Pending:    []string{"inactive", "removed", "removing"},
		Target:     []string{"removed"},
		Refresh:    RegistrationTokenStateRefreshFunc(client, id),
		Timeout:    10 * time.Minute,
		Delay:      1 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	_, waitErr = stateConf.WaitForState()
	if waitErr != nil {
		return fmt.Errorf(
			"Error waiting for registration token (%s) to be removed: %s", id, waitErr)
	}

	d.SetId("")
	return nil
}

func resourceRancherRegistrationTokenImport(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	envID, resourceID := splitID(d.Id())
	d.SetId(resourceID)
	if envID != "" {
		d.Set("environment_id", envID)
	} else {
		client, err := meta.(*Config).GlobalClient()
		if err != nil {
			return []*schema.ResourceData{}, err
		}
		token, err := client.RegistrationToken.ById(d.Id())
		if err != nil {
			return []*schema.ResourceData{}, err
		}
		d.Set("environment_id", token.AccountId)
	}
	return []*schema.ResourceData{d}, nil
}

// RegistrationTokenStateRefreshFunc returns a resource.StateRefreshFunc that is used to watch
// a Rancher RegistrationToken.
func RegistrationTokenStateRefreshFunc(client *rancherClient.RancherClient, regTID string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		regT, err := client.RegistrationToken.ById(regTID)

		if err != nil {
			return nil, "", err
		}

		return regT, regT.State, nil
	}
}
