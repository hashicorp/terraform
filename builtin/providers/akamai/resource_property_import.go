package akamai

import (
	"github.com/akamai/AkamaiOPEN-edgegrid-golang/papi-v1"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourcePropertyImport(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	resourceId := d.Id()
	propertyId := resourceId

	for _, searchKey := range []papi.SearchKey{papi.SearchByPropertyName, papi.SearchByHostname, papi.SearchByEdgeHostname} {
		results, _ := papi.Search(searchKey, resourceId)
		if len(results.Versions.Items) > 0 {
			propertyId = results.Versions.Items[0].PropertyID
			break
		}
	}

	property := papi.NewProperty(papi.NewProperties())
	property.PropertyID = propertyId
	e := property.GetProperty()
	if e != nil {
		return nil, e
	}

	d.Set("account_id", property.AccountID)
	d.Set("contract_id", property.ContractID)
	d.Set("group_id", property.GroupID)
	d.Set("product_id", property.ProductID)
	d.Set("rule_format", property.RuleFormat)
	// d.Set("clone_from", property.CloneFrom)
	d.Set("name", property.PropertyName)
	d.SetId(property.PropertyID)

	return []*schema.ResourceData{d}, nil
}