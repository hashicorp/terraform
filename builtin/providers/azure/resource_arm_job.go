package azure

import (
	"github.com/hashicorp/terraform/helper/schema"
)

// resourceArmJob returns the *schema.Resource
// associated to job resources on ARM.
func resourceArmJob() *schema.Resource {
	return &schema.Resource{
		Create: resourceArmJobCreate,
		Read:   resourceArmJobRead,
		Exists: resourceArmJobExists,
		Update: resourceArmJobUpdate,
		Delete: resourceArmJobDelete,

		Schema: map[string]*schema.Schema{
		// TODO
		},
	}
}

// resourceArmJobCreate goes ahead and creates the specified ARM job.
func resourceArmJobCreate(d *schema.ResourceData, meta interface{}) error {
	// TODO

	return nil
}

// resourceArmJobRead goes ahead and reads the state of the corresponding ARM job.
func resourceArmJobRead(d *schema.ResourceData, meta interface{}) error {
	// TODO

	return nil
}

// resourceArmJobUpdate goes ahead and updates the corresponding ARM job.
func resourceArmJobUpdate(d *schema.ResourceData, meta interface{}) error {
	// TODO

	return nil
}

// resourceArmJobExists goes ahead and checks for the existence of the correspoding ARM job.
func resourceArmJobExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	// TODO

	return false, nil
}

// resourceArmJobDelete deletes the specified ARM job.
func resourceArmJobDelete(d *schema.ResourceData, meta interface{}) error {
	// TODO

	return nil
}
