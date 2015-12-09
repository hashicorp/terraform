package azure

import (
	"github.com/hashicorp/terraform/helper/schema"
)

// resourceArmApplicationGateway returns the *schema.Resource
// associated to application gateway resources on ARM.
func resourceArmApplicationGateway() *schema.Resource {
	return &schema.Resource{
		Create: resourceArmApplicationGatewayCreate,
		Read:   resourceArmApplicationGatewayRead,
		Exists: resourceArmApplicationGatewayExists,
		Update: resourceArmApplicationGatewayUpdate,
		Delete: resourceArmApplicationGatewayDelete,

		Schema: map[string]*schema.Schema{
		// TODO
		},
	}
}

// resourceArmApplicationGatewayCreate goes ahead and creates the specified ARM application gateway.
func resourceArmApplicationGatewayCreate(d *schema.ResourceData, meta interface{}) error {
	// TODO

	return nil
}

// resourceArmApplicationGatewayRead goes ahead and reads the state of the corresponding ARM application gateway.
func resourceArmApplicationGatewayRead(d *schema.ResourceData, meta interface{}) error {
	// TODO

	return nil
}

// resourceArmApplicationGatewayUpdate goes ahead and updates the corresponding ARM application gateway.
func resourceArmApplicationGatewayUpdate(d *schema.ResourceData, meta interface{}) error {
	// TODO

	return nil
}

// resourceArmApplicationGatewayExists goes ahead and checks for the existence of the correspoding ARM application gateway.
func resourceArmApplicationGatewayExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	// TODO

	return false, nil
}

// resourceArmApplicationGatewayDelete deletes the specified ARM application gateway.
func resourceArmApplicationGatewayDelete(d *schema.ResourceData, meta interface{}) error {
	// TODO

	return nil
}
