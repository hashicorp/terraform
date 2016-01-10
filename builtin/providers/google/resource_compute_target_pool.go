package google

import (
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/terraform/helper/schema"
	"google.golang.org/api/compute/v1"
	"google.golang.org/api/googleapi"
)

func resourceComputeTargetPool() *schema.Resource {
	return &schema.Resource{
		Create: resourceComputeTargetPoolCreate,
		Read:   resourceComputeTargetPoolRead,
		Delete: resourceComputeTargetPoolDelete,
		Update: resourceComputeTargetPoolUpdate,

		Schema: map[string]*schema.Schema{
			"backup_pool": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: false,
			},

			"description": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"failover_ratio": &schema.Schema{
				Type:     schema.TypeFloat,
				Optional: true,
				ForceNew: true,
			},

			"health_checks": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: false,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},

			"instances": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: false,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},

			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"self_link": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"session_affinity": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"region": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
		},
	}
}

func convertStringArr(ifaceArr []interface{}) []string {
	arr := make([]string, len(ifaceArr))
	for i, v := range ifaceArr {
		arr[i] = v.(string)
	}
	return arr
}

// Healthchecks need to exist before being referred to from the target pool.
func convertHealthChecks(config *Config, names []string) ([]string, error) {
	urls := make([]string, len(names))
	for i, name := range names {
		// Look up the healthcheck
		res, err := config.clientCompute.HttpHealthChecks.Get(config.Project, name).Do()
		if err != nil {
			return nil, fmt.Errorf("Error reading HealthCheck: %s", err)
		}
		urls[i] = res.SelfLink
	}
	return urls, nil
}

// Instances do not need to exist yet, so we simply generate URLs.
// Instances can be full URLS or zone/name
func convertInstances(config *Config, names []string) ([]string, error) {
	urls := make([]string, len(names))
	for i, name := range names {
		if strings.HasPrefix(name, "https://www.googleapis.com/compute/v1/") {
			urls[i] = name
		} else {
			splitName := strings.Split(name, "/")
			if len(splitName) != 2 {
				return nil, fmt.Errorf("Invalid instance name, require URL or zone/name: %s", name)
			} else {
				urls[i] = fmt.Sprintf(
					"https://www.googleapis.com/compute/v1/projects/%s/zones/%s/instances/%s",
					config.Project, splitName[0], splitName[1])
			}
		}
	}
	return urls, nil
}

func resourceComputeTargetPoolCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	region := getOptionalRegion(d, config)

	hchkUrls, err := convertHealthChecks(
		config, convertStringArr(d.Get("health_checks").([]interface{})))
	if err != nil {
		return err
	}

	instanceUrls, err := convertInstances(
		config, convertStringArr(d.Get("instances").([]interface{})))
	if err != nil {
		return err
	}

	// Build the parameter
	tpool := &compute.TargetPool{
		BackupPool:      d.Get("backup_pool").(string),
		Description:     d.Get("description").(string),
		HealthChecks:    hchkUrls,
		Instances:       instanceUrls,
		Name:            d.Get("name").(string),
		SessionAffinity: d.Get("session_affinity").(string),
	}
	if d.Get("failover_ratio") != nil {
		tpool.FailoverRatio = d.Get("failover_ratio").(float64)
	}
	log.Printf("[DEBUG] TargetPool insert request: %#v", tpool)
	op, err := config.clientCompute.TargetPools.Insert(
		config.Project, region, tpool).Do()
	if err != nil {
		return fmt.Errorf("Error creating TargetPool: %s", err)
	}

	// It probably maybe worked, so store the ID now
	d.SetId(tpool.Name)

	err = computeOperationWaitRegion(config, op, region, "Creating Target Pool")
	if err != nil {
		return err
	}
	return resourceComputeTargetPoolRead(d, meta)
}

func calcAddRemove(from []string, to []string) ([]string, []string) {
	add := make([]string, 0)
	remove := make([]string, 0)
	for _, u := range to {
		found := false
		for _, v := range from {
			if u == v {
				found = true
				break
			}
		}
		if !found {
			add = append(add, u)
		}
	}
	for _, u := range from {
		found := false
		for _, v := range to {
			if u == v {
				found = true
				break
			}
		}
		if !found {
			remove = append(remove, u)
		}
	}
	return add, remove
}

func resourceComputeTargetPoolUpdate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	region := getOptionalRegion(d, config)

	d.Partial(true)

	if d.HasChange("health_checks") {

		from_, to_ := d.GetChange("health_checks")
		from := convertStringArr(from_.([]interface{}))
		to := convertStringArr(to_.([]interface{}))
		fromUrls, err := convertHealthChecks(config, from)
		if err != nil {
			return err
		}
		toUrls, err := convertHealthChecks(config, to)
		if err != nil {
			return err
		}
		add, remove := calcAddRemove(fromUrls, toUrls)

		removeReq := &compute.TargetPoolsRemoveHealthCheckRequest{
			HealthChecks: make([]*compute.HealthCheckReference, len(remove)),
		}
		for i, v := range remove {
			removeReq.HealthChecks[i] = &compute.HealthCheckReference{HealthCheck: v}
		}
		op, err := config.clientCompute.TargetPools.RemoveHealthCheck(
			config.Project, region, d.Id(), removeReq).Do()
		if err != nil {
			return fmt.Errorf("Error updating health_check: %s", err)
		}

		err = computeOperationWaitRegion(config, op, region, "Updating Target Pool")
		if err != nil {
			return err
		}
		addReq := &compute.TargetPoolsAddHealthCheckRequest{
			HealthChecks: make([]*compute.HealthCheckReference, len(add)),
		}
		for i, v := range add {
			addReq.HealthChecks[i] = &compute.HealthCheckReference{HealthCheck: v}
		}
		op, err = config.clientCompute.TargetPools.AddHealthCheck(
			config.Project, region, d.Id(), addReq).Do()
		if err != nil {
			return fmt.Errorf("Error updating health_check: %s", err)
		}

		err = computeOperationWaitRegion(config, op, region, "Updating Target Pool")
		if err != nil {
			return err
		}
		d.SetPartial("health_checks")
	}

	if d.HasChange("instances") {

		from_, to_ := d.GetChange("instances")
		from := convertStringArr(from_.([]interface{}))
		to := convertStringArr(to_.([]interface{}))
		fromUrls, err := convertInstances(config, from)
		if err != nil {
			return err
		}
		toUrls, err := convertInstances(config, to)
		if err != nil {
			return err
		}
		add, remove := calcAddRemove(fromUrls, toUrls)

		addReq := &compute.TargetPoolsAddInstanceRequest{
			Instances: make([]*compute.InstanceReference, len(add)),
		}
		for i, v := range add {
			addReq.Instances[i] = &compute.InstanceReference{Instance: v}
		}
		op, err := config.clientCompute.TargetPools.AddInstance(
			config.Project, region, d.Id(), addReq).Do()
		if err != nil {
			return fmt.Errorf("Error updating instances: %s", err)
		}

		err = computeOperationWaitRegion(config, op, region, "Updating Target Pool")
		if err != nil {
			return err
		}
		removeReq := &compute.TargetPoolsRemoveInstanceRequest{
			Instances: make([]*compute.InstanceReference, len(remove)),
		}
		for i, v := range remove {
			removeReq.Instances[i] = &compute.InstanceReference{Instance: v}
		}
		op, err = config.clientCompute.TargetPools.RemoveInstance(
			config.Project, region, d.Id(), removeReq).Do()
		if err != nil {
			return fmt.Errorf("Error updating instances: %s", err)
		}
		err = computeOperationWaitRegion(config, op, region, "Updating Target Pool")
		if err != nil {
			return err
		}
		d.SetPartial("instances")
	}

	if d.HasChange("backup_pool") {
		bpool_name := d.Get("backup_pool").(string)
		tref := &compute.TargetReference{
			Target: bpool_name,
		}
		op, err := config.clientCompute.TargetPools.SetBackup(
			config.Project, region, d.Id(), tref).Do()
		if err != nil {
			return fmt.Errorf("Error updating backup_pool: %s", err)
		}

		err = computeOperationWaitRegion(config, op, region, "Updating Target Pool")
		if err != nil {
			return err
		}
		d.SetPartial("backup_pool")
	}

	d.Partial(false)

	return resourceComputeTargetPoolRead(d, meta)
}

func resourceComputeTargetPoolRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	region := getOptionalRegion(d, config)

	tpool, err := config.clientCompute.TargetPools.Get(
		config.Project, region, d.Id()).Do()
	if err != nil {
		if gerr, ok := err.(*googleapi.Error); ok && gerr.Code == 404 {
			log.Printf("[WARN] Removing Target Pool %q because it's gone", d.Get("name").(string))
			// The resource doesn't exist anymore
			d.SetId("")

			return nil
		}

		return fmt.Errorf("Error reading TargetPool: %s", err)
	}

	d.Set("self_link", tpool.SelfLink)

	return nil
}

func resourceComputeTargetPoolDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	region := getOptionalRegion(d, config)

	// Delete the TargetPool
	op, err := config.clientCompute.TargetPools.Delete(
		config.Project, region, d.Id()).Do()
	if err != nil {
		return fmt.Errorf("Error deleting TargetPool: %s", err)
	}

	err = computeOperationWaitRegion(config, op, region, "Deleting Target Pool")
	if err != nil {
		return err
	}
	d.SetId("")
	return nil
}
