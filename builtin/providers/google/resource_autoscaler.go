package google

import (
	"fmt"
	"log"
	"time"

	"code.google.com/p/google-api-go-client/autoscaler/v1beta2"
	"code.google.com/p/google-api-go-client/googleapi"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAutoscaler() *schema.Resource {
	return &schema.Resource{
		Create: resourceAutoscalerCreate,
		Read:   resourceAutoscalerRead,
		Update: resourceAutoscalerUpdate,
		Delete: resourceAutoscalerDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				ForceNew: true,
				Required: true,
			},

			"description": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},

			"target": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"autoscaling_policy": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"min_replicas": &schema.Schema{
							Type:     schema.TypeInt,
							Required: true,
						},

						"max_replicas": &schema.Schema{
							Type:     schema.TypeInt,
							Required: true,
						},

						"cooldown_period": &schema.Schema{
							Type:     schema.TypeInt,
							Required: true,
						},

						"cpu_utilization": &schema.Schema{
							Type:     schema.TypeList,
							Optional: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"target": &schema.Schema{
										Type:     schema.TypeFloat,
										Required: true,
									},
								},
							},
						},

						"metric": &schema.Schema{
							Type:     schema.TypeList,
							Optional: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"name": &schema.Schema{
										Type:     schema.TypeString,
										Required: true,
									},
									"target": &schema.Schema{
										Type:     schema.TypeFloat,
										Required: true,
									},

									"type": &schema.Schema{
										Type:     schema.TypeString,
										Required: true,
									},
								},
							},
						},

						"load_balancing_utilization": &schema.Schema{
							Type:     schema.TypeList,
							Optional: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"target": &schema.Schema{
										Type:     schema.TypeFloat,
										Required: true,
									},
								},
							},
						},
					},
				},
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

func resourceAutoscalerCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	// Get the zone
	log.Printf("[DEBUG] Loading zone: %s", d.Get("zone").(string))
	zone, err := config.clientCompute.Zones.Get(
		config.Project, d.Get("zone").(string)).Do()
	if err != nil {
		return fmt.Errorf(
			"Error loading zone '%s': %s", d.Get("zone").(string), err)
	}

	// Build the parameter
	scaler := &autoscaler.Autoscaler{
		Name:   d.Get("name").(string),
		Target: d.Get("target").(string),
	}

	// Optional fields
	if v, ok := d.GetOk("description"); ok {
		scaler.Description = v.(string)
	}

	prefix := "autoscaling_policy.0."

	scaler.AutoscalingPolicy = &autoscaler.AutoscalingPolicy{
		MaxNumReplicas:    int64(d.Get(prefix + "max_replicas").(int)),
		MinNumReplicas:    int64(d.Get(prefix + "min_replicas").(int)),
		CoolDownPeriodSec: int64(d.Get(prefix + "cooldown_period").(int)),
	}

	// Check that only one autoscaling policy is defined

	policyCounter := 0
	if _, ok := d.GetOk(prefix + "cpu_utilization"); ok {
		policyCounter++
		scaler.AutoscalingPolicy.CpuUtilization = &autoscaler.AutoscalingPolicyCpuUtilization{
			UtilizationTarget: d.Get(prefix + "cpu_utilization.0.target").(float64),
		}
	}
	if _, ok := d.GetOk("autoscaling_policy.0.metric"); ok {
		policyCounter++
		scaler.AutoscalingPolicy.CustomMetricUtilizations = []*autoscaler.AutoscalingPolicyCustomMetricUtilization{
			{
				Metric:                d.Get(prefix + "metric.0.name").(string),
				UtilizationTarget:     d.Get(prefix + "metric.0.target").(float64),
				UtilizationTargetType: d.Get(prefix + "metric.0.type").(string),
			},
		}

	}
	if _, ok := d.GetOk("autoscaling_policy.0.load_balancing_utilization"); ok {
		policyCounter++
		scaler.AutoscalingPolicy.LoadBalancingUtilization = &autoscaler.AutoscalingPolicyLoadBalancingUtilization{
			UtilizationTarget: d.Get(prefix + "load_balancing_utilization.0.target").(float64),
		}
	}

	if policyCounter != 1 {
		return fmt.Errorf("One policy must be defined for an autoscaler.")
	}

	op, err := config.clientAutoscaler.Autoscalers.Insert(
		config.Project, zone.Name, scaler).Do()
	if err != nil {
		return fmt.Errorf("Error creating Autoscaler: %s", err)
	}

	// It probably maybe worked, so store the ID now
	d.SetId(scaler.Name)

	// Wait for the operation to complete
	w := &AutoscalerOperationWaiter{
		Service: config.clientAutoscaler,
		Op:      op,
		Project: config.Project,
		Zone:    zone.Name,
	}
	state := w.Conf()
	state.Timeout = 2 * time.Minute
	state.MinTimeout = 1 * time.Second
	opRaw, err := state.WaitForState()
	if err != nil {
		return fmt.Errorf("Error waiting for Autoscaler to create: %s", err)
	}
	op = opRaw.(*autoscaler.Operation)
	if op.Error != nil {
		// The resource didn't actually create
		d.SetId("")

		// Return the error
		return AutoscalerOperationError(*op.Error)
	}

	return resourceAutoscalerRead(d, meta)
}

func resourceAutoscalerRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	zone := d.Get("zone").(string)
	scaler, err := config.clientAutoscaler.Autoscalers.Get(
		config.Project, zone, d.Id()).Do()
	if err != nil {
		if gerr, ok := err.(*googleapi.Error); ok && gerr.Code == 404 {
			// The resource doesn't exist anymore
			d.SetId("")

			return nil
		}

		return fmt.Errorf("Error reading Autoscaler: %s", err)
	}

	d.Set("self_link", scaler.SelfLink)

	return nil
}

func resourceAutoscalerUpdate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	zone := d.Get("zone").(string)

	// Build the parameter
	scaler := &autoscaler.Autoscaler{
		Name:   d.Get("name").(string),
		Target: d.Get("target").(string),
	}

	// Optional fields
	if v, ok := d.GetOk("description"); ok {
		scaler.Description = v.(string)
	}

	prefix := "autoscaling_policy.0."

	scaler.AutoscalingPolicy = &autoscaler.AutoscalingPolicy{
		MaxNumReplicas:    int64(d.Get(prefix + "max_replicas").(int)),
		MinNumReplicas:    int64(d.Get(prefix + "min_replicas").(int)),
		CoolDownPeriodSec: int64(d.Get(prefix + "cooldown_period").(int)),
	}

	// Check that only one autoscaling policy is defined

	policyCounter := 0
	if _, ok := d.GetOk(prefix + "cpu_utilization"); ok {
		if d.Get(prefix+"cpu_utilization.0.target").(float64) != 0 {
			policyCounter++
			scaler.AutoscalingPolicy.CpuUtilization = &autoscaler.AutoscalingPolicyCpuUtilization{
				UtilizationTarget: d.Get(prefix + "cpu_utilization.0.target").(float64),
			}
		}
	}
	if _, ok := d.GetOk("autoscaling_policy.0.metric"); ok {
		if d.Get(prefix+"metric.0.name") != "" {
			policyCounter++
			scaler.AutoscalingPolicy.CustomMetricUtilizations = []*autoscaler.AutoscalingPolicyCustomMetricUtilization{
				{
					Metric:                d.Get(prefix + "metric.0.name").(string),
					UtilizationTarget:     d.Get(prefix + "metric.0.target").(float64),
					UtilizationTargetType: d.Get(prefix + "metric.0.type").(string),
				},
			}
		}

	}
	if _, ok := d.GetOk("autoscaling_policy.0.load_balancing_utilization"); ok {
		if d.Get(prefix+"load_balancing_utilization.0.target").(float64) != 0 {
			policyCounter++
			scaler.AutoscalingPolicy.LoadBalancingUtilization = &autoscaler.AutoscalingPolicyLoadBalancingUtilization{
				UtilizationTarget: d.Get(prefix + "load_balancing_utilization.0.target").(float64),
			}
		}
	}

	if policyCounter != 1 {
		return fmt.Errorf("One policy must be defined for an autoscaler.")
	}

	op, err := config.clientAutoscaler.Autoscalers.Patch(
		config.Project, zone, d.Id(), scaler).Do()
	if err != nil {
		return fmt.Errorf("Error updating Autoscaler: %s", err)
	}

	// It probably maybe worked, so store the ID now
	d.SetId(scaler.Name)

	// Wait for the operation to complete
	w := &AutoscalerOperationWaiter{
		Service: config.clientAutoscaler,
		Op:      op,
		Project: config.Project,
		Zone:    zone,
	}
	state := w.Conf()
	state.Timeout = 2 * time.Minute
	state.MinTimeout = 1 * time.Second
	opRaw, err := state.WaitForState()
	if err != nil {
		return fmt.Errorf("Error waiting for Autoscaler to update: %s", err)
	}
	op = opRaw.(*autoscaler.Operation)
	if op.Error != nil {
		// Return the error
		return AutoscalerOperationError(*op.Error)
	}

	return resourceAutoscalerRead(d, meta)
}

func resourceAutoscalerDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	zone := d.Get("zone").(string)
	op, err := config.clientAutoscaler.Autoscalers.Delete(
		config.Project, zone, d.Id()).Do()
	if err != nil {
		return fmt.Errorf("Error deleting autoscaler: %s", err)
	}

	// Wait for the operation to complete
	w := &AutoscalerOperationWaiter{
		Service: config.clientAutoscaler,
		Op:      op,
		Project: config.Project,
		Zone:    zone,
	}
	state := w.Conf()
	state.Timeout = 2 * time.Minute
	state.MinTimeout = 1 * time.Second
	opRaw, err := state.WaitForState()
	if err != nil {
		return fmt.Errorf("Error waiting for Autoscaler to delete: %s", err)
	}
	op = opRaw.(*autoscaler.Operation)
	if op.Error != nil {
		// Return the error
		return AutoscalerOperationError(*op.Error)
	}

	d.SetId("")
	return nil
}
