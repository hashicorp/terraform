package azure

import (
	"github.com/hashicorp/terraform/helper/schema"
)

// resourceArmNetworkInterface returns the *schema.Resource
// associated to network interface resources on ARM.
func resourceArmNetworkInterface() *schema.Resource {
	return &schema.Resource{
		Create: resourceArmNetworkInterfaceCreate,
		Read:   resourceArmNetworkInterfaceRead,
		Exists: resourceArmNetworkInterfaceExists,
		Update: resourceArmNetworkInterfaceUpdate,
		Delete: resourceArmNetworkInterfaceDelete,

		Schema: map[string]*schema.Schema{
		// TODO
		},
	}
}

// resourceArmNetworkInterfaceCreate goes ahead and creates the specified ARM network interface.
func resourceArmNetworkInterfaceCreate(d *schema.ResourceData, meta interface{}) error {
	// TODO

	return nil
}

// resourceArmNetworkInterfaceRead goes ahead and reads the state of the corresponding ARM network interface.
func resourceArmNetworkInterfaceRead(d *schema.ResourceData, meta interface{}) error {
	// TODO

	return nil
}

// resourceArmNetworkInterfaceUpdate goes ahead and updates the corresponding ARM network interface.
func resourceArmNetworkInterfaceUpdate(d *schema.ResourceData, meta interface{}) error {
	// TODO

	return nil
}

// resourceArmNetworkInterfaceExists goes ahead and checks for the existence of the correspoding ARM network interface.
func resourceArmNetworkInterfaceExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	// TODO

	return false, nil
}

// resourceArmNetworkInterfaceDelete deletes the specified ARM network interface.
func resourceArmNetworkInterfaceDelete(d *schema.ResourceData, meta interface{}) error {
	// TODO

	return nil
}
