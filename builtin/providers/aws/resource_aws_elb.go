package aws

import (
	"bytes"
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/mitchellh/goamz/elb"
)

func resourceAwsElb() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsElbCreate,
		Read:   resourceAwsElbRead,
		Update: resourceAwsElbUpdate,
		Delete: resourceAwsElbDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"internal": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				ForceNew: true,
				Computed: true,
			},

			"availability_zones": &schema.Schema{
				Type:     schema.TypeList,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Optional: true,
				ForceNew: true,
			},

			"instances": &schema.Schema{
				Type:     schema.TypeSet,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Optional: true,
				Computed: true,
				Set: func(v interface{}) int {
					return hashcode.String(v.(string))
				},
			},

			// TODO: could be not ForceNew
			"security_groups": &schema.Schema{
				Type:     schema.TypeList,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Optional: true,
				ForceNew: true,
				Computed: true,
			},

			// TODO: could be not ForceNew
			"subnets": &schema.Schema{
				Type:     schema.TypeList,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Optional: true,
				ForceNew: true,
				Computed: true,
			},

			// TODO: could be not ForceNew
			"listener": &schema.Schema{
				Type:     schema.TypeSet,
				Required: true,
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"instance_port": &schema.Schema{
							Type:     schema.TypeInt,
							Required: true,
						},

						"instance_protocol": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},

						"lb_port": &schema.Schema{
							Type:     schema.TypeInt,
							Required: true,
						},

						"lb_protocol": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},

						"ssl_certificate_id": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
					},
				},
				Set: resourceAwsElbListenerHash,
			},

			// TODO: could be not ForceNew
			"health_check": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: true,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"healthy_threshold": &schema.Schema{
							Type:     schema.TypeInt,
							Required: true,
						},

						"unhealthy_threshold": &schema.Schema{
							Type:     schema.TypeInt,
							Required: true,
						},

						"target": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},

						"interval": &schema.Schema{
							Type:     schema.TypeInt,
							Required: true,
						},

						"timeout": &schema.Schema{
							Type:     schema.TypeInt,
							Required: true,
						},
					},
				},
				Set: resourceAwsElbHealthCheckHash,
			},

			"dns_name": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceAwsElbHealthCheckHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})
	buf.WriteString(fmt.Sprintf("%d-", m["healthy_threshold"].(int)))
	buf.WriteString(fmt.Sprintf("%d-", m["unhealthy_threshold"].(int)))
	buf.WriteString(fmt.Sprintf("%s-", m["target"].(string)))
	buf.WriteString(fmt.Sprintf("%d-", m["interval"].(int)))
	buf.WriteString(fmt.Sprintf("%d-", m["timeout"].(int)))

	return hashcode.String(buf.String())
}

func resourceAwsElbListenerHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})
	buf.WriteString(fmt.Sprintf("%d-", m["instance_port"].(int)))
	buf.WriteString(fmt.Sprintf("%s-", m["instance_protocol"].(string)))
	buf.WriteString(fmt.Sprintf("%d-", m["lb_port"].(int)))
	buf.WriteString(fmt.Sprintf("%s-", m["lb_protocol"].(string)))

	if v, ok := m["ssl_certificate_id"]; ok {
		buf.WriteString(fmt.Sprintf("%s-", v.(string)))
	}

	return hashcode.String(buf.String())
}

func resourceAwsElbCreate(d *schema.ResourceData, meta interface{}) error {
	p := meta.(*ResourceProvider)
	elbconn := p.elbconn

	// Expand the "listener" set to goamz compat []elb.Listener
	listeners, err := expandListeners(d.Get("listener").(*schema.Set).List())
	if err != nil {
		return err
	}

	// Provision the elb
	elbOpts := &elb.CreateLoadBalancer{
		LoadBalancerName: d.Get("name").(string),
		Listeners:        listeners,
		Internal:         d.Get("internal").(bool),
	}

	if v, ok := d.GetOk("availability_zones"); ok {
		elbOpts.AvailZone = expandStringList(v.([]interface{}))
	}

	if v, ok := d.GetOk("security_groups"); ok {
		elbOpts.SecurityGroups = expandStringList(v.([]interface{}))
	}

	if v, ok := d.GetOk("subnets"); ok {
		elbOpts.Subnets = expandStringList(v.([]interface{}))
	}

	log.Printf("[DEBUG] ELB create configuration: %#v", elbOpts)
	if _, err := elbconn.CreateLoadBalancer(elbOpts); err != nil {
		return fmt.Errorf("Error creating ELB: %s", err)
	}

	// Assign the elb's unique identifier for use later
	d.SetId(d.Get("name").(string))
	log.Printf("[INFO] ELB ID: %s", d.Id())

	// Enable partial mode and record what we set
	d.Partial(true)
	d.SetPartial("name")
	d.SetPartial("internal")
	d.SetPartial("availability_zones")
	d.SetPartial("listener")
	d.SetPartial("security_groups")
	d.SetPartial("subnets")

	if d.HasChange("health_check") {
		vs := d.Get("health_check").(*schema.Set).List()
		if len(vs) > 0 {
			check := vs[0].(map[string]interface{})

			configureHealthCheckOpts := elb.ConfigureHealthCheck{
				LoadBalancerName: d.Id(),
				Check: elb.HealthCheck{
					HealthyThreshold:   int64(check["healthy_threshold"].(int)),
					UnhealthyThreshold: int64(check["unhealthy_threshold"].(int)),
					Interval:           int64(check["interval"].(int)),
					Target:             check["target"].(string),
					Timeout:            int64(check["timeout"].(int)),
				},
			}

			_, err = elbconn.ConfigureHealthCheck(&configureHealthCheckOpts)
			if err != nil {
				return fmt.Errorf("Failure configuring health check: %s", err)
			}
		}
	}

	return resourceAwsElbUpdate(d, meta)
}

func resourceAwsElbUpdate(d *schema.ResourceData, meta interface{}) error {
	p := meta.(*ResourceProvider)
	elbconn := p.elbconn

	d.Partial(true)

	// If we currently have instances, or did have instances,
	// we want to figure out what to add and remove from the load
	// balancer
	if d.HasChange("instances") {
		o, n := d.GetChange("instances")
		os := o.(*schema.Set)
		ns := n.(*schema.Set)
		remove := expandStringList(os.Difference(ns).List())
		add := expandStringList(ns.Difference(os).List())

		if len(add) > 0 {
			registerInstancesOpts := elb.RegisterInstancesWithLoadBalancer{
				LoadBalancerName: d.Id(),
				Instances:        add,
			}

			_, err := elbconn.RegisterInstancesWithLoadBalancer(&registerInstancesOpts)
			if err != nil {
				return fmt.Errorf("Failure registering instances: %s", err)
			}
		}
		if len(remove) > 0 {
			deRegisterInstancesOpts := elb.DeregisterInstancesFromLoadBalancer{
				LoadBalancerName: d.Id(),
				Instances:        remove,
			}

			_, err := elbconn.DeregisterInstancesFromLoadBalancer(&deRegisterInstancesOpts)
			if err != nil {
				return fmt.Errorf("Failure deregistering instances: %s", err)
			}
		}

		d.SetPartial("instances")
	}

	d.Partial(false)
	return resourceAwsElbRead(d, meta)
}

func resourceAwsElbDelete(d *schema.ResourceData, meta interface{}) error {
	p := meta.(*ResourceProvider)
	elbconn := p.elbconn

	log.Printf("[INFO] Deleting ELB: %s", d.Id())

	// Destroy the load balancer
	deleteElbOpts := elb.DeleteLoadBalancer{
		LoadBalancerName: d.Id(),
	}
	if _, err := elbconn.DeleteLoadBalancer(&deleteElbOpts); err != nil {
		return fmt.Errorf("Error deleting ELB: %s", err)
	}

	return nil
}

func resourceAwsElbRead(d *schema.ResourceData, meta interface{}) error {
	p := meta.(*ResourceProvider)
	elbconn := p.elbconn

	// Retrieve the ELB properties for updating the state
	describeElbOpts := &elb.DescribeLoadBalancer{
		Names: []string{d.Id()},
	}

	describeResp, err := elbconn.DescribeLoadBalancers(describeElbOpts)
	if err != nil {
		if ec2err, ok := err.(*elb.Error); ok && ec2err.Code == "LoadBalancerNotFound" {
			// The ELB is gone now, so just remove it from the state
			d.SetId("")
			return nil
		}

		return fmt.Errorf("Error retrieving ELB: %s", err)
	}
	if len(describeResp.LoadBalancers) != 1 {
		return fmt.Errorf("Unable to find ELB: %#v", describeResp.LoadBalancers)
	}

	lb := describeResp.LoadBalancers[0]

	d.Set("name", lb.LoadBalancerName)
	d.Set("dns_name", lb.DNSName)
	d.Set("internal", lb.Scheme == "internal")
	d.Set("instances", flattenInstances(lb.Instances))
	d.Set("listener", flattenListeners(lb.Listeners))
	d.Set("security_groups", lb.SecurityGroups)
	d.Set("subnets", lb.Subnets)

	// There's only one health check, so save that to state as we
	// currently can
	if lb.HealthCheck.Target != "" {
		d.Set("health_check", flattenHealthCheck(lb.HealthCheck))
	}

	return nil
}
