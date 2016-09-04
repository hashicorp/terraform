package openstack

import (
	"fmt"

	"github.com/gophercloud/gophercloud"
	"github.com/hashicorp/terraform/helper/schema"
)

// CheckDeleted checks the error to see if it's a 404 (Not Found) and, if so,
// sets the resource ID to the empty string instead of throwing an error.
func CheckDeleted(d *schema.ResourceData, err error, msg string) error {
	errCode, ok := err.(*gophercloud.ErrUnexpectedResponseCode)
	if !ok {
		return fmt.Errorf("%s: %s", msg, err)
	}
	if errCode.Actual == 404 {
		d.SetId("")
		return nil
	}
	return fmt.Errorf("%s: %s", msg, err)
}
