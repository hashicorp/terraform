package akamai

import (
	"strings"

	"github.com/akamai/AkamaiOPEN-edgegrid-golang/papi-v1"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourcePropertyImport(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	resourceId := d.Id()
	propertyId := resourceId

	if !strings.HasPrefix(resourceId, "prp_") {
		for _, searchKey := range []papi.SearchKey{papi.SearchByPropertyName, papi.SearchByHostname, papi.SearchByEdgeHostname} {
			results, err := papi.Search(searchKey, resourceId)
			if err != nil {
				continue
			}

			if results != nil && len(results.Versions.Items) > 0 {
				propertyId = results.Versions.Items[0].PropertyID
				break
			}
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
	//d.Set("clone_from", property.CloneFrom.PropertyID)
	d.Set("name", property.PropertyName)
	d.Set("version", property.LatestVersion)
	d.SetId(property.PropertyID)

	return []*schema.ResourceData{d}, nil
}
