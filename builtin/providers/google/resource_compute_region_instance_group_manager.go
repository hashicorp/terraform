package google

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/hashicorp/terraform/helper/schema"
	"google.golang.org/api/compute/v1"
	"google.golang.org/api/googleapi"
)

func resourceComputeRegionInstanceGroupManager() *schema.Resource {
	return &schema.Resource{
		Create: resourceComputeRegionInstanceGroupManagerCreate,
		Read:   resourceComputeRegionInstanceGroupManagerRead,
		Update: resourceComputeRegionInstanceGroupManagerUpdate,
		Delete: resourceComputeRegionInstanceGroupManagerDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"base_instance_name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"instance_template": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"region": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"description": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"fingerprint": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"instance_group": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"named_port": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},

						"port": &schema.Schema{
							Type:     schema.TypeInt,
							Required: true,
						},
					},
				},
			},

			"project": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Computed: true,
			},

			"self_link": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"update_strategy": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "RESTART",
			},

			"target_pools": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},

			"target_size": &schema.Schema{
				Type:     schema.TypeInt,
				Computed: true,
				Optional: true,
			},
		},
	}
}

func resourceComputeRegionInstanceGroupManagerCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	project, err := getProject(d, config)
	if err != nil {
		return err
	}

	region, err := getRegion(d, config)
	if err != nil {
		return err
	}

	var targetSize int64 = 1
	if v, ok := d.GetOk("target_size"); ok {
		targetSize = int64(v.(int))
	}

	// Build the parameter
	manager := &compute.InstanceGroupManager{
		Name:             d.Get("name").(string),
		BaseInstanceName: d.Get("base_instance_name").(string),
		InstanceTemplate: d.Get("instance_template").(string),
		TargetSize:       targetSize,
	}

	// Set optional fields
	if v, ok := d.GetOk("description"); ok {
		manager.Description = v.(string)
	}

	if v, ok := d.GetOk("named_port"); ok {
		manager.NamedPorts = getNamedPorts(v.([]interface{}))
	}

	if attr := d.Get("target_pools").(*schema.Set); attr.Len() > 0 {
		var s []string
		for _, v := range attr.List() {
			s = append(s, v.(string))
		}
		manager.TargetPools = s
	}

	updateStrategy := d.Get("update_strategy").(string)
	if !(updateStrategy == "NONE" || updateStrategy == "RESTART") {
		return fmt.Errorf(`Update strategy must be "NONE" or "RESTART"`)
	}

	log.Printf("[DEBUG] RegionInstanceGroupManager insert request: %#v", manager)

	op, err := config.clientCompute.RegionInstanceGroupManagers.Insert(
		project, region, manager).Do()
	if err != nil {
		return fmt.Errorf("Error creating RegionInstanceGroupManager: %s", err)
	}

	d.SetId(manager.Name)

	err = computeOperationWaitRegion(config, op, project, region, "Creating RegionInstanceGroupManager")
	if err != nil {
		return err
	}

	return resourceComputeRegionInstanceGroupManagerRead(d, meta)
}

func resourceComputeRegionInstanceGroupManagerRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	project, err := getProject(d, config)
	if err != nil {
		return err
	}

	region, err := getRegion(d, config)
	if err != nil {
		return err
	}

	service, err := config.clientCompute.RegionInstanceGroupManagers.Get(
		project, region, d.Id()).Do()
	if err != nil {
		if gerr, ok := err.(*googleapi.Error); ok && gerr.Code == 404 {
			log.Printf("[WARN] Removing Instance Group Manager %q because it's gone", d.Get("name").(string))

			d.SetId("")

			return nil
		}

		return fmt.Errorf("Error reading service: %s", err)
	}

	d.Set("base_instance_name", service.BaseInstanceName)
	d.Set("instance_template", service.InstanceTemplate)
	d.Set("name", service.Name)
	d.Set("description", service.Description)
	d.Set("project", project)
	d.Set("target_size", service.TargetSize)
	d.Set("target_pools", service.TargetPools)
	d.Set("named_port", flattenNamedPorts(service.NamedPorts))
	d.Set("fingerprint", service.Fingerprint)
	d.Set("instance_group", service.InstanceGroup)
	d.Set("target_size", service.TargetSize)
	d.Set("self_link", service.SelfLink)
	d.Set("update_strategy", "RESTART") //this field doesn't match the manager api, set to default value

	return nil
}

func resourceComputeRegionInstanceGroupManagerUpdate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	project, err := getProject(d, config)
	if err != nil {
		return err
	}

	region, err := getRegion(d, config)
	if err != nil {
		return err
	}

	d.Partial(true)

	if d.HasChange("target_pools") {
		var targetPools []string
		if attr := d.Get("target_pools").(*schema.Set); attr.Len() > 0 {
			for _, v := range attr.List() {
				targetPools = append(targetPools, v.(string))
			}
		}

		setTargetPools := &compute.RegionInstanceGroupManagersSetTargetPoolsRequest{
			Fingerprint: d.Get("fingerprint").(string),
			TargetPools: targetPools,
		}

		op, err := config.clientCompute.RegionInstanceGroupManagers.SetTargetPools(
			project, region, d.Id(), setTargetPools).Do()
		if err != nil {
			return fmt.Errorf("Error updating RegionInstanceGroupManager: %s", err)
		}

		err = computeOperationWaitRegion(config, op, project, region, "Updating RegionInstanceGroupManager")
		if err != nil {
			return err
		}

		d.SetPartial("target_pools")
	}

	if d.HasChange("instance_template") {
		setInstanceTemplate := &compute.RegionInstanceGroupManagersSetTemplateRequest{
			InstanceTemplate: d.Get("instance_template").(string),
		}

		op, err := config.clientCompute.RegionInstanceGroupManagers.SetInstanceTemplate(
			project, region, d.Id(), setInstanceTemplate).Do()
		if err != nil {
			return fmt.Errorf("Error updating RegionInstanceGroupManager: %s", err)
		}

		err = computeOperationWaitRegion(config, op, project, region, "Updating RegionInstanceGroupManager")
		if err != nil {
			return err
		}

		if d.Get("update_strategy").(string) == "RESTART" {
			managedInstances, err := config.clientCompute.RegionInstanceGroupManagers.ListManagedInstances(
				project, region, d.Id()).Do()

			managedInstanceCount := len(managedInstances.ManagedInstances)
			instances := make([]string, managedInstanceCount)
			for i, v := range managedInstances.ManagedInstances {
				instances[i] = v.Instance
			}

			recreateInstances := &compute.RegionInstanceGroupManagersRecreateRequest{
				Instances: instances,
			}

			op, err = config.clientCompute.RegionInstanceGroupManagers.RecreateInstances(
				project, region, d.Id(), recreateInstances).Do()

			if err != nil {
				return fmt.Errorf("Error restarting instance group managers instances: %s", err)
			}

			err = computeOperationWaitRegionTime(config, op, project, region,
				managedInstanceCount*4, "Restarting RegionInstanceGroupManagers instances")
			if err != nil {
				return err
			}
		}

		d.SetPartial("instance_template")
	}

	if d.HasChange("named_port") {
		namedPorts := getNamedPorts(d.Get("named_port").([]interface{}))
		setNamedPorts := &compute.RegionInstanceGroupsSetNamedPortsRequest{
			NamedPorts: namedPorts,
		}

		op, err := config.clientCompute.RegionInstanceGroups.SetNamedPorts(
			project, region, d.Id(), setNamedPorts).Do()
		if err != nil {
			return fmt.Errorf("Error updating RegionInstanceGroupManager: %s", err)
		}

		err = computeOperationWaitRegion(config, op, project, region, "Updating RegionInstanceGroupManager")
		if err != nil {
			return err
		}

		d.SetPartial("named_port")
	}

	if d.HasChange("target_size") {
		if v, ok := d.GetOk("target_size"); ok {
			targetSize := int64(v.(int))

			op, err := config.clientCompute.RegionInstanceGroupManagers.Resize(
				project, region, d.Id(), targetSize).Do()
			if err != nil {
				return fmt.Errorf("Error updating RegionInstanceGroupManager: %s", err)
			}

			err = computeOperationWaitRegion(config, op, project, region, "Updating RegionInstanceGroupManager")
			if err != nil {
				return err
			}
		}

		d.SetPartial("target_size")
	}

	d.Partial(false)

	return resourceComputeRegionInstanceGroupManagerRead(d, meta)
}

func resourceComputeRegionInstanceGroupManagerDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	project, err := getProject(d, config)
	if err != nil {
		return err
	}

	region, err := getRegion(d, config)
	if err != nil {
		return err
	}

	var op *compute.Operation
	for i := 0; i < 40; i++ {
		op, err = config.clientCompute.RegionInstanceGroupManagers.Delete(project, region, d.Id()).Do()
		if err != nil {
			time.Sleep(4000 * time.Millisecond)
			continue
		}
	}
	if gerr, ok := err.(*googleapi.Error); ok {
		if gerr.Code == 404 {
			log.Printf("[WARN] Removing Instance Group Manager %q because it's gone", d.Get("name").(string))

			d.SetId("")

			err = nil
		} else {
			return fmt.Errorf("Error deleting region instance group manager: %s", err)
		}
	}

	currentSize := int64(d.Get("target_size").(int))

	for err != nil && currentSize > 0 {
		if !strings.Contains(err.Error(), "timeout") {
			return err
		}

		instanceGroup, err := config.clientCompute.RegionInstanceGroups.Get(
			project, region, d.Id()).Do()

		if err != nil {
			return fmt.Errorf("Error getting instance group size: %s", err)
		}

		if instanceGroup.Size >= currentSize {
			return fmt.Errorf("Error, instance group isn't shrinking during delete")
		}

		log.Printf("[INFO] timeout occured, but instance group is shrinking (%d < %d)", instanceGroup.Size, currentSize)

		currentSize = instanceGroup.Size

		err = computeOperationWaitRegion(config, op, project, region, "Deleting RegionInstanceGroupManager")
	}

	d.SetId("")

	return nil
}
