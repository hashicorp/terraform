package google

import (
	"fmt"

	"strconv"
	"time"

	"code.google.com/p/google-api-go-client/compute/v1"
	"code.google.com/p/google-api-go-client/googleapi"
	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceComputeTargetpool() *schema.Resource {
	return &schema.Resource{
		Create: resourceComputeTargetpoolCreate,
		Read:   resourceComputeTargetpoolRead,
		Update: resourceComputeTargetpoolUpdate,
		Delete: resourceComputeTargetpoolDelete,

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
			"session_affinity": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"region": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"instances": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set: func(v interface{}) int {
					return hashcode.String(v.(string))
				},
			},
			"health_checks": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set: func(v interface{}) int {
					return hashcode.String(v.(string))
				},
			},
			"backup_pool": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"failover_ratio": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"creation_timestamp": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"self_link": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceComputeTargetpoolCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	// Build the targetpool parameter
	var failoverratio float64
	if v := d.Get("failover_ratio").(string); v != "" {
		var err error
		failoverratio, err = strconv.ParseFloat(d.Get("failover_ratio").(string), 64)
		if err != nil {
			return fmt.Errorf("Error converting failover_ratio to a float: %s", err)
		}
	}

	var instances []string
	if v := d.Get("instances").(*schema.Set); v.Len() > 0 {
		instances = make([]string, v.Len())
		for i, v := range v.List() {
			instances[i] = v.(string)
		}
	}
	var healthchecks []string
	if v := d.Get("health_checks").(*schema.Set); v.Len() > 0 {
		healthchecks = make([]string, v.Len())
		for i, v := range v.List() {
			healthchecks[i] = v.(string)
		}
	}

	targetpool := &compute.TargetPool{
		Name:            d.Get("name").(string),
		Description:     d.Get("description").(string),
		FailoverRatio:   failoverratio,
		HealthChecks:    healthchecks,
		BackupPool:      d.Get("backup_pool").(string),
		SessionAffinity: d.Get("session_affinity").(string),
		Instances:       instances,
	}

	op, err := config.clientCompute.TargetPools.Insert(
		config.Project, config.Region, targetpool).Do()
	if err != nil {
		return fmt.Errorf("Error creating targetpool: %s", err)
	}

	d.SetId(targetpool.Name)

	// Wait for the operation to complete
	w := &OperationWaiter{
		Service: config.clientCompute,
		Op:      op,
		Project: config.Project,
		Region:  config.Region,
		Type:    OperationWaitRegion,
	}
	state := w.Conf()
	state.Timeout = 2 * time.Minute
	state.MinTimeout = 1 * time.Second
	opRaw, err := state.WaitForState()
	if err != nil {
		return fmt.Errorf("Error waiting for targetpool to create: %s", err)
	}
	op = opRaw.(*compute.Operation)
	if op.Error != nil {
		// The resource didn't actually create
		d.SetId("")

		// Return the error
		return OperationError(*op.Error)
	}

	return resourceComputeTargetpoolRead(d, meta)
}

func resourceComputeTargetpoolRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	targetpool, err := config.clientCompute.TargetPools.Get(
		config.Project, config.Region, d.Id()).Do()
	if err != nil {
		if gerr, ok := err.(*googleapi.Error); ok && gerr.Code == 404 {
			// The resource doesn't exist anymore
			d.SetId("")

			return nil
		}
		return fmt.Errorf("Error reading targetpool: %s", err)
	}

	d.Set("name", targetpool.Name)
	d.Set("description", targetpool.Description)
	d.Set("session_affinity", targetpool.SessionAffinity)
	d.Set("region", targetpool.Region)
	d.Set("instances", targetpool.Instances)
	d.Set("health_checks", targetpool.HealthChecks)
	d.Set("backup_pool", targetpool.BackupPool)
	d.Set("failover_ratio", targetpool.FailoverRatio)
	d.Set("creation_timestamp", targetpool.CreationTimestamp)
	d.Set("self_link", targetpool.SelfLink)
	return nil
}

func resourceComputeTargetpoolUpdate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	// Enable partial mode for the resource since it is possible
	d.Partial(true)

	// update instances if needed
	if d.HasChange("instances") {
		before, after := d.GetChange("instances")
		beforeSet := before.(*schema.Set)
		afterSet := after.(*schema.Set)
		remove := expandInstanceList(beforeSet.Difference(afterSet).List())
		add := expandInstanceList(afterSet.Difference(beforeSet).List())

		if len(add) > 0 {
			targetpooladdinstance := &compute.TargetPoolsAddInstanceRequest{
				Instances: add,
			}

			op, err := config.clientCompute.TargetPools.AddInstance(
				config.Project, config.Region, d.Get("name").(string), targetpooladdinstance).Do()

			if err != nil {
				return fmt.Errorf("Error adding instance to targetpool: %s", err)
			}

			// Wait for the operation to complete
			w := &OperationWaiter{
				Service: config.clientCompute,
				Op:      op,
				Project: config.Project,
				Region:  config.Region,
				Type:    OperationWaitRegion,
			}
			state := w.Conf()
			state.Timeout = 2 * time.Minute
			state.MinTimeout = 1 * time.Second
			opRaw, err := state.WaitForState()
			if err != nil {
				return fmt.Errorf("Error waiting for adding instance to targetpool: %s", err)
			}
			op = opRaw.(*compute.Operation)
			if op.Error != nil {
				// Return the error
				return OperationError(*op.Error)
			}
		}

		if len(remove) > 0 {
			targetpoolremoveinstance := &compute.TargetPoolsRemoveInstanceRequest{
				Instances: remove,
			}

			op, err := config.clientCompute.TargetPools.RemoveInstance(
				config.Project, config.Region, d.Get("name").(string), targetpoolremoveinstance).Do()

			if err != nil {
				return fmt.Errorf("Error removing instance from targetpool: %s", err)
			}

			// Wait for the operation to complete
			w := &OperationWaiter{
				Service: config.clientCompute,
				Op:      op,
				Project: config.Project,
				Region:  config.Region,
				Type:    OperationWaitRegion,
			}
			state := w.Conf()
			state.Timeout = 2 * time.Minute
			state.MinTimeout = 1 * time.Second
			opRaw, err := state.WaitForState()
			if err != nil {
				return fmt.Errorf("Error waiting for removing instance from targetpool: %s", err)
			}
			op = opRaw.(*compute.Operation)
			if op.Error != nil {
				// Return the error
				return OperationError(*op.Error)
			}
		}

		d.SetPartial("instances")
	}

	if d.HasChange("health_checks") {
		before, after := d.GetChange("health_checks")
		beforeSet := before.(*schema.Set)
		afterSet := after.(*schema.Set)
		remove := expandHealthCheckList(beforeSet.Difference(afterSet).List())
		add := expandHealthCheckList(afterSet.Difference(beforeSet).List())

		if len(add) > 0 {
			targetpooladdhealthcheck := &compute.TargetPoolsAddHealthCheckRequest{
				HealthChecks: add,
			}

			op, err := config.clientCompute.TargetPools.AddHealthCheck(
				config.Project, config.Region, d.Get("name").(string), targetpooladdhealthcheck).Do()

			if err != nil {
				return fmt.Errorf("Error adding health check to targetpool: %s", err)
			}

			// Wait for the operation to complete
			w := &OperationWaiter{
				Service: config.clientCompute,
				Op:      op,
				Project: config.Project,
				Region:  config.Region,
				Type:    OperationWaitRegion,
			}
			state := w.Conf()
			state.Timeout = 2 * time.Minute
			state.MinTimeout = 1 * time.Second
			opRaw, err := state.WaitForState()
			if err != nil {
				return fmt.Errorf("Error waiting for adding health check to targetpool: %s", err)
			}
			op = opRaw.(*compute.Operation)
			if op.Error != nil {
				// Return the error
				return OperationError(*op.Error)
			}
		}

		if len(remove) > 0 {
			targetpoolremovehealthcheck := &compute.TargetPoolsRemoveHealthCheckRequest{
				HealthChecks: remove,
			}

			op, err := config.clientCompute.TargetPools.RemoveHealthCheck(
				config.Project, config.Region, d.Get("name").(string), targetpoolremovehealthcheck).Do()

			if err != nil {
				return fmt.Errorf("Error removing health check from targetpool: %s", err)
			}

			// Wait for the operation to complete
			w := &OperationWaiter{
				Service: config.clientCompute,
				Op:      op,
				Project: config.Project,
				Region:  config.Region,
				Type:    OperationWaitRegion,
			}
			state := w.Conf()
			state.Timeout = 2 * time.Minute
			state.MinTimeout = 1 * time.Second
			opRaw, err := state.WaitForState()
			if err != nil {
				return fmt.Errorf("Error waiting for removing health check from targetpool: %s", err)
			}
			op = opRaw.(*compute.Operation)
			if op.Error != nil {
				// Return the error
				return OperationError(*op.Error)
			}
		}

		d.SetPartial("health_checks")
	}

	// Backup pool and failover ratio work together, if either changes set both
	if d.HasChange("backup_pool") || d.HasChange("failover_ratio") {
		var failoverratio float64
		if v := d.Get("failover_ratio").(string); v != "" {
			var err error
			failoverratio, err = strconv.ParseFloat(d.Get("failover_ratio").(string), 64)
			if err != nil {
				return fmt.Errorf("Error converting failover_ratio to a float: %s", err)
			}
		}

		targetpoolupdatebackuppool := &compute.TargetReference{
			Target: d.Get("backup_pool").(string),
		}
		request := config.clientCompute.TargetPools.SetBackup(config.Project, config.Region, d.Get("name").(string), targetpoolupdatebackuppool)
		request.FailoverRatio(failoverratio)

		op, err := request.Do()
		if err != nil {
			return fmt.Errorf("Error backup pool in targetpool: %s", err)
		}

		// Wait for the operation to complete
		w := &OperationWaiter{
			Service: config.clientCompute,
			Op:      op,
			Project: config.Project,
			Region:  config.Region,
			Type:    OperationWaitRegion,
		}
		state := w.Conf()
		state.Timeout = 2 * time.Minute
		state.MinTimeout = 1 * time.Second
		opRaw, err := state.WaitForState()
		if err != nil {
			return fmt.Errorf("Error waiting for backup pool update in targetpool: %s", err)
		}
		op = opRaw.(*compute.Operation)
		if op.Error != nil {
			// Return the error
			return OperationError(*op.Error)
		}

		d.SetPartial("backup_pool")
		d.SetPartial("failover_ratio")
	}

	// We made it, disable partial mode
	d.Partial(false)

	return resourceComputeTargetpoolRead(d, meta)
}

func resourceComputeTargetpoolDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	// Delete the targetpool
	op, err := config.clientCompute.TargetPools.Delete(
		config.Project, config.Region, d.Id()).Do()
	if err != nil {
		return fmt.Errorf("Error deleting targetpool: %s", err)
	}

	// Wait for the operation to complete
	w := &OperationWaiter{
		Service: config.clientCompute,
		Op:      op,
		Project: config.Project,
		Region:  config.Region,
		Type:    OperationWaitRegion,
	}
	state := w.Conf()
	state.Timeout = 2 * time.Minute
	state.MinTimeout = 1 * time.Second
	opRaw, err := state.WaitForState()
	if err != nil {
		return fmt.Errorf("Error waiting for targetpool to delete: %s", err)
	}
	op = opRaw.(*compute.Operation)
	if op.Error != nil {
		// Return the error
		return OperationError(*op.Error)
	}

	d.SetId("")
	return nil
}

func expandInstanceList(configured []interface{}) []*compute.InstanceReference {
	vs := make([]*compute.InstanceReference, 0, len(configured))
	for _, v := range configured {
		vs = append(vs, &compute.InstanceReference{Instance: v.(string)})
	}
	return vs
}

func expandHealthCheckList(configured []interface{}) []*compute.HealthCheckReference {
	vs := make([]*compute.HealthCheckReference, 0, len(configured))
	for _, v := range configured {
		vs = append(vs, &compute.HealthCheckReference{HealthCheck: v.(string)})
	}
	return vs
}
