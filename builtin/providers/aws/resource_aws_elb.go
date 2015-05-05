package aws

import (
	"bytes"
	"fmt"
	"log"

	"github.com/awslabs/aws-sdk-go/aws"
	"github.com/awslabs/aws-sdk-go/service/elb"
	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/schema"
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

			"cross_zone_load_balancing": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
			},

			"availability_zones": &schema.Schema{
				Type:     schema.TypeSet,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Optional: true,
				ForceNew: true,
				Computed: true,
				Set:      schema.HashString,
			},

			"instances": &schema.Schema{
				Type:     schema.TypeSet,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Optional: true,
				Computed: true,
				Set:      schema.HashString,
			},

			"security_groups": &schema.Schema{
				Type:     schema.TypeSet,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Optional: true,
				Computed: true,
				Set:      schema.HashString,
			},

			"source_security_group": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"subnets": &schema.Schema{
				Type:     schema.TypeSet,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Optional: true,
				ForceNew: true,
				Computed: true,
				Set:      schema.HashString,
			},

			"idle_timeout": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				Default:  60,
			},

			"connection_draining": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},

			"connection_draining_timeout": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				Default:  300,
			},

			"listener": &schema.Schema{
				Type:     schema.TypeSet,
				Required: true,
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

			"health_check": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
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

			"zone_id": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"tags": tagsSchema(),
		},
	}
}

func resourceAwsElbCreate(d *schema.ResourceData, meta interface{}) error {
	elbconn := meta.(*AWSClient).elbconn

	// Expand the "listener" set to aws-sdk-go compat []*elb.Listener
	listeners, err := expandListeners(d.Get("listener").(*schema.Set).List())
	if err != nil {
		return err
	}

	tags := tagsFromMapELB(d.Get("tags").(map[string]interface{}))
	// Provision the elb
	elbOpts := &elb.CreateLoadBalancerInput{
		LoadBalancerName: aws.String(d.Get("name").(string)),
		Listeners:        listeners,
		Tags:             tags,
	}

	if scheme, ok := d.GetOk("internal"); ok && scheme.(bool) {
		elbOpts.Scheme = aws.String("internal")
	}

	if v, ok := d.GetOk("availability_zones"); ok {
		elbOpts.AvailabilityZones = expandStringList(v.(*schema.Set).List())
	}

	if v, ok := d.GetOk("security_groups"); ok {
		elbOpts.SecurityGroups = expandStringList(v.(*schema.Set).List())
	}

	if v, ok := d.GetOk("subnets"); ok {
		elbOpts.Subnets = expandStringList(v.(*schema.Set).List())
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

	d.Set("tags", tagsToMapELB(tags))

	return resourceAwsElbUpdate(d, meta)
}

func resourceAwsElbRead(d *schema.ResourceData, meta interface{}) error {
	elbconn := meta.(*AWSClient).elbconn
	elbName := d.Id()

	// Retrieve the ELB properties for updating the state
	describeElbOpts := &elb.DescribeLoadBalancersInput{
		LoadBalancerNames: []*string{aws.String(elbName)},
	}

	describeResp, err := elbconn.DescribeLoadBalancers(describeElbOpts)
	if err != nil {
		if isLoadBalancerNotFound(err) {
			// The ELB is gone now, so just remove it from the state
			d.SetId("")
			return nil
		}

		return fmt.Errorf("Error retrieving ELB: %s", err)
	}
	if len(describeResp.LoadBalancerDescriptions) != 1 {
		return fmt.Errorf("Unable to find ELB: %#v", describeResp.LoadBalancerDescriptions)
	}

	describeAttrsOpts := &elb.DescribeLoadBalancerAttributesInput{
		LoadBalancerName: aws.String(elbName),
	}
	describeAttrsResp, err := elbconn.DescribeLoadBalancerAttributes(describeAttrsOpts)
	if err != nil {
		if isLoadBalancerNotFound(err) {
			// The ELB is gone now, so just remove it from the state
			d.SetId("")
			return nil
		}

		return fmt.Errorf("Error retrieving ELB: %s", err)
	}

	lbAttrs := describeAttrsResp.LoadBalancerAttributes

	lb := describeResp.LoadBalancerDescriptions[0]

	d.Set("name", *lb.LoadBalancerName)
	d.Set("dns_name", *lb.DNSName)
	d.Set("zone_id", *lb.CanonicalHostedZoneNameID)
	d.Set("internal", *lb.Scheme == "internal")
	d.Set("availability_zones", lb.AvailabilityZones)
	d.Set("instances", flattenInstances(lb.Instances))
	d.Set("listener", flattenListeners(lb.ListenerDescriptions))
	d.Set("security_groups", lb.SecurityGroups)
	if lb.SourceSecurityGroup != nil {
		d.Set("source_security_group", lb.SourceSecurityGroup.GroupName)
	}
	d.Set("subnets", lb.Subnets)
	d.Set("idle_timeout", lbAttrs.ConnectionSettings.IdleTimeout)
	d.Set("connection_draining", lbAttrs.ConnectionDraining.Enabled)
	d.Set("connection_draining_timeout", lbAttrs.ConnectionDraining.Timeout)

	resp, err := elbconn.DescribeTags(&elb.DescribeTagsInput{
		LoadBalancerNames: []*string{lb.LoadBalancerName},
	})

	var et []*elb.Tag
	if len(resp.TagDescriptions) > 0 {
		et = resp.TagDescriptions[0].Tags
	}
	d.Set("tags", tagsToMapELB(et))
	// There's only one health check, so save that to state as we
	// currently can
	if *lb.HealthCheck.Target != "" {
		d.Set("health_check", flattenHealthCheck(lb.HealthCheck))
	}

	return nil
}

func resourceAwsElbUpdate(d *schema.ResourceData, meta interface{}) error {
	elbconn := meta.(*AWSClient).elbconn

	d.Partial(true)

	if d.HasChange("listener") {
		o, n := d.GetChange("listener")
		os := o.(*schema.Set)
		ns := n.(*schema.Set)

		remove, _ := expandListeners(os.Difference(ns).List())
		add, _ := expandListeners(ns.Difference(os).List())

		if len(remove) > 0 {
			ports := make([]*int64, 0, len(remove))
			for _, listener := range remove {
				ports = append(ports, listener.LoadBalancerPort)
			}

			deleteListenersOpts := &elb.DeleteLoadBalancerListenersInput{
				LoadBalancerName:  aws.String(d.Id()),
				LoadBalancerPorts: ports,
			}

			_, err := elbconn.DeleteLoadBalancerListeners(deleteListenersOpts)
			if err != nil {
				return fmt.Errorf("Failure removing outdated listeners: %s", err)
			}
		}

		if len(add) > 0 {
			createListenersOpts := &elb.CreateLoadBalancerListenersInput{
				LoadBalancerName: aws.String(d.Id()),
				Listeners:        add,
			}

			_, err := elbconn.CreateLoadBalancerListeners(createListenersOpts)
			if err != nil {
				return fmt.Errorf("Failure adding new or updated listeners: %s", err)
			}
		}

		d.SetPartial("listener")
	}

	// If we currently have instances, or did have instances,
	// we want to figure out what to add and remove from the load
	// balancer
	if d.HasChange("instances") {
		o, n := d.GetChange("instances")
		os := o.(*schema.Set)
		ns := n.(*schema.Set)
		remove := expandInstanceString(os.Difference(ns).List())
		add := expandInstanceString(ns.Difference(os).List())

		if len(add) > 0 {
			registerInstancesOpts := elb.RegisterInstancesWithLoadBalancerInput{
				LoadBalancerName: aws.String(d.Id()),
				Instances:        add,
			}

			_, err := elbconn.RegisterInstancesWithLoadBalancer(&registerInstancesOpts)
			if err != nil {
				return fmt.Errorf("Failure registering instances: %s", err)
			}
		}
		if len(remove) > 0 {
			deRegisterInstancesOpts := elb.DeregisterInstancesFromLoadBalancerInput{
				LoadBalancerName: aws.String(d.Id()),
				Instances:        remove,
			}

			_, err := elbconn.DeregisterInstancesFromLoadBalancer(&deRegisterInstancesOpts)
			if err != nil {
				return fmt.Errorf("Failure deregistering instances: %s", err)
			}
		}

		d.SetPartial("instances")
	}

	log.Println("[INFO] outside modify attributes")
	if d.HasChange("cross_zone_load_balancing") || d.HasChange("idle_timeout") || d.HasChange("connection_draining") || d.HasChange("connection_draining_timeout") {
		log.Println("[INFO] inside modify attributes")
		attrs := elb.ModifyLoadBalancerAttributesInput{
			LoadBalancerName: aws.String(d.Get("name").(string)),
			LoadBalancerAttributes: &elb.LoadBalancerAttributes{
				CrossZoneLoadBalancing: &elb.CrossZoneLoadBalancing{
					Enabled: aws.Boolean(d.Get("cross_zone_load_balancing").(bool)),
				},
				ConnectionSettings: &elb.ConnectionSettings{
					IdleTimeout: aws.Long(int64(d.Get("idle_timeout").(int))),
				},
				ConnectionDraining: &elb.ConnectionDraining{
					Enabled: aws.Boolean(d.Get("connection_draining").(bool)),
					Timeout: aws.Long(int64(d.Get("connection_draining_timeout").(int))),
				},
			},
		}
		_, err := elbconn.ModifyLoadBalancerAttributes(&attrs)
		if err != nil {
			return fmt.Errorf("Failure configuring elb attributes: %s", err)
		}
		d.SetPartial("cross_zone_load_balancing")
		d.SetPartial("idle_timeout")
		d.SetPartial("connection_draining")
		d.SetPartial("connection_draining_timeout")
	}

	if d.HasChange("health_check") {
		vs := d.Get("health_check").(*schema.Set).List()
		if len(vs) > 0 {
			check := vs[0].(map[string]interface{})
			configureHealthCheckOpts := elb.ConfigureHealthCheckInput{
				LoadBalancerName: aws.String(d.Id()),
				HealthCheck: &elb.HealthCheck{
					HealthyThreshold:   aws.Long(int64(check["healthy_threshold"].(int))),
					UnhealthyThreshold: aws.Long(int64(check["unhealthy_threshold"].(int))),
					Interval:           aws.Long(int64(check["interval"].(int))),
					Target:             aws.String(check["target"].(string)),
					Timeout:            aws.Long(int64(check["timeout"].(int))),
				},
			}
			_, err := elbconn.ConfigureHealthCheck(&configureHealthCheckOpts)
			if err != nil {
				return fmt.Errorf("Failure configuring health check: %s", err)
			}
			d.SetPartial("health_check")
		}
	}

	if d.HasChange("security_groups") {
		groups := d.Get("security_groups").(*schema.Set).List()

		applySecurityGroupsOpts := elb.ApplySecurityGroupsToLoadBalancerInput{
			LoadBalancerName: aws.String(d.Id()),
			SecurityGroups:   expandStringList(groups),
		}

		_, err := elbconn.ApplySecurityGroupsToLoadBalancer(&applySecurityGroupsOpts)
		if err != nil {
			return fmt.Errorf("Failure applying security groups: %s", err)
		}

		d.SetPartial("security_groups")
	}

	if err := setTagsELB(elbconn, d); err != nil {
		return err
	}

	d.SetPartial("tags")
	d.Partial(false)

	return resourceAwsElbRead(d, meta)
}

func resourceAwsElbDelete(d *schema.ResourceData, meta interface{}) error {
	elbconn := meta.(*AWSClient).elbconn

	log.Printf("[INFO] Deleting ELB: %s", d.Id())

	// Destroy the load balancer
	deleteElbOpts := elb.DeleteLoadBalancerInput{
		LoadBalancerName: aws.String(d.Id()),
	}
	if _, err := elbconn.DeleteLoadBalancer(&deleteElbOpts); err != nil {
		return fmt.Errorf("Error deleting ELB: %s", err)
	}

	return nil
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

func isLoadBalancerNotFound(err error) bool {
	elberr, ok := err.(aws.APIError)
	return ok && elberr.Code == "LoadBalancerNotFound"
}
