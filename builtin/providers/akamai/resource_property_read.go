package akamai

import (
	"log"

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
	log.Printf("[DEBUG] READING")

	propertyId := d.Id()

	if propertyId == "" {
		results, err := papi.Search(papi.SearchByPropertyName, d.Get("name").(string))
		if err != nil {
			return err
		}
		
		if len(results.Versions.Items) > 0 {
			propertyId = results.Versions.Items[0].PropertyID
		}
	}

	if propertyId != "" {
		property := papi.NewProperty(papi.NewProperties())
		property.PropertyID = propertyId
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
	}

	log.Println("[DEBUG] Done")

	return nil
}
