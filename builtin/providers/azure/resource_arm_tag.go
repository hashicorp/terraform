package azure

import (
	"github.com/hashicorp/terraform/helper/schema"
)

// resourceArmTag returns the *schema.Resource
// associated to tag resources on ARM.
func resourceArmTag() *schema.Resource {
	return &schema.Resource{
		Create: resourceArmTagCreate,
		Read:   resourceArmTagRead,
		Exists: resourceArmTagExists,
		Update: resourceArmTagUpdate,
		Delete: resourceArmTagDelete,

		Schema: map[string]*schema.Schema{
		// TODO
		},
	}
}

// resourceArmTagCreate goes ahead and creates the specified ARM tag.
func resourceArmTagCreate(d *schema.ResourceData, meta interface{}) error {
	// TODO

	return nil
}

// resourceArmTagRead goes ahead and reads the state of the corresponding ARM tag.
func resourceArmTagRead(d *schema.ResourceData, meta interface{}) error {
	// TODO

	return nil
}

// resourceArmTagUpdate goes ahead and updates the corresponding ARM tag.
func resourceArmTagUpdate(d *schema.ResourceData, meta interface{}) error {
	// TODO

	return nil
}

// resourceArmTagExists goes ahead and checks for the existence of the correspoding ARM tag.
func resourceArmTagExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	// TODO

	return false, nil
}

// resourceArmTagDelete deletes the specified ARM tag.
func resourceArmTagDelete(d *schema.ResourceData, meta interface{}) error {
	// TODO

	return nil
}
