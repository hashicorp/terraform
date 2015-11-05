package azure

import (
	"github.com/hashicorp/terraform/helper/schema"
)

// resourceArmVmExtensionImage returns the *schema.Resource
// associated to vm extension image resources on ARM.
func resourceArmVmExtensionImage() *schema.Resource {
	return &schema.Resource{
		Create: resourceArmVmExtensionImageCreate,
		Read:   resourceArmVmExtensionImageRead,
		Exists: resourceArmVmExtensionImageExists,
		Update: resourceArmVmExtensionImageUpdate,
		Delete: resourceArmVmExtensionImageDelete,

		Schema: map[string]*schema.Schema{
		// TODO
		},
	}
}

// resourceArmVmExtensionImageCreate goes ahead and creates the specified ARM vm extension image.
func resourceArmVmExtensionImageCreate(d *schema.ResourceData, meta interface{}) error {
	// TODO

	return nil
}

// resourceArmVmExtensionImageRead goes ahead and reads the state of the corresponding ARM vm extension image.
func resourceArmVmExtensionImageRead(d *schema.ResourceData, meta interface{}) error {
	// TODO

	return nil
}

// resourceArmVmExtensionImageUpdate goes ahead and updates the corresponding ARM vm extension image.
func resourceArmVmExtensionImageUpdate(d *schema.ResourceData, meta interface{}) error {
	// TODO

	return nil
}

// resourceArmVmExtensionImageExists goes ahead and checks for the existence of the correspoding ARM vm extension image.
func resourceArmVmExtensionImageExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	// TODO

	return false, nil
}

// resourceArmVmExtensionImageDelete deletes the specified ARM vm extension image.
func resourceArmVmExtensionImageDelete(d *schema.ResourceData, meta interface{}) error {
	// TODO

	return nil
}
