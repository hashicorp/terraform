package akamai

import (
	"errors"
	"log"

	"github.com/akamai/AkamaiOPEN-edgegrid-golang/papi-v1"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourcePropertyExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	return true, nil
}

func resourcePropertyRead(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] READING")
	contractId, ok := d.GetOk("contract_id")
	if !ok {
		return errors.New("No contract ID")
	}
	groupId, ok := d.GetOk("group_id")
	if !ok {
		return errors.New("No group ID")
	}
	propertyId := d.Id()

	property := papi.NewProperty(papi.NewProperties())
	property.PropertyID = propertyId
	property.Contract = &papi.Contract{ContractID: contractId.(string)}
	property.Group = &papi.Group{GroupID: groupId.(string)}
	e := property.GetProperty()
	if e != nil {
		return e
	}

	d.Set("account_id", property.AccountID)
	d.Set("contract_id", property.ContractID)
	d.Set("group_id", property.GroupID)
	d.Set("product_id", property.ProductID)
	d.Set("rule_format", property.RuleFormat)
	// d.Set("clone_from", property.CloneFrom)
	d.Set("name", property.PropertyName)
	d.SetId(property.PropertyID)

	log.Println("[DEBUG] Done")

	return nil
}
