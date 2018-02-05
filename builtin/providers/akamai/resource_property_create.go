package akamai

import (
	"errors"
	"fmt"
	"log"
	"time"

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
		Importer: &schema.ResourceImporter{
			State: resourcePropertyImport,
		},
		Schema: akamaiPropertySchema,
	}
}

func resourcePropertyCreate(d *schema.ResourceData, meta interface{}) error {
	d.Partial(true)

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

	cloneFrom, e := getCloneFrom(d)
	if e != nil {
		return e
	}

	var property *papi.Property
	if property = findProperty(d); property == nil {
		if group == nil {
			return errors.New("A group_id must be specified to create a new property")
		}

		if contract == nil {
			return errors.New("A contract_id must be specified to create a new property")
		}

		property, e = createProperty(contract, group, product, cloneFrom, d)
		if e != nil {
			return e
		}
	}

	err := ensureEditableVersion(property)
	if err != nil {
		return err
	}
	d.Set("version", property.LatestVersion)

	// The API now has data, so save the partial state
	d.SetId(property.PropertyID)
	d.SetPartial("name")
	d.SetPartial("rule_format")
	d.SetPartial("account_id")
	d.SetPartial("contract_id")
	d.SetPartial("group_id")
	d.SetPartial("product_id")
	d.SetPartial("clone_from")
	d.SetPartial("network")

	cpCode, e := createCpCode(property.Contract, property.Group, product, d)
	if e != nil {
		return e
	}
	d.SetPartial("cp_code")

	rules, e := property.GetRules()
	if e != nil {
		return e
	}

	origin, e := createOrigin(d)
	if e != nil {
		return e
	}

	updateStandardBehaviors(rules, cpCode, origin)
	fixupPerformanceBehaviors(rules)

	// get rules from the TF config
	unmarshalRules(d, rules)

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
	d.SetPartial("default")
	d.SetPartial("origin")
	d.SetPartial("rule")

	hostnameEdgeHostnameMap, err := createHostnames(property, product, d)
	if err != nil {
		return err
	}

	edgeHostnames, err := setEdgeHostnames(property, hostnameEdgeHostnameMap)
	if err != nil {
		return err
	}
	d.SetPartial("hostname")
	d.SetPartial("ipv6")
	_, edgeHostnameOk := d.GetOk("edge_hostname")
	if edgeHostnameOk {
		d.Set("edge_hostname", edgeHostnames)
	}

	if d.Get("activate").(bool) {
		activation, err := activateProperty(property, d)
		if err != nil {
			return err
		}
		d.SetPartial("contact")

		go activation.PollStatus(property)

	polling:
		for activation.Status != papi.StatusActive {
			select {
			case statusChanged := <-activation.StatusChange:
				log.Printf("[DEBUG] Property Status: %s\n", activation.Status)
				if statusChanged == false {
					break polling
				}
				continue polling
			case <-time.After(time.Minute * 90):
				log.Println("[DEBUG] Activation Timeout (90 minutes)")
				break polling
			}
		}
	}

	d.Partial(false)
	log.Println("[DEBUG] Done")
	return nil
}

func createProperty(contract *papi.Contract, group *papi.Group, product *papi.Product, cloneFrom *papi.ClonePropertyFrom, d *schema.ResourceData) (*papi.Property, error) {
	log.Println("[DEBUG] Creating property")

	property, err := group.NewProperty(contract)
	if err != nil {
		return nil, err
	}

	property.ProductID = product.ProductID
	property.PropertyName = d.Get("name").(string)
	if cloneFrom != nil {
		property.CloneFrom = cloneFrom
	}

	if ruleFormat, ok := d.GetOk("rule_format"); ok {
		property.RuleFormat = ruleFormat.(string)
	} else {
		ruleFormats := papi.NewRuleFormats()
		property.RuleFormat, err = ruleFormats.GetLatest()
	}

	err = property.Save()
	if err != nil {
		return nil, err
	}

	log.Printf("[DEBUG] Property created: %s\n", property.PropertyID)
	return property, nil
}
