package akamai

import (
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/akamai/AkamaiOPEN-edgegrid-golang/papi-v1"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourcePropertyUpdate(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] UPDATING")
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

	property, e := getProperty(d, contract, group)
	if e != nil {
		return e
	}

	var cpCode *papi.CpCode
	if d.HasChange("cp_code") {
		cpCode, e = createCpCode(contract, group, product, d)
		if e != nil {
			return e
		}
		d.SetPartial("cp_code")
	} else {
		cpCode = papi.NewCpCode(papi.NewCpCodes(contract, group))
		cpCode.CpcodeID = d.Get("cp_code").(string)
		e := cpCode.GetCpCode()
		if e != nil {
			return e
		}
	}

	rules, e := property.GetRules()
	if e != nil {
		return e
	}

	origin, e := createOrigin(d)
	if e != nil {
		return e
	}

	addStandardBehaviors(rules, cpCode, origin)

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

	if d.HasChange("hostname") || d.HasChange("ipv6") {
		hostnameEdgeHostnameMap, err := createHostnames(contract, group, product, d)
		if err != nil {
			return err
		}

		edgeHostnames, err := setEdgeHostnames(property, hostnameEdgeHostnameMap)
		if err != nil {
			return err
		}
		d.SetPartial("hostname")
		d.SetPartial("ipv6")
		d.Set("edge_hostname", edgeHostnames)
	}

	// an existing activation on this property will be automatically deactivated upon
	// creation of this new activation
	activation, err := activateProperty(property, d)
	if err != nil {
		return err
	}
	d.SetPartial("contact")

	go activation.PollStatus(property)

	d.Partial(false)

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

	log.Println("[DEBUG] Done")
	return nil
}
