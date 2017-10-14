package akamai

import (
	"errors"
	"log"
	"strings"
	"time"

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
	// if there was no error, then activations were found, this can be an Activation or a Deactivation, so we check the ActivationType
	// in case it has already been deactivated
	if e == nil && activation.ActivationType == papi.ActivationTypeActivate {
		deactivation := papi.NewActivation(papi.NewActivations())
		deactivation.PropertyVersion = property.LatestVersion
		deactivation.ActivationType = papi.ActivationTypeDeactivate
		deactivation.Network = activation.Network
		deactivation.NotifyEmails = activation.NotifyEmails
		e = deactivation.Save(property, true)
		if e != nil {
			return e
		}
		log.Println("[DEBUG] DEACTIVATION SAVED - ID %s STATUS %s", deactivation.ActivationID, deactivation.Status)

		go deactivation.PollStatus(property)

	polling:
		for deactivation.Status != papi.StatusActive {
			select {
			case statusChanged := <-deactivation.StatusChange:
				log.Printf("[DEBUG] Property Status: %s\n", deactivation.Status)
				if statusChanged == false {
					break polling
				}
				continue polling
			case <-time.After(time.Minute * 90):
				log.Println("[DEBUG] Deactivation Timeout (90 minutes)")
				break polling
			}
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
