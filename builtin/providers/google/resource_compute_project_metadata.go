package google

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/schema"
	"google.golang.org/api/compute/v1"
	"google.golang.org/api/googleapi"
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

func resourceComputeProjectMetadataCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	createMD := func() error {
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

		return computeOperationWaitGlobal(config, op, "SetCommonMetadata")
	}

	err := MetadataRetryWrapper(createMD)
	if err != nil {
		return err
	}

	return resourceComputeProjectMetadataRead(d, meta)
}

func resourceComputeProjectMetadataRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	// Load project service
	log.Printf("[DEBUG] Loading project service: %s", config.Project)
	project, err := config.clientCompute.Projects.Get(config.Project).Do()
	if err != nil {
		if gerr, ok := err.(*googleapi.Error); ok && gerr.Code == 404 {
			log.Printf("[WARN] Removing Project Metadata because it's gone")
			// The resource doesn't exist anymore
			d.SetId("")

			return nil
		}

		return fmt.Errorf("Error loading project '%s': %s", config.Project, err)
	}

	md := project.CommonInstanceMetadata

	if err = d.Set("metadata", MetadataFormatSchema(d.Get("metadata").(map[string]interface{}), md)); err != nil {
		return fmt.Errorf("Error setting metadata: %s", err)
	}

	d.SetId("common_metadata")

	return nil
}

func resourceComputeProjectMetadataUpdate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	if d.HasChange("metadata") {
		o, n := d.GetChange("metadata")

		updateMD := func() error {
			// Load project service
			log.Printf("[DEBUG] Loading project service: %s", config.Project)
			project, err := config.clientCompute.Projects.Get(config.Project).Do()
			if err != nil {
				return fmt.Errorf("Error loading project '%s': %s", config.Project, err)
			}

			md := project.CommonInstanceMetadata

			MetadataUpdate(o.(map[string]interface{}), n.(map[string]interface{}), md)

			op, err := config.clientCompute.Projects.SetCommonInstanceMetadata(config.Project, md).Do()

			if err != nil {
				return fmt.Errorf("SetCommonInstanceMetadata failed: %s", err)
			}

			log.Printf("[DEBUG] SetCommonMetadata: %d (%s)", op.Id, op.SelfLink)

			// Optimistic locking requires the fingerprint received to match
			// the fingerprint we send the server, if there is a mismatch then we
			// are working on old data, and must retry
			return computeOperationWaitGlobal(config, op, "SetCommonMetadata")
		}

		err := MetadataRetryWrapper(updateMD)
		if err != nil {
			return err
		}

		return resourceComputeProjectMetadataRead(d, meta)
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

	err = computeOperationWaitGlobal(config, op, "SetCommonMetadata")
	if err != nil {
		return err
	}

	return resourceComputeProjectMetadataRead(d, meta)
}
