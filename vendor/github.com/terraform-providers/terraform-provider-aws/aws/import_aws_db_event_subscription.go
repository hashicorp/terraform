package aws

import "github.com/hashicorp/terraform/helper/schema"

func resourceAwsDbEventSubscriptionImport(
	d *schema.ResourceData,
	meta interface{}) ([]*schema.ResourceData, error) {

	// The db event subscription Read function only needs the "name" of the event subscription
	// in order to populate the necessary values. This takes the "id" from the supplied StateFunc
	// and sets it as the "name" attribute, as described in the import documentation. This allows
	// the Read function to actually succeed and set the ID of the resource
	results := make([]*schema.ResourceData, 1, 1)
	d.Set("name", d.Id())
	results[0] = d
	return results, nil
}
