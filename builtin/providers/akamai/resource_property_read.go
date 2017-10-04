package akamai

import (
	"errors"
	"fmt"
	"log"

	"github.com/akamai/AkamaiOPEN-edgegrid-golang/papi-v1"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourcePropertyExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	return true, nil
}

func resourcePropertyRead(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] READING")
	contractId := d.Get("contract_id").(string)
	groupId := d.Get("group_id").(string)
	propertyId, ok := d.GetOk("property_id")
	if !ok {
		return errors.New("No property ID")
	}

	property := papi.NewProperty(papi.NewProperties())
	property.PropertyID = propertyId.(string)
	property.Contract = &papi.Contract{ContractID: contractId}
	property.Group = &papi.Group{GroupID: groupId}
	e := property.GetProperty()
	if e != nil {
		return e
	}

	d.Set("account_id", property.AccountID)
	d.Set("contract_id", property.ContractID)
	d.Set("group_id", property.GroupID)
	d.Set("product_id", property.ProductID)
	d.Set("property_id", property.PropertyID)
	d.Set("property_name", property.PropertyName)
	d.SetId(fmt.Sprintf("%s-%s-%s-%s", property.GroupID, property.ContractID, property.ProductID, property.PropertyID))

	log.Println("[DEBUG] Done")

	return nil
}
