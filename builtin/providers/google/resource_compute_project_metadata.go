package google

import (
	"fmt"
	"log"
	"time"

	//	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/schema"
	"google.golang.org/api/compute/v1"
	//	"google.golang.org/api/googleapi"
)

func resourceComputeProjectMetadata() *schema.Resource {
	return &schema.Resource{
		Create: resourceComputeProjectMetadataCreate,
		Read:   resourceComputeProjectMetadataRead,
		Update: resourceComputeProjectMetadataUpdate,
		Delete: resourceComputeProjectMetadataDelete,

		SchemaVersion: 0,

		Schema: map[string]*schema.Schema{
			"metadata": &schema.Schema{
				Elem:     schema.TypeString,
				Type:     schema.TypeMap,
				Required: true,
			},
		},
	}
}

const FINGERPRINT_RETRIES = 10
const FINGERPRINT_FAIL = "Invalid fingerprint."

func resourceOperationWaitGlobal(config *Config, op *compute.Operation, activity string) error {
	w := &OperationWaiter{
		Service: config.clientCompute,
		Op:      op,
		Project: config.Project,
		Type:    OperationWaitGlobal,
	}

	state := w.Conf()
	state.Timeout = 2 * time.Minute
	state.MinTimeout = 1 * time.Second
	opRaw, err := state.WaitForState()
	if err != nil {
		return fmt.Errorf("Error waiting for %s: %s", activity, err)
	}

	op = opRaw.(*compute.Operation)
	if op.Error != nil {
		return OperationError(*op.Error)
	}

	return nil
}

func resourceComputeProjectMetadataCreate(d *schema.ResourceData, meta interface{}) error {
	attempt := 0

	config := meta.(*Config)

	for attempt < FINGERPRINT_RETRIES {
		// Load project service
		log.Printf("[DEBUG] Loading project service: %s", config.Project)
		project, err := config.clientCompute.Projects.Get(config.Project).Do()
		if err != nil {
			return fmt.Errorf("Error loading project '%s': %s", config.Project, err)
		}

		md := project.CommonInstanceMetadata

		newMDMap := d.Get("metadata").(map[string]interface{})
		// Ensure that we aren't overwriting entries that already exist
		for _, kv := range md.Items {
			if _, ok := newMDMap[kv.Key]; ok {
				return fmt.Errorf("Error, key '%s' already exists in project '%s'", kv.Key, config.Project)
			}
		}

		// Append new metadata to existing metadata
		for key, val := range newMDMap {
			v := val.(string)
			md.Items = append(md.Items, &compute.MetadataItems{
				Key:   key,
				Value: &v,
			})
		}

		op, err := config.clientCompute.Projects.SetCommonInstanceMetadata(config.Project, md).Do()

		if err != nil {
			return fmt.Errorf("SetCommonInstanceMetadata failed: %s", err)
		}

		log.Printf("[DEBUG] SetCommonMetadata: %d (%s)", op.Id, op.SelfLink)

		// Optimistic locking requires the fingerprint recieved to match
		// the fingerprint we send the server, if there is a mismatch then we
		// are working on old data, and must retry
		err = resourceOperationWaitGlobal(config, op, "SetCommonMetadata")
		if err == nil {
			return resourceComputeProjectMetadataRead(d, meta)
		} else if err.Error() == FINGERPRINT_FAIL {
			attempt++
		} else {
			return err
		}
	}

	return fmt.Errorf("Error, unable to set metadata resource after %d attempts", attempt)
}

func resourceComputeProjectMetadataRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	// Load project service
	log.Printf("[DEBUG] Loading project service: %s", config.Project)
	project, err := config.clientCompute.Projects.Get(config.Project).Do()
	if err != nil {
		return fmt.Errorf("Error loading project '%s': %s", config.Project, err)
	}

	md := project.CommonInstanceMetadata

	newMD := make(map[string]interface{})

	for _, kv := range md.Items {
		newMD[kv.Key] = kv.Value
	}

	if err = d.Set("metadata", newMD); err != nil {
		return fmt.Errorf("Error setting metadata: %s", err)
	}

	d.SetId("common_metadata")

	return nil
}

func resourceComputeProjectMetadataUpdate(d *schema.ResourceData, meta interface{}) error {
	attempt := 0

	config := meta.(*Config)

	if d.HasChange("metadata") {
		o, n := d.GetChange("metadata")
		oMDMap, nMDMap := o.(map[string]interface{}), n.(map[string]interface{})

		for attempt < FINGERPRINT_RETRIES {
			// Load project service
			log.Printf("[DEBUG] Loading project service: %s", config.Project)
			project, err := config.clientCompute.Projects.Get(config.Project).Do()
			if err != nil {
				return fmt.Errorf("Error loading project '%s': %s", config.Project, err)
			}

			md := project.CommonInstanceMetadata

			curMDMap := make(map[string]string)
			// Load metadata on server into map
			for _, kv := range md.Items {
				// If the server state has a key that we had in our old
				// state, but not in our new state, we should delete it
				_, okOld := oMDMap[kv.Key]
				_, okNew := nMDMap[kv.Key]
				if okOld && !okNew {
					continue
				} else {
					if kv.Value != nil {
						curMDMap[kv.Key] = *kv.Value
					}
				}
			}

			// Insert new metadata into existing metadata (overwriting when needed)
			for key, val := range nMDMap {
				curMDMap[key] = val.(string)
			}

			// Reformat old metadata into a list
			md.Items = nil
			for key, val := range curMDMap {
				md.Items = append(md.Items, &compute.MetadataItems{
					Key:   key,
					Value: &val,
				})
			}

			op, err := config.clientCompute.Projects.SetCommonInstanceMetadata(config.Project, md).Do()

			if err != nil {
				return fmt.Errorf("SetCommonInstanceMetadata failed: %s", err)
			}

			log.Printf("[DEBUG] SetCommonMetadata: %d (%s)", op.Id, op.SelfLink)

			// Optimistic locking requires the fingerprint recieved to match
			// the fingerprint we send the server, if there is a mismatch then we
			// are working on old data, and must retry
			err = resourceOperationWaitGlobal(config, op, "SetCommonMetadata")
			if err == nil {
				return resourceComputeProjectMetadataRead(d, meta)
			} else if err.Error() == FINGERPRINT_FAIL {
				attempt++
			} else {
				return err
			}
		}

		return fmt.Errorf("Error, unable to set metadata resource after %d attempts", attempt)
	}

	return nil
}

func resourceComputeProjectMetadataDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	// Load project service
	log.Printf("[DEBUG] Loading project service: %s", config.Project)
	project, err := config.clientCompute.Projects.Get(config.Project).Do()
	if err != nil {
		return fmt.Errorf("Error loading project '%s': %s", config.Project, err)
	}

	md := project.CommonInstanceMetadata

	// Remove all items
	md.Items = nil

	op, err := config.clientCompute.Projects.SetCommonInstanceMetadata(config.Project, md).Do()

	log.Printf("[DEBUG] SetCommonMetadata: %d (%s)", op.Id, op.SelfLink)

	err = resourceOperationWaitGlobal(config, op, "SetCommonMetadata")
	if err != nil {
		return err
	}

	return resourceComputeProjectMetadataRead(d, meta)
}
