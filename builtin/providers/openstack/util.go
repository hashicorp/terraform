package openstack

import (
	"fmt"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/racker/perigee"
)

// CheckDeleted checks the error to see if it's a 404 (Not Found) and, if so,
// sets the resource ID to the empty string instead of throwing an error.
func CheckDeleted(d *schema.ResourceData, err error, resource string) error {
	errCode, ok := err.(*perigee.UnexpectedResponseCodeError)
	if !ok {
		return fmt.Errorf("Error retrieving OpenStack %s: %s", resource, err)
	}
	if errCode.Actual == 404 {
		d.SetId("")
		return nil
	}
	return fmt.Errorf("Error retrieving OpenStack %s: %s", resource, err)
}
