package azure

import (
	"github.com/hashicorp/terraform/helper/schema"
)

// resourceArmJobCollection returns the *schema.Resource
// associated to job collection resources on ARM.
func resourceArmJobCollection() *schema.Resource {
	return &schema.Resource{
		Create: resourceArmJobCollectionCreate,
		Read:   resourceArmJobCollectionRead,
		Exists: resourceArmJobCollectionExists,
		Update: resourceArmJobCollectionUpdate,
		Delete: resourceArmJobCollectionDelete,

		Schema: map[string]*schema.Schema{
		// TODO
		},
	}
}

// resourceArmJobCollectionCreate goes ahead and creates the specified ARM job collection.
func resourceArmJobCollectionCreate(d *schema.ResourceData, meta interface{}) error {
	// TODO

	return nil
}

// resourceArmJobCollectionRead goes ahead and reads the state of the corresponding ARM job collection.
func resourceArmJobCollectionRead(d *schema.ResourceData, meta interface{}) error {
	// TODO

	return nil
}

// resourceArmJobCollectionUpdate goes ahead and updates the corresponding ARM job collection.
func resourceArmJobCollectionUpdate(d *schema.ResourceData, meta interface{}) error {
	// TODO

	return nil
}

// resourceArmJobCollectionExists goes ahead and checks for the existence of the correspoding ARM job collection.
func resourceArmJobCollectionExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	// TODO

	return false, nil
}

// resourceArmJobCollectionDelete deletes the specified ARM job collection.
func resourceArmJobCollectionDelete(d *schema.ResourceData, meta interface{}) error {
	// TODO

	return nil
}
