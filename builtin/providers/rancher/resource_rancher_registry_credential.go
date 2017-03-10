package rancher

import (
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	rancherClient "github.com/rancher/go-rancher/client"
)

func resourceRancherRegistryCredential() *schema.Resource {
	return &schema.Resource{
		Create: resourceRancherRegistryCredentialCreate,
		Read:   resourceRancherRegistryCredentialRead,
		Update: resourceRancherRegistryCredentialUpdate,
		Delete: resourceRancherRegistryCredentialDelete,
		Importer: &schema.ResourceImporter{
			State: resourceRancherRegistryCredentialImport,
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
			"registry_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"email": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"public_value": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"secret_value": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
		},
	}
}

func resourceRancherRegistryCredentialCreate(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[INFO] Creating RegistryCredential: %s", d.Id())
	client, err := meta.(*Config).RegistryClient(d.Get("registry_id").(string))
	if err != nil {
		return err
	}

	name := d.Get("name").(string)
	description := d.Get("description").(string)
	email := d.Get("email").(string)
	publicValue := d.Get("public_value").(string)
	secretValue := d.Get("secret_value").(string)
	registryID := d.Get("registry_id").(string)

	registryCred := rancherClient.RegistryCredential{
		Name:        name,
		Description: description,
		Email:       email,
		PublicValue: publicValue,
		SecretValue: secretValue,
		RegistryId:  registryID,
	}
	newRegistryCredential, err := client.RegistryCredential.Create(&registryCred)
	if err != nil {
		return err
	}

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"active", "removed", "removing"},
		Target:     []string{"active"},
		Refresh:    RegistryCredentialStateRefreshFunc(client, newRegistryCredential.Id),
		Timeout:    10 * time.Minute,
		Delay:      1 * time.Second,
		MinTimeout: 3 * time.Second,
	}
	_, waitErr := stateConf.WaitForState()
	if waitErr != nil {
		return fmt.Errorf(
			"Error waiting for registry credential (%s) to be created: %s", newRegistryCredential.Id, waitErr)
	}

	d.SetId(newRegistryCredential.Id)
	log.Printf("[INFO] RegistryCredential ID: %s", d.Id())

	return resourceRancherRegistryCredentialRead(d, meta)
}

func resourceRancherRegistryCredentialRead(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[INFO] Refreshing RegistryCredential: %s", d.Id())
	client, err := meta.(*Config).RegistryClient(d.Get("registry_id").(string))
	if err != nil {
		return err
	}

	registryCred, err := client.RegistryCredential.ById(d.Id())
	if err != nil {
		return err
	}

	if registryCred == nil {
		log.Printf("[INFO] RegistryCredential %s not found", d.Id())
		d.SetId("")
		return nil
	}

	if removed(registryCred.State) {
		log.Printf("[INFO] Registry Credential %s was removed on %v", d.Id(), registryCred.Removed)
		d.SetId("")
		return nil
	}

	log.Printf("[INFO] RegistryCredential Name: %s", registryCred.Name)

	d.Set("description", registryCred.Description)
	d.Set("name", registryCred.Name)
	d.Set("email", registryCred.Email)
	d.Set("public_value", registryCred.PublicValue)
	d.Set("registry_id", registryCred.RegistryId)

	return nil
}

func resourceRancherRegistryCredentialUpdate(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[INFO] Updating RegistryCredential: %s", d.Id())
	client, err := meta.(*Config).RegistryClient(d.Get("registry_id").(string))
	if err != nil {
		return err
	}

	registryCred, err := client.RegistryCredential.ById(d.Id())
	if err != nil {
		return err
	}

	name := d.Get("name").(string)
	description := d.Get("description").(string)
	email := d.Get("email").(string)
	publicValue := d.Get("public_value").(string)
	secretValue := d.Get("secret_value").(string)

	registryCred.Name = name
	registryCred.Description = description
	registryCred.Email = email
	registryCred.PublicValue = publicValue
	registryCred.SecretValue = secretValue
	client.RegistryCredential.Update(registryCred, &registryCred)

	return resourceRancherRegistryCredentialRead(d, meta)
}

func resourceRancherRegistryCredentialDelete(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[INFO] Deleting RegistryCredential: %s", d.Id())
	id := d.Id()
	client, err := meta.(*Config).RegistryClient(d.Get("registry_id").(string))
	if err != nil {
		return err
	}

	reg, err := client.RegistryCredential.ById(id)
	if err != nil {
		return err
	}

	// Step 1: Deactivate
	if _, e := client.RegistryCredential.ActionDeactivate(reg); e != nil {
		return fmt.Errorf("Error deactivating RegistryCredential: %s", err)
	}

	log.Printf("[DEBUG] Waiting for registry credential (%s) to be deactivated", id)

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"active", "inactive", "deactivating"},
		Target:     []string{"inactive"},
		Refresh:    RegistryCredentialStateRefreshFunc(client, id),
		Timeout:    10 * time.Minute,
		Delay:      1 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	_, waitErr := stateConf.WaitForState()
	if waitErr != nil {
		return fmt.Errorf(
			"Error waiting for registry credential (%s) to be deactivated: %s", id, waitErr)
	}

	// Update resource to reflect its state
	reg, err = client.RegistryCredential.ById(id)
	if err != nil {
		return fmt.Errorf("Failed to refresh state of deactivated registry credential (%s): %s", id, err)
	}

	// Step 2: Remove
	if _, err := client.RegistryCredential.ActionRemove(reg); err != nil {
		return fmt.Errorf("Error removing RegistryCredential: %s", err)
	}

	log.Printf("[DEBUG] Waiting for registry (%s) to be removed", id)

	stateConf = &resource.StateChangeConf{
		Pending:    []string{"inactive", "removed", "removing"},
		Target:     []string{"removed"},
		Refresh:    RegistryCredentialStateRefreshFunc(client, id),
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

func resourceRancherRegistryCredentialImport(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	regID, resourceID := splitID(d.Id())
	d.SetId(resourceID)
	if regID != "" {
		d.Set("registry_id", regID)
	} else {
		client, err := meta.(*Config).GlobalClient()
		if err != nil {
			return []*schema.ResourceData{}, err
		}
		cred, err := client.RegistryCredential.ById(d.Id())
		if err != nil {
			return []*schema.ResourceData{}, err
		}
		d.Set("registry_id", cred.RegistryId)
	}
	return []*schema.ResourceData{d}, nil
}

// RegistryCredentialStateRefreshFunc returns a resource.StateRefreshFunc that is used to watch
// a Rancher Environment.
func RegistryCredentialStateRefreshFunc(client *rancherClient.RancherClient, registryCredID string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		regC, err := client.RegistryCredential.ById(registryCredID)

		if err != nil {
			return nil, "", err
		}

		return regC, regC.State, nil
	}
}
