package azure

import (
	"github.com/hashicorp/terraform/helper/schema"
)

// resourceArmGateway returns the *schema.Resource
// associated to gateway resources on ARM.
func resourceArmGateway() *schema.Resource {
	return &schema.Resource{
		Create: resourceArmGatewayCreate,
		Read:   resourceArmGatewayRead,
		Exists: resourceArmGatewayExists,
		Update: resourceArmGatewayUpdate,
		Delete: resourceArmGatewayDelete,

		Schema: map[string]*schema.Schema{
		// TODO
		},
	}
}

// resourceArmGatewayCreate goes ahead and creates the specified ARM gateway.
func resourceArmGatewayCreate(d *schema.ResourceData, meta interface{}) error {
	// TODO

	return nil
}

// resourceArmGatewayRead goes ahead and reads the state of the corresponding ARM gateway.
func resourceArmGatewayRead(d *schema.ResourceData, meta interface{}) error {
	// TODO

	return nil
}

// resourceArmGatewayUpdate goes ahead and updates the corresponding ARM gateway.
func resourceArmGatewayUpdate(d *schema.ResourceData, meta interface{}) error {
	// TODO

	return nil
}

// resourceArmGatewayExists goes ahead and checks for the existence of the correspoding ARM gateway.
func resourceArmGatewayExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	// TODO

	return false, nil
}

// resourceArmGatewayDelete deletes the specified ARM gateway.
func resourceArmGatewayDelete(d *schema.ResourceData, meta interface{}) error {
	// TODO

	return nil
}
