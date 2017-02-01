package openstack

import (
	"fmt"
	"os"

	"github.com/gophercloud/gophercloud"
	"github.com/hashicorp/terraform/helper/schema"
)

// BuildRequest takes an opts struct and builds a request body for
// Gophercloud to execute
func BuildRequest(opts interface{}, parent string) (map[string]interface{}, error) {
	b, err := gophercloud.BuildRequestBody(opts, "")
	if err != nil {
		return nil, err
	}

	if b["value_specs"] != nil {
		for k, v := range b["value_specs"].(map[string]interface{}) {
			b[k] = v
		}
		delete(b, "value_specs")
	}

	return map[string]interface{}{parent: b}, nil
}

// CheckDeleted checks the error to see if it's a 404 (Not Found) and, if so,
// sets the resource ID to the empty string instead of throwing an error.
func CheckDeleted(d *schema.ResourceData, err error, msg string) error {
	if _, ok := err.(gophercloud.ErrDefault404); ok {
		d.SetId("")
		return nil
	}

	return fmt.Errorf("%s: %s", msg, err)
}

// GetRegion returns the region from either d.Get("region") or OS_REGION_NAME
func GetRegion(d *schema.ResourceData) string {
	if v, ok := d.GetOk("region"); ok {
		return v.(string)
	}

	if v := os.Getenv("OS_REGION_NAME"); v != "" {
		return v
	}

	return ""
}

// MapValueSpecs converts ResourceData into a map
func MapValueSpecs(d *schema.ResourceData) map[string]string {
	m := make(map[string]string)
	for key, val := range d.Get("value_specs").(map[string]interface{}) {
		m[key] = val.(string)
	}
	return m
}
