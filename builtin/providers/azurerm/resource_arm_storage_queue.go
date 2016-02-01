package azurerm

import (
	"fmt"
	"log"
	"regexp"

	"github.com/hashicorp/terraform/helper/schema"
)

func resourceArmStorageQueue() *schema.Resource {
	return &schema.Resource{
		Create: resourceArmStorageQueueCreate,
		Read:   resourceArmStorageQueueRead,
		Exists: resourceArmStorageQueueExists,
		Delete: resourceArmStorageQueueDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validateArmStorageQueueName,
			},
			"resource_group_name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"storage_account_name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
		},
	}
}

func validateArmStorageQueueName(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)

	if !regexp.MustCompile(`^[a-z0-9-]+$`).MatchString(value) {
		errors = append(errors, fmt.Errorf(
			"only lowercase alphanumeric characters and hyphens allowed in %q", k))
	}

	if regexp.MustCompile(`^-`).MatchString(value) {
		errors = append(errors, fmt.Errorf("%q cannot start with a hyphen", k))
	}

	if regexp.MustCompile(`-$`).MatchString(value) {
		errors = append(errors, fmt.Errorf("%q cannot end with a hyphen", k))
	}

	if len(value) > 63 {
		errors = append(errors, fmt.Errorf(
			"%q cannot be longer than 63 characters", k))
	}

	if len(value) < 3 {
		errors = append(errors, fmt.Errorf(
			"%q must be at least 3 characters", k))
	}

	return
}

func resourceArmStorageQueueCreate(d *schema.ResourceData, meta interface{}) error {
	armClient := meta.(*ArmClient)

	resourceGroupName := d.Get("resource_group_name").(string)
	storageAccountName := d.Get("storage_account_name").(string)

	queueClient, err := armClient.getQueueServiceClientForStorageAccount(resourceGroupName, storageAccountName)
	if err != nil {
		return err
	}

	name := d.Get("name").(string)

	log.Printf("[INFO] Creating queue %q in storage account %q", name, storageAccountName)
	err = queueClient.CreateQueue(name)
	if err != nil {
		return fmt.Errorf("Error creating storage queue on Azure: %s", err)
	}

	d.SetId(name)
	return resourceArmStorageQueueRead(d, meta)
}

func resourceArmStorageQueueRead(d *schema.ResourceData, meta interface{}) error {

	exists, err := resourceArmStorageQueueExists(d, meta)
	if err != nil {
		return err
	}

	if !exists {
		// Exists already removed this from state
		return nil
	}

	return nil
}

func resourceArmStorageQueueExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	armClient := meta.(*ArmClient)

	resourceGroupName := d.Get("resource_group_name").(string)
	storageAccountName := d.Get("storage_account_name").(string)

	queueClient, err := armClient.getQueueServiceClientForStorageAccount(resourceGroupName, storageAccountName)
	if err != nil {
		return false, err
	}

	name := d.Get("name").(string)

	log.Printf("[INFO] Checking for existence of storage queue %q.", name)
	exists, err := queueClient.QueueExists(name)
	if err != nil {
		return false, fmt.Errorf("error testing existence of storage queue %q: %s", name, err)
	}

	if !exists {
		log.Printf("[INFO] Storage queue %q no longer exists, removing from state...", name)
		d.SetId("")
	}

	return exists, nil
}

func resourceArmStorageQueueDelete(d *schema.ResourceData, meta interface{}) error {
	armClient := meta.(*ArmClient)

	resourceGroupName := d.Get("resource_group_name").(string)
	storageAccountName := d.Get("storage_account_name").(string)

	queueClient, err := armClient.getQueueServiceClientForStorageAccount(resourceGroupName, storageAccountName)
	if err != nil {
		return err
	}

	name := d.Get("name").(string)

	log.Printf("[INFO] Deleting storage queue %q", name)
	if err = queueClient.DeleteQueue(name); err != nil {
		return fmt.Errorf("Error deleting storage queue %q: %s", name, err)
	}

	d.SetId("")
	return nil
}
