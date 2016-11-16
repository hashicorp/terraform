package akamai

import (
	"errors"
	"log"
	"time"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceGTMWaitUntilDeployed(d *schema.ResourceData, meta interface{}) error {
	stateConf := &resource.StateChangeConf{
		Pending:    []string{"PENDING"},
		Target:     []string{"COMPLETE"},
		Refresh:    resourceGTMStateRefreshFunc(d, meta),
		Timeout:    30 * time.Minute,
		Delay:      1 * time.Minute,
		MinTimeout: 15 * time.Second,
	}
	_, err := stateConf.WaitForState()

	return err
}

func resourceGTMStateRefreshFunc(d *schema.ResourceData, meta interface{}) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		log.Printf("[INFO] waiting for %v COMPLETE propagation status", d.Get("domain"))
		status, err := meta.(*Clients).GTM.DomainStatus(d.Get("domain").(string))
		if err != nil {
			log.Printf("[ERROR] %#v", err)
			return nil, "", err
		}
		if status.PropagationStatus == "DENIED" {
			err = errors.New(status.Message)
			log.Printf("[ERROR] propagation status DENIED: %#v", err)
			return status, status.PropagationStatus, err
		}

		return status, status.PropagationStatus, nil
	}
}

func stringSetToStringSlice(stringSet *schema.Set) []string {
	ret := []string{}
	if stringSet == nil {
		return ret
	}

	for _, envVal := range stringSet.List() {
		ret = append(ret, envVal.(string))
	}

	return ret
}
