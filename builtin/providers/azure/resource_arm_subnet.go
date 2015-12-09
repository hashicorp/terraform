package azure

import (
	"github.com/hashicorp/terraform/helper/schema"
)

// resourceArmSubnet returns the *schema.Resource
// associated to subnet resources on ARM.
func resourceArmSubnet() *schema.Resource {
	return &schema.Resource{
		Create: resourceArmSubnetCreate,
		Read:   resourceArmSubnetRead,
		Exists: resourceArmSubnetExists,
		Update: resourceArmSubnetUpdate,
		Delete: resourceArmSubnetDelete,

		Schema: map[string]*schema.Schema{
		// TODO
		},
	}
}

// resourceArmSubnetCreate goes ahead and creates the specified ARM subnet.
func resourceArmSubnetCreate(d *schema.ResourceData, meta interface{}) error {
	// TODO

	return nil
}

// resourceArmSubnetRead goes ahead and reads the state of the corresponding ARM subnet.
func resourceArmSubnetRead(d *schema.ResourceData, meta interface{}) error {
	// TODO

	return nil
}

// resourceArmSubnetUpdate goes ahead and updates the corresponding ARM subnet.
func resourceArmSubnetUpdate(d *schema.ResourceData, meta interface{}) error {
	// TODO

	return nil
}

// resourceArmSubnetExists goes ahead and checks for the existence of the correspoding ARM subnet.
func resourceArmSubnetExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	// TODO

	return false, nil
}

// resourceArmSubnetDelete deletes the specified ARM subnet.
func resourceArmSubnetDelete(d *schema.ResourceData, meta interface{}) error {
	// TODO

	return nil
}
