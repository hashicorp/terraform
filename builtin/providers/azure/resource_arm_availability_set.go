package azure

import (
	"github.com/hashicorp/terraform/helper/schema"
)

// resourceArmAvailabilitySet returns the *schema.Resource
// associated to availability set resources on ARM.
func resourceArmAvailabilitySet() *schema.Resource {
	return &schema.Resource{
		Create: resourceArmAvailabilitySetCreate,
		Read:   resourceArmAvailabilitySetRead,
		Exists: resourceArmAvailabilitySetExists,
		Update: resourceArmAvailabilitySetUpdate,
		Delete: resourceArmAvailabilitySetDelete,

		Schema: map[string]*schema.Schema{
		// TODO
		},
	}
}

// resourceArmAvailabilitySetCreate goes ahead and creates the
// specified ARM availability set.
func resourceArmAvailabilitySetCreate(d *schema.ResourceData, meta interface{}) error {
	// TODO

	return nil
}

// resourceArmAvailabilitySetRead goes ahead and reads the state of the
// corresponding ARM availability set.
func resourceArmAvailabilitySetRead(d *schema.ResourceData, meta interface{}) error {
	// TODO

	return nil
}

// resourceArmAvailabilitySetUpdate goes ahead and updates the
// corresponding ARM availability set.
func resourceArmAvailabilitySetUpdate(d *schema.ResourceData, meta interface{}) error {
	// TODO

	return nil
}

// resourceArmAvailabilitySetExists goes ahead and checks for the existence
// of the correspoding ARM availability set.
func resourceArmAvailabilitySetExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	// TODO

	return false, nil
}

// resourceArmAvailabilitySetDelete deletes the specified ARM availability set.
func resourceArmAvailabilitySetDelete(d *schema.ResourceData, meta interface{}) error {
	// TODO

	return nil
}
