package akamai

import (
	"errors"
	"log"
	"strings"

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
	network, ok := d.GetOk("network")
	if !ok {
		return errors.New("No network")
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

	activations, e := property.GetActivations()
	if e != nil {
		return e
	}

	activation, e := activations.GetLatestActivation(papi.NetworkValue(strings.ToUpper(network.(string))), papi.StatusActive)
	// an error here means there has not been any activation yet, so we can skip deactivating the property
	// if there was no error, then the activation was found and we should deactivate the property
	if e == nil {
		activation.ActivationType = papi.ActivationTypeDeactivate
		e = activation.Save(property, true)
		if e != nil {
			return e
		}
	}

	e = property.Delete()
	if e != nil {
		return e
	}

	d.SetId("")

	log.Println("[DEBUG] Done")

	return nil
}
