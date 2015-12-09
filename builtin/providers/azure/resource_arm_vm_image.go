package azure

import (
	"github.com/hashicorp/terraform/helper/schema"
)

// resourceArmVmImage returns the *schema.Resource
// associated to vm image resources on ARM.
func resourceArmVmImage() *schema.Resource {
	return &schema.Resource{
		Create: resourceArmVmImageCreate,
		Read:   resourceArmVmImageRead,
		Exists: resourceArmVmImageExists,
		Update: resourceArmVmImageUpdate,
		Delete: resourceArmVmImageDelete,

		Schema: map[string]*schema.Schema{
		// TODO
		},
	}
}

// resourceArmVmImageCreate goes ahead and creates the specified ARM vm image.
func resourceArmVmImageCreate(d *schema.ResourceData, meta interface{}) error {
	// TODO

	return nil
}

// resourceArmVmImageRead goes ahead and reads the state of the corresponding ARM vm image.
func resourceArmVmImageRead(d *schema.ResourceData, meta interface{}) error {
	// TODO

	return nil
}

// resourceArmVmImageUpdate goes ahead and updates the corresponding ARM vm image.
func resourceArmVmImageUpdate(d *schema.ResourceData, meta interface{}) error {
	// TODO

	return nil
}

// resourceArmVmImageExists goes ahead and checks for the existence of the correspoding ARM vm image.
func resourceArmVmImageExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	// TODO

	return false, nil
}

// resourceArmVmImageDelete deletes the specified ARM vm image.
func resourceArmVmImageDelete(d *schema.ResourceData, meta interface{}) error {
	// TODO

	return nil
}
