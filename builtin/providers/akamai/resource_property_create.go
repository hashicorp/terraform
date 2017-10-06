package akamai

import (
	"errors"
	"fmt"
	"log"

	"github.com/akamai/AkamaiOPEN-edgegrid-golang/papi-v1"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceProperty() *schema.Resource {
	return &schema.Resource{
		Create: resourcePropertyCreate,
		Read:   resourcePropertyRead,
		Update: resourcePropertyUpdate,
		Delete: resourcePropertyDelete,
		Exists: resourcePropertyExists,
		// Importer: &schema.ResourceImporter{
		// State: importRecord,
		// },
		Schema: akamaiPropertySchema,
	}
}

func resourcePropertyCreate(d *schema.ResourceData, meta interface{}) error {
	group, e := getGroup(d)
	if e != nil {
		return e
	}

	contract, e := getContract(d)
	if e != nil {
		return e
	}

	product, e := getProduct(d, contract)
	if e != nil {
		return e
	}

	property, e := createProperty(contract, group, product, d)
	if e != nil {
		return e
	}

	// The API now has data, so save the state
	d.Set("property_id", property.PropertyID)
	d.SetId(fmt.Sprintf("%s-%s-%s-%s", group.GroupID, contract.ContractID, product.ProductID, property.PropertyID))

	rules, e := property.GetRules()
	if e != nil {
		return e
	}

	cpCode, e := createCpCode(contract, group, product, d)
	if e != nil {
		return e
	}

	rules.AddBehaviorOptions("/cpCode", papi.OptionValue{
		"value": papi.OptionValue{
			"id": cpCode.ID(),
		},
	})

	origin, e := createOrigin(d)
	if e != nil {
		return e
	}
	rules.AddBehaviorOptions("/origin", origin)

	e = rules.Save()
	if e != nil {
		if e == papi.ErrorMap[papi.ErrInvalidRules] && len(rules.Errors) > 0 {
			var msg string
			for _, v := range rules.Errors {
				msg = msg + fmt.Sprintf("\n Rule validation error: %s %s %s %s %s", v.Type, v.Title, v.Detail, v.Instance, v.BehaviorName)
			}
			return errors.New("Error - Invalid Property Rules" + msg)
		}
		return e
	}

	d.SetId(fmt.Sprintf("%s-%s-%s-%s", group.GroupID, contract.ContractID, product.ProductID, property.PropertyID))

	log.Println("[DEBUG] Done")

	return nil
}

func createProperty(contract *papi.Contract, group *papi.Group, product *papi.Product, d *schema.ResourceData) (*papi.Property, error) {
	log.Println("[DEBUG] Creating property")

	property, err := group.NewProperty(contract)
	if err != nil {
		return nil, err
	}

	property.ProductID = product.ProductID
	property.PropertyName = d.Get("name").(string)

	err = property.Save()
	if err != nil {
		return nil, err
	}

	log.Printf("[DEBUG] Property created: %s\n", property.PropertyID)
	return property, nil
}
