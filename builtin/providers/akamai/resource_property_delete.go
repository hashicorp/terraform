package akamai

import (
	"errors"
	"log"

	"github.com/akamai/AkamaiOPEN-edgegrid-golang/papi-v1"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourcePropertyDelete(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] DELETING")
	contractId, ok := d.GetOk("contract_id")
	if !ok {
		return errors.New("No contract ID")
	}
	groupId, ok := d.GetOk("group_id")
	if !ok {
		return errors.New("No group ID")
	}
	propertyId, ok := d.GetOk("property_id")
	if !ok {
		return errors.New("No property ID")
	}

	property := papi.NewProperty(papi.NewProperties())
	property.PropertyID = propertyId.(string)
	property.Contract = &papi.Contract{ContractID: contractId.(string)}
	property.Group = &papi.Group{GroupID: groupId.(string)}

	e := property.Delete()
	if e != nil {
		return e
	}

	d.SetId("")

	log.Println("[DEBUG] Done")

	return nil
}
