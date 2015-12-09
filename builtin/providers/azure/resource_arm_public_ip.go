package azure

import (
	"github.com/hashicorp/terraform/helper/schema"
)

// resourceArmPublicIp returns the *schema.Resource
// associated to public ip resources on ARM.
func resourceArmPublicIp() *schema.Resource {
	return &schema.Resource{
		Create: resourceArmPublicIpCreate,
		Read:   resourceArmPublicIpRead,
		Exists: resourceArmPublicIpExists,
		Update: resourceArmPublicIpUpdate,
		Delete: resourceArmPublicIpDelete,

		Schema: map[string]*schema.Schema{
		// TODO
		},
	}
}

// resourceArmPublicIpCreate goes ahead and creates the specified ARM public ip.
func resourceArmPublicIpCreate(d *schema.ResourceData, meta interface{}) error {
	// TODO

	return nil
}

// resourceArmPublicIpRead goes ahead and reads the state of the corresponding ARM public ip.
func resourceArmPublicIpRead(d *schema.ResourceData, meta interface{}) error {
	// TODO

	return nil
}

// resourceArmPublicIpUpdate goes ahead and updates the corresponding ARM public ip.
func resourceArmPublicIpUpdate(d *schema.ResourceData, meta interface{}) error {
	// TODO

	return nil
}

// resourceArmPublicIpExists goes ahead and checks for the existence of the correspoding ARM public ip.
func resourceArmPublicIpExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	// TODO

	return false, nil
}

// resourceArmPublicIpDelete deletes the specified ARM public ip.
func resourceArmPublicIpDelete(d *schema.ResourceData, meta interface{}) error {
	// TODO

	return nil
}
