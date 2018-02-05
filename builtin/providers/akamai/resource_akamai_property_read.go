package akamai

import (
	"github.com/akamai/AkamaiOPEN-edgegrid-golang/papi-v1"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourcePropertyExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	property := papi.NewProperty(papi.NewProperties())
	property.PropertyID = d.Id()
	e := property.GetProperty()
	if e != nil {
		return false, e
	}

	return true, nil
}

func resourcePropertyRead(d *schema.ResourceData, meta interface{}) error {
	return nil
}
