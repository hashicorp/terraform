package google

import (
	"fmt"
	"log"
	"time"

	"google.golang.org/api/googleapi"
	"google.golang.org/api/replicapool/v1beta2"

	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceReplicaPoolInstanceGroupManager() *schema.Resource {
	return &schema.Resource{
		Create: resourceReplicaPoolInstanceGroupManagerCreate,
		Read:   resourceReplicaPoolInstanceGroupManagerRead,
		Update: resourceReplicaPoolInstanceGroupManagerUpdate,
		Delete: resourceReplicaPoolInstanceGroupManagerDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"description": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"base_instance_name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"current_size": &schema.Schema{
				Type:     schema.TypeInt,
				Computed: true,
			},

			"fingerprint": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"group": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"instance_template": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"target_pools": &schema.Schema{
				Type:     schema.TypeSet,
				Required: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set: func(v interface{}) int {
					return hashcode.String(v.(string))
				},
			},

			"target_size": &schema.Schema{
				Type:     schema.TypeInt,
				Computed: true,
				Optional: true,
			},

			"zone": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"self_link": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func waitOpZone(config *Config, op *replicapool.Operation, zone string,
	resource string, action string) (*replicapool.Operation, error) {

	w := &ReplicaPoolOperationWaiter{
		Service: config.clientReplicaPool,
		Op:      op,
		Project: config.Project,
		Zone:    zone,
	}
	state := w.Conf()
	state.Timeout = 2 * time.Minute
	state.MinTimeout = 1 * time.Second
	opRaw, err := state.WaitForState()
	if err != nil {
		return nil, fmt.Errorf("Error waiting for %s to %s: %s", resource, action, err)
	}
	return opRaw.(*replicapool.Operation), nil
}

func resourceReplicaPoolInstanceGroupManagerCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	// Get group size, default to 1 if not given
	var target_size int64 = 1
	if v, ok := d.GetOk("target_size"); ok {
		target_size = int64(v.(int))
	}

	// Build the parameter
	manager := &replicapool.InstanceGroupManager{
		Name:             d.Get("name").(string),
		BaseInstanceName: d.Get("base_instance_name").(string),
		InstanceTemplate: d.Get("instance_template").(string),
	}

	// Set optional fields
	if v, ok := d.GetOk("description"); ok {
		manager.Description = v.(string)
	}

	if attr := d.Get("target_pools").(*schema.Set); attr.Len() > 0 {
		var s []string
		for _, v := range attr.List() {
			s = append(s, v.(string))
		}
		manager.TargetPools = s
	}

	log.Printf("[DEBUG] InstanceGroupManager insert request: %#v", manager)
	op, err := config.clientReplicaPool.InstanceGroupManagers.Insert(
		config.Project, d.Get("zone").(string), target_size, manager).Do()
	if err != nil {
		return fmt.Errorf("Error creating InstanceGroupManager: %s", err)
	}

	// It probably maybe worked, so store the ID now
	d.SetId(manager.Name)

	// Wait for the operation to complete
	op, err = waitOpZone(config, op, d.Get("zone").(string), "InstanceGroupManager", "create")
	if err != nil {
		return err
	}
	if op.Error != nil {
		// The resource didn't actually create
		d.SetId("")
		// Return the error
		return ReplicaPoolOperationError(*op.Error)
	}

	return resourceReplicaPoolInstanceGroupManagerRead(d, meta)
}

func resourceReplicaPoolInstanceGroupManagerRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	manager, err := config.clientReplicaPool.InstanceGroupManagers.Get(
		config.Project, d.Get("zone").(string), d.Id()).Do()
	if err != nil {
		if gerr, ok := err.(*googleapi.Error); ok && gerr.Code == 404 {
			// The resource doesn't exist anymore
			d.SetId("")

			return nil
		}

		return fmt.Errorf("Error reading instance group manager: %s", err)
	}

	// Set computed fields
	d.Set("current_size", manager.CurrentSize)
	d.Set("fingerprint", manager.Fingerprint)
	d.Set("group", manager.Group)
	d.Set("target_size", manager.TargetSize)
	d.Set("self_link", manager.SelfLink)

	return nil
}
func resourceReplicaPoolInstanceGroupManagerUpdate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	d.Partial(true)

	// If target_pools changes then update
	if d.HasChange("target_pools") {
		var targetPools []string
		if attr := d.Get("target_pools").(*schema.Set); attr.Len() > 0 {
			for _, v := range attr.List() {
				targetPools = append(targetPools, v.(string))
			}
		}

		// Build the parameter
		setTargetPools := &replicapool.InstanceGroupManagersSetTargetPoolsRequest{
			Fingerprint: d.Get("fingerprint").(string),
			TargetPools: targetPools,
		}

		op, err := config.clientReplicaPool.InstanceGroupManagers.SetTargetPools(
			config.Project, d.Get("zone").(string), d.Id(), setTargetPools).Do()
		if err != nil {
			return fmt.Errorf("Error updating InstanceGroupManager: %s", err)
		}

		// Wait for the operation to complete
		op, err = waitOpZone(config, op, d.Get("zone").(string), "InstanceGroupManager", "update TargetPools")
		if err != nil {
			return err
		}
		if op.Error != nil {
			return ReplicaPoolOperationError(*op.Error)
		}

		d.SetPartial("target_pools")
	}

	// If instance_template changes then update
	if d.HasChange("instance_template") {
		// Build the parameter
		setInstanceTemplate := &replicapool.InstanceGroupManagersSetInstanceTemplateRequest{
			InstanceTemplate: d.Get("instance_template").(string),
		}

		op, err := config.clientReplicaPool.InstanceGroupManagers.SetInstanceTemplate(
			config.Project, d.Get("zone").(string), d.Id(), setInstanceTemplate).Do()
		if err != nil {
			return fmt.Errorf("Error updating InstanceGroupManager: %s", err)
		}

		// Wait for the operation to complete
		op, err = waitOpZone(config, op, d.Get("zone").(string), "InstanceGroupManager", "update instance template")
		if err != nil {
			return err
		}
		if op.Error != nil {
			return ReplicaPoolOperationError(*op.Error)
		}

		d.SetPartial("instance_template")
	}

	// If size changes trigger a resize
	if d.HasChange("target_size") {
		if v, ok := d.GetOk("target_size"); ok {
			// Only do anything if the new size is set
			target_size := int64(v.(int))

			op, err := config.clientReplicaPool.InstanceGroupManagers.Resize(
				config.Project, d.Get("zone").(string), d.Id(), target_size).Do()
			if err != nil {
				return fmt.Errorf("Error updating InstanceGroupManager: %s", err)
			}

			// Wait for the operation to complete
			op, err = waitOpZone(config, op, d.Get("zone").(string), "InstanceGroupManager", "update target_size")
			if err != nil {
				return err
			}
			if op.Error != nil {
				return ReplicaPoolOperationError(*op.Error)
			}
		}

		d.SetPartial("target_size")
	}

	d.Partial(false)

	return resourceReplicaPoolInstanceGroupManagerRead(d, meta)
}

func resourceReplicaPoolInstanceGroupManagerDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	zone := d.Get("zone").(string)
	op, err := config.clientReplicaPool.InstanceGroupManagers.Delete(config.Project, zone, d.Id()).Do()
	if err != nil {
		return fmt.Errorf("Error deleting instance group manager: %s", err)
	}

	// Wait for the operation to complete
	w := &ReplicaPoolOperationWaiter{
		Service: config.clientReplicaPool,
		Op:      op,
		Project: config.Project,
		Zone:    d.Get("zone").(string),
	}
	state := w.Conf()
	state.Timeout = 2 * time.Minute
	state.MinTimeout = 1 * time.Second
	opRaw, err := state.WaitForState()
	if err != nil {
		return fmt.Errorf("Error waiting for InstanceGroupManager to delete: %s", err)
	}
	op = opRaw.(*replicapool.Operation)
	if op.Error != nil {
		// The resource didn't actually create
		d.SetId("")

		// Return the error
		return ReplicaPoolOperationError(*op.Error)
	}

	d.SetId("")
	return nil
}
