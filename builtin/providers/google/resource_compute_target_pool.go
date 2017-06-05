package google

import (
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/terraform/helper/schema"
	"google.golang.org/api/compute/v1"
)

func resourceComputeTargetPool() *schema.Resource {
	return &schema.Resource{
		Create: resourceComputeTargetPoolCreate,
		Read:   resourceComputeTargetPoolRead,
		Delete: resourceComputeTargetPoolDelete,
		Update: resourceComputeTargetPoolUpdate,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"backup_pool": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: false,
			},

			"description": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"failover_ratio": {
				Type:     schema.TypeFloat,
				Optional: true,
				ForceNew: true,
			},

			"health_checks": {
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: false,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},

			"instances": {
				Type:     schema.TypeList,
				Optional: true,
				Computed: true,
				ForceNew: false,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},

			"project": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Computed: true,
			},

			"region": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Computed: true,
			},

			"self_link": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"session_affinity": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Default:  "NONE",
			},
		},
	}
}

func convertStringArr(ifaceArr []interface{}) []string {
	var arr []string
	for _, v := range ifaceArr {
		if v == nil {
			continue
		}
		arr = append(arr, v.(string))
	}
	return arr
}

// Healthchecks need to exist before being referred to from the target pool.
func convertHealthChecks(config *Config, project string, names []string) ([]string, error) {
	urls := make([]string, len(names))
	for i, name := range names {
		// Look up the healthcheck
		res, err := config.clientCompute.HttpHealthChecks.Get(project, name).Do()
		if err != nil {
			return nil, fmt.Errorf("Error reading HealthCheck: %s", err)
		}
		urls[i] = res.SelfLink
	}
	return urls, nil
}

// Instances do not need to exist yet, so we simply generate URLs.
// Instances can be full URLS or zone/name
func convertInstancesToUrls(config *Config, project string, names []string) ([]string, error) {
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
					project, splitName[0], splitName[1])
			}
		}
	}
	return urls, nil
}

func resourceComputeTargetPoolCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	region, err := getRegion(d, config)
	if err != nil {
		return err
	}

	project, err := getProject(d, config)
	if err != nil {
		return err
	}

	hchkUrls, err := convertHealthChecks(
		config, project, convertStringArr(d.Get("health_checks").([]interface{})))
	if err != nil {
		return err
	}

	instanceUrls, err := convertInstancesToUrls(
		config, project, convertStringArr(d.Get("instances").([]interface{})))
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
		project, region, tpool).Do()
	if err != nil {
		return fmt.Errorf("Error creating TargetPool: %s", err)
	}

	// It probably maybe worked, so store the ID now
	d.SetId(tpool.Name)

	err = computeOperationWaitRegion(config, op, project, region, "Creating Target Pool")
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

	region, err := getRegion(d, config)
	if err != nil {
		return err
	}

	project, err := getProject(d, config)
	if err != nil {
		return err
	}

	d.Partial(true)

	if d.HasChange("health_checks") {

		from_, to_ := d.GetChange("health_checks")
		from := convertStringArr(from_.([]interface{}))
		to := convertStringArr(to_.([]interface{}))
		fromUrls, err := convertHealthChecks(config, project, from)
		if err != nil {
			return err
		}
		toUrls, err := convertHealthChecks(config, project, to)
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
			project, region, d.Id(), removeReq).Do()
		if err != nil {
			return fmt.Errorf("Error updating health_check: %s", err)
		}

		err = computeOperationWaitRegion(config, op, project, region, "Updating Target Pool")
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
			project, region, d.Id(), addReq).Do()
		if err != nil {
			return fmt.Errorf("Error updating health_check: %s", err)
		}

		err = computeOperationWaitRegion(config, op, project, region, "Updating Target Pool")
		if err != nil {
			return err
		}
		d.SetPartial("health_checks")
	}

	if d.HasChange("instances") {

		from_, to_ := d.GetChange("instances")
		from := convertStringArr(from_.([]interface{}))
		to := convertStringArr(to_.([]interface{}))
		fromUrls, err := convertInstancesToUrls(config, project, from)
		if err != nil {
			return err
		}
		toUrls, err := convertInstancesToUrls(config, project, to)
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
			project, region, d.Id(), addReq).Do()
		if err != nil {
			return fmt.Errorf("Error updating instances: %s", err)
		}

		err = computeOperationWaitRegion(config, op, project, region, "Updating Target Pool")
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
			project, region, d.Id(), removeReq).Do()
		if err != nil {
			return fmt.Errorf("Error updating instances: %s", err)
		}
		err = computeOperationWaitRegion(config, op, project, region, "Updating Target Pool")
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
			project, region, d.Id(), tref).Do()
		if err != nil {
			return fmt.Errorf("Error updating backup_pool: %s", err)
		}

		err = computeOperationWaitRegion(config, op, project, region, "Updating Target Pool")
		if err != nil {
			return err
		}
		d.SetPartial("backup_pool")
	}

	d.Partial(false)

	return resourceComputeTargetPoolRead(d, meta)
}

func convertInstancesFromUrls(urls []string) []string {
	result := make([]string, 0, len(urls))
	for _, url := range urls {
		urlArray := strings.Split(url, "/")
		instance := fmt.Sprintf("%s/%s", urlArray[len(urlArray)-3], urlArray[len(urlArray)-1])
		result = append(result, instance)
	}
	return result
}

func convertHealthChecksFromUrls(urls []string) []string {
	result := make([]string, 0, len(urls))
	for _, url := range urls {
		urlArray := strings.Split(url, "/")
		healthCheck := fmt.Sprintf("%s", urlArray[len(urlArray)-1])
		result = append(result, healthCheck)
	}
	return result
}

func resourceComputeTargetPoolRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	region, err := getRegion(d, config)
	if err != nil {
		return err
	}

	project, err := getProject(d, config)
	if err != nil {
		return err
	}

	tpool, err := config.clientCompute.TargetPools.Get(
		project, region, d.Id()).Do()
	if err != nil {
		return handleNotFoundError(err, d, fmt.Sprintf("Target Pool %q", d.Get("name").(string)))
	}

	regionUrl := strings.Split(tpool.Region, "/")
	d.Set("self_link", tpool.SelfLink)
	d.Set("backup_pool", tpool.BackupPool)
	d.Set("description", tpool.Description)
	d.Set("failover_ratio", tpool.FailoverRatio)
	if tpool.HealthChecks != nil {
		d.Set("health_checks", convertHealthChecksFromUrls(tpool.HealthChecks))
	} else {
		d.Set("health_checks", nil)
	}
	if tpool.Instances != nil {
		d.Set("instances", convertInstancesFromUrls(tpool.Instances))
	} else {
		d.Set("instances", nil)
	}
	d.Set("name", tpool.Name)
	d.Set("region", regionUrl[len(regionUrl)-1])
	d.Set("session_affinity", tpool.SessionAffinity)
	d.Set("project", project)
	return nil
}

func resourceComputeTargetPoolDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	region, err := getRegion(d, config)
	if err != nil {
		return err
	}

	project, err := getProject(d, config)
	if err != nil {
		return err
	}

	// Delete the TargetPool
	op, err := config.clientCompute.TargetPools.Delete(
		project, region, d.Id()).Do()
	if err != nil {
		return fmt.Errorf("Error deleting TargetPool: %s", err)
	}

	err = computeOperationWaitRegion(config, op, project, region, "Deleting Target Pool")
	if err != nil {
		return err
	}
	d.SetId("")
	return nil
}
