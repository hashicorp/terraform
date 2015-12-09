package azure

import (
	"github.com/hashicorp/terraform/helper/schema"
)

// resourceArmGatewayConnection returns the *schema.Resource
// associated to gateway connection resources on ARM.
func resourceArmGatewayConnection() *schema.Resource {
	return &schema.Resource{
		Create: resourceArmGatewayConnectionCreate,
		Read:   resourceArmGatewayConnectionRead,
		Exists: resourceArmGatewayConnectionExists,
		Update: resourceArmGatewayConnectionUpdate,
		Delete: resourceArmGatewayConnectionDelete,

		Schema: map[string]*schema.Schema{
		// TODO
		},
	}
}

// resourceArmGatewayConnectionCreate goes ahead and creates the specified ARM gateway connection.
func resourceArmGatewayConnectionCreate(d *schema.ResourceData, meta interface{}) error {
	// TODO

	return nil
}

// resourceArmGatewayConnectionRead goes ahead and reads the state of the corresponding ARM gateway connection.
func resourceArmGatewayConnectionRead(d *schema.ResourceData, meta interface{}) error {
	// TODO

	return nil
}

// resourceArmGatewayConnectionUpdate goes ahead and updates the corresponding ARM gateway connection.
func resourceArmGatewayConnectionUpdate(d *schema.ResourceData, meta interface{}) error {
	// TODO

	return nil
}

// resourceArmGatewayConnectionExists goes ahead and checks for the existence of the correspoding ARM gateway connection.
func resourceArmGatewayConnectionExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	// TODO

	return false, nil
}

// resourceArmGatewayConnectionDelete deletes the specified ARM gateway connection.
func resourceArmGatewayConnectionDelete(d *schema.ResourceData, meta interface{}) error {
	// TODO

	return nil
}
