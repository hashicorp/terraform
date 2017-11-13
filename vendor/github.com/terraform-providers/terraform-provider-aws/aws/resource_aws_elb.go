package aws

import (
	"bytes"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/elb"
	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsElb() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsElbCreate,
		Read:   resourceAwsElbRead,
		Update: resourceAwsElbUpdate,
		Delete: resourceAwsElbDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:          schema.TypeString,
				Optional:      true,
				Computed:      true,
				ForceNew:      true,
				ConflictsWith: []string{"name_prefix"},
				ValidateFunc:  validateElbName,
			},
			"name_prefix": &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				ForceNew:     true,
				ValidateFunc: validateElbNamePrefix,
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
				Default:  true,
			},

			"availability_zones": &schema.Schema{
				Type:     schema.TypeSet,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Optional: true,
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

			"source_security_group_id": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"subnets": &schema.Schema{
				Type:     schema.TypeSet,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Optional: true,
				Computed: true,
				Set:      schema.HashString,
			},

			"idle_timeout": &schema.Schema{
				Type:         schema.TypeInt,
				Optional:     true,
				Default:      60,
				ValidateFunc: validateIntegerInRange(1, 3600),
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

			"access_logs": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"interval": &schema.Schema{
							Type:         schema.TypeInt,
							Optional:     true,
							Default:      60,
							ValidateFunc: validateAccessLogsInterval,
						},
						"bucket": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"bucket_prefix": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"enabled": &schema.Schema{
							Type:     schema.TypeBool,
							Optional: true,
							Default:  true,
						},
					},
				},
			},

			"listener": &schema.Schema{
				Type:     schema.TypeSet,
				Required: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"instance_port": &schema.Schema{
							Type:         schema.TypeInt,
							Required:     true,
							ValidateFunc: validateIntegerInRange(1, 65535),
						},

						"instance_protocol": &schema.Schema{
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: validateListenerProtocol,
						},

						"lb_port": &schema.Schema{
							Type:         schema.TypeInt,
							Required:     true,
							ValidateFunc: validateIntegerInRange(1, 65535),
						},

						"lb_protocol": &schema.Schema{
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: validateListenerProtocol,
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
				Type:     schema.TypeList,
				Optional: true,
				Computed: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"healthy_threshold": &schema.Schema{
							Type:         schema.TypeInt,
							Required:     true,
							ValidateFunc: validateIntegerInRange(2, 10),
						},

						"unhealthy_threshold": &schema.Schema{
							Type:         schema.TypeInt,
							Required:     true,
							ValidateFunc: validateIntegerInRange(2, 10),
						},

						"target": &schema.Schema{
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: validateHeathCheckTarget,
						},

						"interval": &schema.Schema{
							Type:         schema.TypeInt,
							Required:     true,
							ValidateFunc: validateIntegerInRange(5, 300),
						},

						"timeout": &schema.Schema{
							Type:         schema.TypeInt,
							Required:     true,
							ValidateFunc: validateIntegerInRange(2, 60),
						},
					},
				},
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

	var elbName string
	if v, ok := d.GetOk("name"); ok {
		elbName = v.(string)
	} else {
		if v, ok := d.GetOk("name_prefix"); ok {
			elbName = resource.PrefixedUniqueId(v.(string))
		} else {
			elbName = resource.PrefixedUniqueId("tf-lb-")
		}
		d.Set("name", elbName)
	}

	tags := tagsFromMapELB(d.Get("tags").(map[string]interface{}))
	// Provision the elb
	elbOpts := &elb.CreateLoadBalancerInput{
		LoadBalancerName: aws.String(elbName),
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
	err = resource.Retry(5*time.Minute, func() *resource.RetryError {
		_, err := elbconn.CreateLoadBalancer(elbOpts)

		if err != nil {
			if awsErr, ok := err.(awserr.Error); ok {
				// Check for IAM SSL Cert error, eventual consistancy issue
				if awsErr.Code() == "CertificateNotFound" {
					return resource.RetryableError(
						fmt.Errorf("[WARN] Error creating ELB Listener with SSL Cert, retrying: %s", err))
				}
			}
			return resource.NonRetryableError(err)
		}
		return nil
	})

	if err != nil {
		return err
	}

	// Assign the elb's unique identifier for use later
	d.SetId(elbName)
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

	d.Set("name", lb.LoadBalancerName)
	d.Set("dns_name", lb.DNSName)
	d.Set("zone_id", lb.CanonicalHostedZoneNameID)

	var scheme bool
	if lb.Scheme != nil {
		scheme = *lb.Scheme == "internal"
	}
	d.Set("internal", scheme)
	d.Set("availability_zones", flattenStringList(lb.AvailabilityZones))
	d.Set("instances", flattenInstances(lb.Instances))
	d.Set("listener", flattenListeners(lb.ListenerDescriptions))
	d.Set("security_groups", flattenStringList(lb.SecurityGroups))
	if lb.SourceSecurityGroup != nil {
		group := lb.SourceSecurityGroup.GroupName
		if lb.SourceSecurityGroup.OwnerAlias != nil && *lb.SourceSecurityGroup.OwnerAlias != "" {
			group = aws.String(*lb.SourceSecurityGroup.OwnerAlias + "/" + *lb.SourceSecurityGroup.GroupName)
		}
		d.Set("source_security_group", group)

		// Manually look up the ELB Security Group ID, since it's not provided
		var elbVpc string
		if lb.VPCId != nil {
			elbVpc = *lb.VPCId
			sgId, err := sourceSGIdByName(meta, *lb.SourceSecurityGroup.GroupName, elbVpc)
			if err != nil {
				return fmt.Errorf("[WARN] Error looking up ELB Security Group ID: %s", err)
			} else {
				d.Set("source_security_group_id", sgId)
			}
		}
	}
	d.Set("subnets", flattenStringList(lb.Subnets))
	if lbAttrs.ConnectionSettings != nil {
		d.Set("idle_timeout", lbAttrs.ConnectionSettings.IdleTimeout)
	}
	d.Set("connection_draining", lbAttrs.ConnectionDraining.Enabled)
	d.Set("connection_draining_timeout", lbAttrs.ConnectionDraining.Timeout)
	d.Set("cross_zone_load_balancing", lbAttrs.CrossZoneLoadBalancing.Enabled)
	if lbAttrs.AccessLog != nil {
		// The AWS API does not allow users to remove access_logs, only disable them.
		// During creation of the ELB, Terraform sets the access_logs to disabled,
		// so there should not be a case where lbAttrs.AccessLog above is nil.

		// Here we do not record the remove value of access_log if:
		// - there is no access_log block in the configuration
		// - the remote access_logs are disabled
		//
		// This indicates there is no access_log in the configuration.
		// - externally added access_logs will be enabled, so we'll detect the drift
		// - locally added access_logs will be in the config, so we'll add to the
		// API/state
		// See https://github.com/hashicorp/terraform/issues/10138
		_, n := d.GetChange("access_logs")
		elbal := lbAttrs.AccessLog
		nl := n.([]interface{})
		if len(nl) == 0 && !*elbal.Enabled {
			elbal = nil
		}
		if err := d.Set("access_logs", flattenAccessLog(elbal)); err != nil {
			return err
		}
	}

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

			log.Printf("[DEBUG] ELB Delete Listeners opts: %s", deleteListenersOpts)
			_, err := elbconn.DeleteLoadBalancerListeners(deleteListenersOpts)
			if err != nil {
				return fmt.Errorf("Failure removing outdated ELB listeners: %s", err)
			}
		}

		if len(add) > 0 {
			createListenersOpts := &elb.CreateLoadBalancerListenersInput{
				LoadBalancerName: aws.String(d.Id()),
				Listeners:        add,
			}

			// Occasionally AWS will error with a 'duplicate listener', without any
			// other listeners on the ELB. Retry here to eliminate that.
			err := resource.Retry(5*time.Minute, func() *resource.RetryError {
				log.Printf("[DEBUG] ELB Create Listeners opts: %s", createListenersOpts)
				if _, err := elbconn.CreateLoadBalancerListeners(createListenersOpts); err != nil {
					if awsErr, ok := err.(awserr.Error); ok {
						if awsErr.Code() == "DuplicateListener" {
							log.Printf("[DEBUG] Duplicate listener found for ELB (%s), retrying", d.Id())
							return resource.RetryableError(awsErr)
						}
						if awsErr.Code() == "CertificateNotFound" && strings.Contains(awsErr.Message(), "Server Certificate not found for the key: arn") {
							log.Printf("[DEBUG] SSL Cert not found for given ARN, retrying")
							return resource.RetryableError(awsErr)
						}
					}

					// Didn't recognize the error, so shouldn't retry.
					return resource.NonRetryableError(err)
				}
				// Successful creation
				return nil
			})
			if err != nil {
				return fmt.Errorf("Failure adding new or updated ELB listeners: %s", err)
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
				return fmt.Errorf("Failure registering instances with ELB: %s", err)
			}
		}
		if len(remove) > 0 {
			deRegisterInstancesOpts := elb.DeregisterInstancesFromLoadBalancerInput{
				LoadBalancerName: aws.String(d.Id()),
				Instances:        remove,
			}

			_, err := elbconn.DeregisterInstancesFromLoadBalancer(&deRegisterInstancesOpts)
			if err != nil {
				return fmt.Errorf("Failure deregistering instances from ELB: %s", err)
			}
		}

		d.SetPartial("instances")
	}

	if d.HasChange("cross_zone_load_balancing") || d.HasChange("idle_timeout") || d.HasChange("access_logs") {
		attrs := elb.ModifyLoadBalancerAttributesInput{
			LoadBalancerName: aws.String(d.Get("name").(string)),
			LoadBalancerAttributes: &elb.LoadBalancerAttributes{
				CrossZoneLoadBalancing: &elb.CrossZoneLoadBalancing{
					Enabled: aws.Bool(d.Get("cross_zone_load_balancing").(bool)),
				},
				ConnectionSettings: &elb.ConnectionSettings{
					IdleTimeout: aws.Int64(int64(d.Get("idle_timeout").(int))),
				},
			},
		}

		logs := d.Get("access_logs").([]interface{})
		if len(logs) == 1 {
			l := logs[0].(map[string]interface{})
			accessLog := &elb.AccessLog{
				Enabled:      aws.Bool(l["enabled"].(bool)),
				EmitInterval: aws.Int64(int64(l["interval"].(int))),
				S3BucketName: aws.String(l["bucket"].(string)),
			}

			if l["bucket_prefix"] != "" {
				accessLog.S3BucketPrefix = aws.String(l["bucket_prefix"].(string))
			}

			attrs.LoadBalancerAttributes.AccessLog = accessLog
		} else if len(logs) == 0 {
			// disable access logs
			attrs.LoadBalancerAttributes.AccessLog = &elb.AccessLog{
				Enabled: aws.Bool(false),
			}
		}

		log.Printf("[DEBUG] ELB Modify Load Balancer Attributes Request: %#v", attrs)
		_, err := elbconn.ModifyLoadBalancerAttributes(&attrs)
		if err != nil {
			return fmt.Errorf("Failure configuring ELB attributes: %s", err)
		}

		d.SetPartial("cross_zone_load_balancing")
		d.SetPartial("idle_timeout")
		d.SetPartial("connection_draining_timeout")
	}

	// We have to do these changes separately from everything else since
	// they have some weird undocumented rules. You can't set the timeout
	// without having connection draining to true, so we set that to true,
	// set the timeout, then reset it to false if requested.
	if d.HasChange("connection_draining") || d.HasChange("connection_draining_timeout") {
		// We do timeout changes first since they require us to set draining
		// to true for a hot second.
		if d.HasChange("connection_draining_timeout") {
			attrs := elb.ModifyLoadBalancerAttributesInput{
				LoadBalancerName: aws.String(d.Get("name").(string)),
				LoadBalancerAttributes: &elb.LoadBalancerAttributes{
					ConnectionDraining: &elb.ConnectionDraining{
						Enabled: aws.Bool(true),
						Timeout: aws.Int64(int64(d.Get("connection_draining_timeout").(int))),
					},
				},
			}

			_, err := elbconn.ModifyLoadBalancerAttributes(&attrs)
			if err != nil {
				return fmt.Errorf("Failure configuring ELB attributes: %s", err)
			}

			d.SetPartial("connection_draining_timeout")
		}

		// Then we always set connection draining even if there is no change.
		// This lets us reset to "false" if requested even with a timeout
		// change.
		attrs := elb.ModifyLoadBalancerAttributesInput{
			LoadBalancerName: aws.String(d.Get("name").(string)),
			LoadBalancerAttributes: &elb.LoadBalancerAttributes{
				ConnectionDraining: &elb.ConnectionDraining{
					Enabled: aws.Bool(d.Get("connection_draining").(bool)),
				},
			},
		}

		_, err := elbconn.ModifyLoadBalancerAttributes(&attrs)
		if err != nil {
			return fmt.Errorf("Failure configuring ELB attributes: %s", err)
		}

		d.SetPartial("connection_draining")
	}

	if d.HasChange("health_check") {
		hc := d.Get("health_check").([]interface{})
		if len(hc) > 0 {
			check := hc[0].(map[string]interface{})
			configureHealthCheckOpts := elb.ConfigureHealthCheckInput{
				LoadBalancerName: aws.String(d.Id()),
				HealthCheck: &elb.HealthCheck{
					HealthyThreshold:   aws.Int64(int64(check["healthy_threshold"].(int))),
					UnhealthyThreshold: aws.Int64(int64(check["unhealthy_threshold"].(int))),
					Interval:           aws.Int64(int64(check["interval"].(int))),
					Target:             aws.String(check["target"].(string)),
					Timeout:            aws.Int64(int64(check["timeout"].(int))),
				},
			}
			_, err := elbconn.ConfigureHealthCheck(&configureHealthCheckOpts)
			if err != nil {
				return fmt.Errorf("Failure configuring health check for ELB: %s", err)
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
			return fmt.Errorf("Failure applying security groups to ELB: %s", err)
		}

		d.SetPartial("security_groups")
	}

	if d.HasChange("availability_zones") {
		o, n := d.GetChange("availability_zones")
		os := o.(*schema.Set)
		ns := n.(*schema.Set)

		removed := expandStringList(os.Difference(ns).List())
		added := expandStringList(ns.Difference(os).List())

		if len(added) > 0 {
			enableOpts := &elb.EnableAvailabilityZonesForLoadBalancerInput{
				LoadBalancerName:  aws.String(d.Id()),
				AvailabilityZones: added,
			}

			log.Printf("[DEBUG] ELB enable availability zones opts: %s", enableOpts)
			_, err := elbconn.EnableAvailabilityZonesForLoadBalancer(enableOpts)
			if err != nil {
				return fmt.Errorf("Failure enabling ELB availability zones: %s", err)
			}
		}

		if len(removed) > 0 {
			disableOpts := &elb.DisableAvailabilityZonesForLoadBalancerInput{
				LoadBalancerName:  aws.String(d.Id()),
				AvailabilityZones: removed,
			}

			log.Printf("[DEBUG] ELB disable availability zones opts: %s", disableOpts)
			_, err := elbconn.DisableAvailabilityZonesForLoadBalancer(disableOpts)
			if err != nil {
				return fmt.Errorf("Failure disabling ELB availability zones: %s", err)
			}
		}

		d.SetPartial("availability_zones")
	}

	if d.HasChange("subnets") {
		o, n := d.GetChange("subnets")
		os := o.(*schema.Set)
		ns := n.(*schema.Set)

		removed := expandStringList(os.Difference(ns).List())
		added := expandStringList(ns.Difference(os).List())

		if len(removed) > 0 {
			detachOpts := &elb.DetachLoadBalancerFromSubnetsInput{
				LoadBalancerName: aws.String(d.Id()),
				Subnets:          removed,
			}

			log.Printf("[DEBUG] ELB detach subnets opts: %s", detachOpts)
			_, err := elbconn.DetachLoadBalancerFromSubnets(detachOpts)
			if err != nil {
				return fmt.Errorf("Failure removing ELB subnets: %s", err)
			}
		}

		if len(added) > 0 {
			attachOpts := &elb.AttachLoadBalancerToSubnetsInput{
				LoadBalancerName: aws.String(d.Id()),
				Subnets:          added,
			}

			log.Printf("[DEBUG] ELB attach subnets opts: %s", attachOpts)
			err := resource.Retry(5*time.Minute, func() *resource.RetryError {
				_, err := elbconn.AttachLoadBalancerToSubnets(attachOpts)
				if err != nil {
					if awsErr, ok := err.(awserr.Error); ok {
						// eventually consistent issue with removing a subnet in AZ1 and
						// immediately adding a new one in the same AZ
						if awsErr.Code() == "InvalidConfigurationRequest" && strings.Contains(awsErr.Message(), "cannot be attached to multiple subnets in the same AZ") {
							log.Printf("[DEBUG] retrying az association")
							return resource.RetryableError(awsErr)
						}
					}
					return resource.NonRetryableError(err)
				}
				return nil
			})
			if err != nil {
				return fmt.Errorf("Failure adding ELB subnets: %s", err)
			}
		}

		d.SetPartial("subnets")
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

	name := d.Get("name").(string)

	err := cleanupELBNetworkInterfaces(meta.(*AWSClient).ec2conn, name)
	if err != nil {
		log.Printf("[WARN] Failed to cleanup ENIs for ELB %q: %#v", name, err)
	}

	return nil
}

func resourceAwsElbListenerHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})
	buf.WriteString(fmt.Sprintf("%d-", m["instance_port"].(int)))
	buf.WriteString(fmt.Sprintf("%s-",
		strings.ToLower(m["instance_protocol"].(string))))
	buf.WriteString(fmt.Sprintf("%d-", m["lb_port"].(int)))
	buf.WriteString(fmt.Sprintf("%s-",
		strings.ToLower(m["lb_protocol"].(string))))

	if v, ok := m["ssl_certificate_id"]; ok {
		buf.WriteString(fmt.Sprintf("%s-", v.(string)))
	}

	return hashcode.String(buf.String())
}

func isLoadBalancerNotFound(err error) bool {
	elberr, ok := err.(awserr.Error)
	return ok && elberr.Code() == "LoadBalancerNotFound"
}

func sourceSGIdByName(meta interface{}, sg, vpcId string) (string, error) {
	conn := meta.(*AWSClient).ec2conn
	var filters []*ec2.Filter
	var sgFilterName, sgFilterVPCID *ec2.Filter
	sgFilterName = &ec2.Filter{
		Name:   aws.String("group-name"),
		Values: []*string{aws.String(sg)},
	}

	if vpcId != "" {
		sgFilterVPCID = &ec2.Filter{
			Name:   aws.String("vpc-id"),
			Values: []*string{aws.String(vpcId)},
		}
	}

	filters = append(filters, sgFilterName)

	if sgFilterVPCID != nil {
		filters = append(filters, sgFilterVPCID)
	}

	req := &ec2.DescribeSecurityGroupsInput{
		Filters: filters,
	}
	resp, err := conn.DescribeSecurityGroups(req)
	if err != nil {
		if ec2err, ok := err.(awserr.Error); ok {
			if ec2err.Code() == "InvalidSecurityGroupID.NotFound" ||
				ec2err.Code() == "InvalidGroup.NotFound" {
				resp = nil
				err = nil
			}
		}

		if err != nil {
			log.Printf("Error on ELB SG look up: %s", err)
			return "", err
		}
	}

	if resp == nil || len(resp.SecurityGroups) == 0 {
		return "", fmt.Errorf("No security groups found for name %s and vpc id %s", sg, vpcId)
	}

	group := resp.SecurityGroups[0]
	return *group.GroupId, nil
}

func validateAccessLogsInterval(v interface{}, k string) (ws []string, errors []error) {
	value := v.(int)

	// Check if the value is either 5 or 60 (minutes).
	if value != 5 && value != 60 {
		errors = append(errors, fmt.Errorf(
			"%q contains an invalid Access Logs interval \"%d\". "+
				"Valid intervals are either 5 or 60 (minutes).",
			k, value))
	}
	return
}

func validateHeathCheckTarget(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)

	// Parse the Health Check target value.
	matches := regexp.MustCompile(`\A(\w+):(\d+)(.+)?\z`).FindStringSubmatch(value)

	// Check if the value contains a valid target.
	if matches == nil || len(matches) < 1 {
		errors = append(errors, fmt.Errorf(
			"%q contains an invalid Health Check: %s",
			k, value))

		// Invalid target? Return immediately,
		// there is no need to collect other
		// errors.
		return
	}

	// Check if the value contains a valid protocol.
	if !isValidProtocol(matches[1]) {
		errors = append(errors, fmt.Errorf(
			"%q contains an invalid Health Check protocol %q. "+
				"Valid protocols are either %q, %q, %q, or %q.",
			k, matches[1], "TCP", "SSL", "HTTP", "HTTPS"))
	}

	// Check if the value contains a valid port range.
	port, _ := strconv.Atoi(matches[2])
	if port < 1 || port > 65535 {
		errors = append(errors, fmt.Errorf(
			"%q contains an invalid Health Check target port \"%d\". "+
				"Valid port is in the range from 1 to 65535 inclusive.",
			k, port))
	}

	switch strings.ToLower(matches[1]) {
	case "tcp", "ssl":
		// Check if value is in the form <PROTOCOL>:<PORT> for TCP and/or SSL.
		if matches[3] != "" {
			errors = append(errors, fmt.Errorf(
				"%q cannot contain a path in the Health Check target: %s",
				k, value))
		}
		break
	case "http", "https":
		// Check if value is in the form <PROTOCOL>:<PORT>/<PATH> for HTTP and/or HTTPS.
		if matches[3] == "" {
			errors = append(errors, fmt.Errorf(
				"%q must contain a path in the Health Check target: %s",
				k, value))
		}

		// Cannot be longer than 1024 multibyte characters.
		if len([]rune(matches[3])) > 1024 {
			errors = append(errors, fmt.Errorf("%q cannot contain a path longer "+
				"than 1024 characters in the Health Check target: %s",
				k, value))
		}
		break
	}

	return
}

func validateListenerProtocol(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)

	if !isValidProtocol(value) {
		errors = append(errors, fmt.Errorf(
			"%q contains an invalid Listener protocol %q. "+
				"Valid protocols are either %q, %q, %q, or %q.",
			k, value, "TCP", "SSL", "HTTP", "HTTPS"))
	}
	return
}

func isValidProtocol(s string) bool {
	if s == "" {
		return false
	}
	s = strings.ToLower(s)

	validProtocols := map[string]bool{
		"http":  true,
		"https": true,
		"ssl":   true,
		"tcp":   true,
	}

	if _, ok := validProtocols[s]; !ok {
		return false
	}

	return true
}

// ELB automatically creates ENI(s) on creation
// but the cleanup is asynchronous and may take time
// which then blocks IGW, SG or VPC on deletion
// So we make the cleanup "synchronous" here
func cleanupELBNetworkInterfaces(conn *ec2.EC2, name string) error {
	out, err := conn.DescribeNetworkInterfaces(&ec2.DescribeNetworkInterfacesInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("attachment.instance-owner-id"),
				Values: []*string{aws.String("amazon-elb")},
			},
			{
				Name:   aws.String("description"),
				Values: []*string{aws.String("ELB " + name)},
			},
		},
	})
	if err != nil {
		return err
	}

	log.Printf("[DEBUG] Found %d ENIs to cleanup for ELB %q",
		len(out.NetworkInterfaces), name)

	if len(out.NetworkInterfaces) == 0 {
		// Nothing to cleanup
		return nil
	}

	err = detachNetworkInterfaces(conn, out.NetworkInterfaces)
	if err != nil {
		return err
	}

	err = deleteNetworkInterfaces(conn, out.NetworkInterfaces)
	if err != nil {
		return err
	}

	return nil
}

func detachNetworkInterfaces(conn *ec2.EC2, nis []*ec2.NetworkInterface) error {
	log.Printf("[DEBUG] Trying to detach %d leftover ENIs", len(nis))
	for _, ni := range nis {
		if ni.Attachment == nil {
			log.Printf("[DEBUG] ENI %s is already detached", *ni.NetworkInterfaceId)
			continue
		}
		_, err := conn.DetachNetworkInterface(&ec2.DetachNetworkInterfaceInput{
			AttachmentId: ni.Attachment.AttachmentId,
			Force:        aws.Bool(true),
		})
		if err != nil {
			awsErr, ok := err.(awserr.Error)
			if ok && awsErr.Code() == "InvalidAttachmentID.NotFound" {
				log.Printf("[DEBUG] ENI %s is already detached", *ni.NetworkInterfaceId)
				continue
			}
			return err
		}

		log.Printf("[DEBUG] Waiting for ENI (%s) to become detached", *ni.NetworkInterfaceId)
		stateConf := &resource.StateChangeConf{
			Pending: []string{"true"},
			Target:  []string{"false"},
			Refresh: networkInterfaceAttachmentRefreshFunc(conn, *ni.NetworkInterfaceId),
			Timeout: 10 * time.Minute,
		}

		if _, err := stateConf.WaitForState(); err != nil {
			awsErr, ok := err.(awserr.Error)
			if ok && awsErr.Code() == "InvalidNetworkInterfaceID.NotFound" {
				continue
			}
			return fmt.Errorf(
				"Error waiting for ENI (%s) to become detached: %s", *ni.NetworkInterfaceId, err)
		}
	}
	return nil
}

func deleteNetworkInterfaces(conn *ec2.EC2, nis []*ec2.NetworkInterface) error {
	log.Printf("[DEBUG] Trying to delete %d leftover ENIs", len(nis))
	for _, ni := range nis {
		_, err := conn.DeleteNetworkInterface(&ec2.DeleteNetworkInterfaceInput{
			NetworkInterfaceId: ni.NetworkInterfaceId,
		})
		if err != nil {
			awsErr, ok := err.(awserr.Error)
			if ok && awsErr.Code() == "InvalidNetworkInterfaceID.NotFound" {
				log.Printf("[DEBUG] ENI %s is already deleted", *ni.NetworkInterfaceId)
				continue
			}
			return err
		}
	}
	return nil
}
