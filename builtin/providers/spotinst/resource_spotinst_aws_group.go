package spotinst

import (
	"bytes"
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/spotinst/spotinst-sdk-go/spotinst"
)

func resourceSpotinstAwsGroup() *schema.Resource {
	return &schema.Resource{
		Create: resourceSpotinstAwsGroupCreate,
		Read:   resourceSpotinstAwsGroupRead,
		Update: resourceSpotinstAwsGroupUpdate,
		Delete: resourceSpotinstAwsGroupDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: false,
			},

			"description": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: false,
			},

			"capacity": &schema.Schema{
				Type:     schema.TypeSet,
				Required: true,
				ForceNew: false,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"minimum": &schema.Schema{
							Type:     schema.TypeInt,
							Optional: true,
							ForceNew: false,
						},

						"maximum": &schema.Schema{
							Type:     schema.TypeInt,
							Optional: true,
							ForceNew: false,
						},

						"target": &schema.Schema{
							Type:     schema.TypeInt,
							Required: true,
							ForceNew: false,
						},
					},
				},
				Set: func(v interface{}) int {
					var buf bytes.Buffer
					m := v.(map[string]interface{})
					buf.WriteString(fmt.Sprintf("%d-", m["minimum"].(int)))
					buf.WriteString(fmt.Sprintf("%d-", m["maximum"].(int)))
					buf.WriteString(fmt.Sprintf("%d-", m["target"].(int)))
					return hashcode.String(buf.String())
				},
			},

			"strategy": &schema.Schema{
				Type:     schema.TypeSet,
				Required: true,
				ForceNew: false,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"risk": &schema.Schema{
							Type:     schema.TypeFloat,
							Optional: true,
							ForceNew: false,
						},

						"availability_vs_cost": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							Default:  "balanced",
							ForceNew: false,
						},

						"ondemand_count": &schema.Schema{
							Type:     schema.TypeInt,
							Optional: true,
							ForceNew: false,
						},

						"draining_timeout": &schema.Schema{
							Type:     schema.TypeInt,
							Optional: true,
							ForceNew: false,
						},
					},
				},
			},

			"scheduled_task": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: false,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"task_type": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: false,
						},

						"frequency": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: false,
						},

						"cron_expression": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: false,
						},

						"scale_target_capacity": &schema.Schema{
							Type:     schema.TypeInt,
							Optional: true,
							ForceNew: false,
						},

						"scale_min_capacity": &schema.Schema{
							Type:     schema.TypeInt,
							Optional: true,
							ForceNew: false,
						},

						"scale_max_capacity": &schema.Schema{
							Type:     schema.TypeInt,
							Optional: true,
							ForceNew: false,
						},
					},
				},
			},

			"product": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"instance_types": &schema.Schema{
				Type:     schema.TypeSet,
				Required: true,
				ForceNew: false,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"ondemand": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
							ForceNew: false,
						},

						"spot": &schema.Schema{
							Type:     schema.TypeList,
							Required: true,
							ForceNew: false,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},
					},
				},
			},

			"signal": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: false,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
							ForceNew: false,
							StateFunc: func(v interface{}) string {
								value := v.(string)
								return strings.ToUpper(value)
							},
						},
					},
				},
			},

			"availability_zone": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: false,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
							ForceNew: false,
						},

						"subnet_id": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: false,
						},
					},
				},
			},

			"launch_specification": &schema.Schema{
				Type:     schema.TypeSet,
				Required: true,
				ForceNew: false,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"load_balancer_names": &schema.Schema{
							Type:     schema.TypeSet,
							Optional: true,
							ForceNew: false,
							Elem:     &schema.Schema{Type: schema.TypeString},
							Set:      schema.HashString,
						},

						"monitoring": &schema.Schema{
							Type:     schema.TypeBool,
							Optional: true,
							ForceNew: false,
							Default:  false,
						},

						"image_id": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
							ForceNew: false,
						},

						"key_pair": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
							ForceNew: false,
						},

						"health_check_type": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: false,
						},

						"health_check_grace_period": &schema.Schema{
							Type:     schema.TypeInt,
							Optional: true,
							ForceNew: false,
						},

						"security_group_ids": &schema.Schema{
							Type:     schema.TypeSet,
							Required: true,
							ForceNew: false,
							Elem:     &schema.Schema{Type: schema.TypeString},
							Set:      schema.HashString,
						},

						"user_data": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: false,
							StateFunc: func(v interface{}) string {
								switch v.(type) {
								case string:
									hash := sha1.Sum([]byte(v.(string)))
									return hex.EncodeToString(hash[:])
								default:
									return ""
								}
							},
						},

						"iam_role": &schema.Schema{
							Type:       schema.TypeString,
							Optional:   true,
							ForceNew:   false,
							Deprecated: "Attribute iam_role is deprecated. Use iam_instance_profile instead",
						},

						"iam_instance_profile": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: false,
						},
					},
				},
			},

			"elastic_ips": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: false,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set: func(v interface{}) int {
					return hashcode.String(v.(string))
				},
			},

			"tags": &schema.Schema{
				Type:     schema.TypeMap,
				Optional: true,
				ForceNew: false,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},

			"ebs_block_device": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: false,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"delete_on_termination": &schema.Schema{
							Type:     schema.TypeBool,
							Optional: true,
							Default:  true,
							ForceNew: false,
						},

						"device_name": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
							ForceNew: false,
						},

						"encrypted": &schema.Schema{
							Type:     schema.TypeBool,
							Optional: true,
							ForceNew: false,
						},

						"iops": &schema.Schema{
							Type:     schema.TypeInt,
							Optional: true,
							ForceNew: false,
						},

						"snapshot_id": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: false,
						},

						"volume_size": &schema.Schema{
							Type:     schema.TypeInt,
							Optional: true,
							ForceNew: false,
						},

						"volume_type": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: false,
						},
					},
				},
				Set: func(v interface{}) int {
					var buf bytes.Buffer
					m := v.(map[string]interface{})
					buf.WriteString(fmt.Sprintf("%s-", m["device_name"].(string)))
					buf.WriteString(fmt.Sprintf("%s-", m["snapshot_id"].(string)))
					return hashcode.String(buf.String())
				},
			},

			"ephemeral_block_device": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: false,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"device_name": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
							ForceNew: false,
						},

						"virtual_name": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
							ForceNew: false,
						},
					},
				},
				Set: func(v interface{}) int {
					var buf bytes.Buffer
					m := v.(map[string]interface{})
					buf.WriteString(fmt.Sprintf("%s-", m["device_name"].(string)))
					buf.WriteString(fmt.Sprintf("%s-", m["virtual_name"].(string)))
					return hashcode.String(buf.String())
				},
			},

			"network_interface": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: false,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"description": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
							ForceNew: false,
						},

						"device_index": &schema.Schema{
							Type:     schema.TypeInt,
							Required: true,
							ForceNew: false,
						},

						"secondary_private_ip_address_count": &schema.Schema{
							Type:     schema.TypeInt,
							Optional: true,
							ForceNew: false,
						},

						"associate_public_ip_address": &schema.Schema{
							Type:     schema.TypeBool,
							Optional: true,
							ForceNew: false,
						},

						"delete_on_termination": &schema.Schema{
							Type:     schema.TypeBool,
							Optional: true,
							ForceNew: false,
						},

						"security_group_ids": &schema.Schema{
							Type:     schema.TypeSet,
							Optional: true,
							ForceNew: false,
							Elem:     &schema.Schema{Type: schema.TypeString},
							Set:      schema.HashString,
						},

						"network_interface_id": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: false,
						},

						"private_ip_address": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: false,
						},

						"subnet_id": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: false,
						},
					},
				},
			},

			"scaling_up_policy": scalingPolicySchema(),

			"scaling_down_policy": scalingPolicySchema(),

			"rancher_integration": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: false,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"master_host": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
							ForceNew: false,
						},

						"access_key": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
							ForceNew: false,
						},

						"secret_key": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
							ForceNew: false,
						},
					},
				},
			},

			"elastic_beanstalk_integration": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: false,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"environment_id": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
							ForceNew: false,
						},
					},
				},
			},

			"ec2_container_service_integration": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: false,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"cluster_name": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
							ForceNew: false,
						},
					},
				},
			},

			"nirmata_integration": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: false,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"api_key": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
							ForceNew: false,
						},
					},
				},
			},
		},
	}
}

func scalingPolicySchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeSet,
		Optional: true,
		ForceNew: false,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"policy_name": &schema.Schema{
					Type:     schema.TypeString,
					Required: true,
					ForceNew: false,
				},

				"metric_name": &schema.Schema{
					Type:     schema.TypeString,
					Required: true,
					ForceNew: false,
				},

				"statistic": &schema.Schema{
					Type:     schema.TypeString,
					Required: true,
					ForceNew: false,
				},

				"unit": &schema.Schema{
					Type:     schema.TypeString,
					Required: true,
					ForceNew: false,
				},

				"threshold": &schema.Schema{
					Type:     schema.TypeFloat,
					Required: true,
					ForceNew: false,
				},

				"adjustment": &schema.Schema{
					Type:     schema.TypeInt,
					Optional: true,
					ForceNew: false,
				},

				"min_target_capacity": &schema.Schema{
					Type:     schema.TypeInt,
					Optional: true,
					ForceNew: false,
				},

				"max_target_capacity": &schema.Schema{
					Type:     schema.TypeInt,
					Optional: true,
					ForceNew: false,
				},

				"namespace": &schema.Schema{
					Type:     schema.TypeString,
					Required: true,
					ForceNew: false,
				},

				"evaluation_periods": &schema.Schema{
					Type:     schema.TypeInt,
					Required: true,
					ForceNew: false,
				},

				"period": &schema.Schema{
					Type:     schema.TypeInt,
					Required: true,
					ForceNew: false,
				},

				"cooldown": &schema.Schema{
					Type:     schema.TypeInt,
					Required: true,
					ForceNew: false,
				},

				"dimensions": &schema.Schema{
					Type:     schema.TypeMap,
					Optional: true,
					ForceNew: false,
					Elem:     &schema.Schema{Type: schema.TypeString},
				},
			},
		},
	}
}

func resourceSpotinstAwsGroupCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*spotinst.Client)
	newAwsGroup, err := buildAwsGroupOpts(d, meta)
	if err != nil {
		return err
	}
	log.Printf("[DEBUG] AwsGroup create configuration: %#v\n", newAwsGroup)
	res, _, err := client.AwsGroup.Create(newAwsGroup)
	if err != nil {
		return fmt.Errorf("[ERROR] Error creating group: %s", err)
	}
	d.SetId(*res[0].ID)
	log.Printf("[INFO] AwsGroup created successfully: %s\n", d.Id())
	return resourceSpotinstAwsGroupRead(d, meta)
}

func resourceSpotinstAwsGroupRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*spotinst.Client)
	groups, _, err := client.AwsGroup.Get(d.Id())
	if err != nil {
		if serr, ok := err.(*spotinst.ErrorResponse); ok {
			if serr.Response.StatusCode == 400 {
				d.SetId("")
				return nil
			} else {
				return fmt.Errorf("[ERROR] Error retrieving group: %s", err)
			}
		} else {
			return fmt.Errorf("[ERROR] Error retrieving group: %s", err)
		}
	}
	if len(groups) == 0 {
		return fmt.Errorf("[ERROR] No matching group %s", d.Id())
	} else if len(groups) > 1 {
		return fmt.Errorf("[ERROR] Got %d results, only one is allowed", len(groups))
	} else if g := groups[0]; g != nil {
		d.Set("name", g.Name)
		d.Set("description", g.Description)
		d.Set("product", g.Compute.Product)

		// Set the capacity.
		capacity := make([]map[string]interface{}, 0, 1)
		capacity = append(capacity, map[string]interface{}{
			"target":  g.Capacity.Target,
			"minimum": g.Capacity.Minimum,
			"maximum": g.Capacity.Maximum,
		})
		d.Set("capacity", capacity)

		// Set the strategy.
		strategy := make([]map[string]interface{}, 0, 1)
		strategy = append(strategy, map[string]interface{}{
			"risk":                 g.Strategy.Risk,
			"ondemand_count":       g.Strategy.OnDemandCount,
			"availability_vs_cost": g.Strategy.AvailabilityVsCost,
			"draining_timeout":     g.Strategy.DrainingTimeout,
		})
		d.Set("strategy", strategy)

		// Set the launch specification.
		lspec := make([]map[string]interface{}, 0, 1)
		l := map[string]interface{}{
			"health_check_grace_period": g.Compute.LaunchSpecification.HealthCheckGracePeriod,
			"health_check_type":         g.Compute.LaunchSpecification.HealthCheckType,
			"image_id":                  g.Compute.LaunchSpecification.ImageID,
			"key_pair":                  g.Compute.LaunchSpecification.KeyPair,
			"load_balancer_names":       g.Compute.LaunchSpecification.LoadBalancerNames,
			"monitoring":                g.Compute.LaunchSpecification.Monitoring,
			"security_group_ids":        g.Compute.LaunchSpecification.SecurityGroupIDs,
			"user_data":                 g.Compute.LaunchSpecification.UserData,
		}
		if g.Compute.LaunchSpecification.IamInstanceProfile != nil {
			l["iam_instance_profile"] = g.Compute.LaunchSpecification.IamInstanceProfile.Arn
		}
		lspec = append(lspec, l)
		d.Set("launch_specification", lspec)

		// Set the availability zones.
		zones := make([]map[string]interface{}, 0, len(g.Compute.AvailabilityZones))
		for _, z := range g.Compute.AvailabilityZones {
			zones = append(zones, map[string]interface{}{
				"name":      z.Name,
				"subnet_id": z.SubnetID,
			})
		}
		d.Set("availability_zone", zones)

		// Set the signals.
		signals := make([]map[string]interface{}, 0, len(g.Strategy.Signals))
		for _, s := range g.Strategy.Signals {
			signals = append(signals, map[string]interface{}{
				"name": s.Name,
			})
		}
		d.Set("signal", signals)

		// Set the scheduled tasks.
		tasks := make([]map[string]interface{}, 0, len(g.Scheduling.Tasks))
		for _, t := range g.Scheduling.Tasks {
			tasks = append(tasks, map[string]interface{}{
				"task_type":             t.TaskType,
				"cron_expression":       t.CronExpression,
				"frequency":             t.Frequency,
				"scale_target_capacity": t.ScaleTargetCapacity,
				"scale_min_capacity":    t.ScaleMinCapacity,
				"scale_max_capacity":    t.ScaleMaxCapacity,
			})
		}
		d.Set("scheduled_task", tasks)

		// Set the tags.
		d.Set("tags", tagsToMap(g.Compute.LaunchSpecification.Tags))

		// Set the elastic IPs.
		d.Set("elastic_ips", g.Compute.ElasticIPs)

		// Set the scaling up policies.
		up := make([]map[string]interface{}, 0, len(g.Scaling.Up))
		for _, p := range g.Scaling.Up {
			up = append(up, map[string]interface{}{
				"adjustment":          p.Adjustment,
				"cooldown":            p.Cooldown,
				"dimensions":          p.Dimensions,
				"evaluation_periods":  p.EvaluationPeriods,
				"max_target_capacity": p.MaxTargetCapacity,
				"metric_name":         p.MetricName,
				"min_target_capacity": p.MinTargetCapacity,
				"namespace":           p.Namespace,
				"period":              p.Period,
				"policy_name":         p.PolicyName,
				"statistic":           p.Statistic,
				"threshold":           p.Threshold,
				"unit":                p.Unit,
			})
		}
		d.Set("scaling_up_policy", up)

		// Set the scaling down policies.
		down := make([]map[string]interface{}, 0, len(g.Scaling.Down))
		for _, p := range g.Scaling.Down {
			down = append(down, map[string]interface{}{
				"adjustment":          p.Adjustment,
				"cooldown":            p.Cooldown,
				"dimensions":          p.Dimensions,
				"evaluation_periods":  p.EvaluationPeriods,
				"max_target_capacity": p.MaxTargetCapacity,
				"metric_name":         p.MetricName,
				"min_target_capacity": p.MinTargetCapacity,
				"namespace":           p.Namespace,
				"period":              p.Period,
				"policy_name":         p.PolicyName,
				"statistic":           p.Statistic,
				"threshold":           p.Threshold,
				"unit":                p.Unit,
			})
		}
		d.Set("scaling_down_policy", down)

		// Set the network interfaces.
		interfaces := make([]map[string]interface{}, 0, len(g.Compute.LaunchSpecification.NetworkInterfaces))
		for _, i := range g.Compute.LaunchSpecification.NetworkInterfaces {
			interfaces = append(interfaces, map[string]interface{}{
				"associate_public_ip_address":        i.AssociatePublicIPAddress,
				"delete_on_termination":              i.DeleteOnTermination,
				"description":                        i.Description,
				"device_index":                       i.DeviceIndex,
				"network_interface_id":               i.ID,
				"private_ip_address":                 i.PrivateIPAddress,
				"secondary_private_ip_address_count": i.SecondaryPrivateIPAddressCount,
				"security_group_ids":                 i.SecurityGroupsIDs,
				"subnet_id":                          i.SubnetID,
			})
		}
		d.Set("network_interface", interfaces)

		// Set the EBS block devices.
		ebsDevices := make([]map[string]interface{}, 0, len(g.Compute.LaunchSpecification.BlockDevices))
		for _, d := range g.Compute.LaunchSpecification.BlockDevices {
			if d.EBS != nil {
				ebsDevices = append(ebsDevices, map[string]interface{}{
					"device_name":           d.DeviceName,
					"delete_on_termination": d.EBS.DeleteOnTermination,
					"encrypted":             d.EBS.Encrypted,
					"iops":                  d.EBS.IOPS,
					"snapshot_id":           d.EBS.SnapshotID,
					"volume_size":           d.EBS.VolumeSize,
					"volume_type":           d.EBS.VolumeType,
				})
			}
		}
		d.Set("ebs_block_device", ebsDevices)

		// Set the Ephemeral block devices.
		ephemeralDevices := make([]map[string]interface{}, 0, len(g.Compute.LaunchSpecification.BlockDevices))
		for _, d := range g.Compute.LaunchSpecification.BlockDevices {
			if d.EBS == nil {
				ephemeralDevices = append(ephemeralDevices, map[string]interface{}{
					"device_name":  d.DeviceName,
					"virtual_name": d.VirtualName,
				})
			}
		}
		d.Set("ephemeral_block_device", ephemeralDevices)

		// Set the Rancher integration.
		rancher := make([]map[string]interface{}, 0, 1)
		if g.Integration.Rancher != nil {
			rancher = append(rancher, map[string]interface{}{
				"master_host": g.Integration.Rancher.MasterHost,
				"access_key":  g.Integration.Rancher.AccessKey,
				"secret_leu":  g.Integration.Rancher.SecretKey,
			})
		}
		d.Set("rancher_integration", rancher)

		// Set the Elastic Beanstalk integration.
		beanstalk := make([]map[string]interface{}, 0, 1)
		if g.Integration.ElasticBeanstalk != nil {
			beanstalk = append(beanstalk, map[string]interface{}{
				"environment_id": g.Integration.ElasticBeanstalk.EnvironmentID,
			})
		}
		d.Set("elastic_beanstalk_integration", beanstalk)

		// Set the EC2 Container Service integration.
		ecs := make([]map[string]interface{}, 0, 1)
		if g.Integration.EC2ContainerService != nil {
			ecs = append(ecs, map[string]interface{}{
				"cluster_name": g.Integration.EC2ContainerService.ClusterName,
			})
		}
		d.Set("ec2_container_service_integration", ecs)

		// Set the Nirmata integration.
		nirmata := make([]map[string]interface{}, 0, 1)
		if g.Integration.Nirmata != nil {
			nirmata = append(nirmata, map[string]interface{}{
				"api_key": g.Integration.Nirmata.APIKey,
			})
		}
		d.Set("nirmata_integration", nirmata)
	} else {
		d.SetId("")
		return nil
	}
	return nil
}

func resourceSpotinstAwsGroupUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*spotinst.Client)
	group := &spotinst.AwsGroup{ID: spotinst.String(d.Id())}
	hasChange := false

	if d.HasChange("name") {
		group.Name = spotinst.String(d.Get("name").(string))
		hasChange = true
	}

	if d.HasChange("description") {
		group.Description = spotinst.String(d.Get("description").(string))
		hasChange = true
	}

	if d.HasChange("capacity") {
		if v, ok := d.GetOk("capacity"); ok {
			if capacity, err := expandAwsGroupCapacity(v); err != nil {
				return err
			} else {
				group.Capacity = capacity
				hasChange = true
			}
		}
	}

	if d.HasChange("strategy") {
		if v, ok := d.GetOk("strategy"); ok {
			if strategy, err := expandAwsGroupStrategy(v); err != nil {
				return err
			} else {
				group.Strategy = strategy
				hasChange = true
			}
		}
	}

	if d.HasChange("launch_specification") {
		if v, ok := d.GetOk("launch_specification"); ok {
			if lc, err := expandAwsGroupLaunchSpecification(v); err != nil {
				return err
			} else {
				if group.Compute == nil {
					group.Compute = &spotinst.AwsGroupCompute{}
				}
				group.Compute.LaunchSpecification = lc
				hasChange = true
			}
		}
	}

	if d.HasChange("network_interface") {
		if v, ok := d.GetOk("network_interface"); ok {
			if interfaces, err := expandAwsGroupNetworkInterfaces(v); err != nil {
				return err
			} else {
				if group.Compute == nil {
					group.Compute = &spotinst.AwsGroupCompute{}
				} else if group.Compute.LaunchSpecification == nil {
					group.Compute.LaunchSpecification = &spotinst.AwsGroupComputeLaunchSpecification{}
				}
				group.Compute.LaunchSpecification.NetworkInterfaces = interfaces
				hasChange = true
			}
		}
	}

	if d.HasChange("availability_zone") {
		if v, ok := d.GetOk("availability_zone"); ok {
			if zones, err := expandAwsGroupAvailabilityZones(v); err != nil {
				return err
			} else {
				if group.Compute == nil {
					group.Compute = &spotinst.AwsGroupCompute{}
				}
				group.Compute.AvailabilityZones = zones
				hasChange = true
			}
		}
	}

	if d.HasChange("signal") {
		if v, ok := d.GetOk("signal"); ok {
			if signals, err := expandAwsGroupSignals(v); err != nil {
				return err
			} else {
				if group.Strategy == nil {
					group.Strategy = &spotinst.AwsGroupStrategy{}
				}
				group.Strategy.Signals = signals
				hasChange = true
			}
		}
	}

	if d.HasChange("instance_types") {
		if v, ok := d.GetOk("instance_types"); ok {
			if types, err := expandAwsGroupInstanceTypes(v); err != nil {
				return err
			} else {
				if group.Compute == nil {
					group.Compute = &spotinst.AwsGroupCompute{}
				}
				group.Compute.InstanceTypes = types
				hasChange = true
			}
		}
	}

	if d.HasChange("tags") {
		if v, ok := d.GetOk("tags"); ok {
			if tags, err := expandAwsGroupTags(v); err != nil {
				return err
			} else {
				if group.Compute == nil {
					group.Compute = &spotinst.AwsGroupCompute{}
				} else if group.Compute.LaunchSpecification == nil {
					group.Compute.LaunchSpecification = &spotinst.AwsGroupComputeLaunchSpecification{}
				}
				group.Compute.LaunchSpecification.Tags = tags
				hasChange = true
			}
		}
	}

	if d.HasChange("elastic_ips") {
		if v, ok := d.GetOk("elastic_ips"); ok {
			if eips, err := expandAwsGroupElasticIPs(v); err != nil {
				return err
			} else {
				if group.Compute == nil {
					group.Compute = &spotinst.AwsGroupCompute{}
				}
				group.Compute.ElasticIPs = eips
				hasChange = true
			}
		}
	}

	if d.HasChange("scheduled_task") {
		if v, ok := d.GetOk("scheduled_task"); ok {
			if tasks, err := expandAwsGroupScheduledTasks(v); err != nil {
				return err
			} else {
				if group.Scheduling == nil {
					group.Scheduling = &spotinst.AwsGroupScheduling{}
				}
				group.Scheduling.Tasks = tasks
				hasChange = true
			}
		}
	}

	if d.HasChange("scaling_up_policy") {
		if v, ok := d.GetOk("scaling_up_policy"); ok {
			if policies, err := expandAwsGroupScalingPolicies(v); err != nil {
				return err
			} else {
				if group.Scaling == nil {
					group.Scaling = &spotinst.AwsGroupScaling{}
				}
				group.Scaling.Up = policies
				hasChange = true
			}
		}
	}

	if d.HasChange("scaling_down_policy") {
		if v, ok := d.GetOk("scaling_down_policy"); ok {
			if policies, err := expandAwsGroupScalingPolicies(v); err != nil {
				return err
			} else {
				if group.Scaling == nil {
					group.Scaling = &spotinst.AwsGroupScaling{}
				}
				group.Scaling.Down = policies
				hasChange = true
			}
		}
	}

	if d.HasChange("rancher_integration") {
		if v, ok := d.GetOk("rancher_integration"); ok {
			if integration, err := expandAwsGroupRancherIntegration(v); err != nil {
				return err
			} else {
				if group.Integration == nil {
					group.Integration = &spotinst.AwsGroupIntegration{}
				}
				group.Integration.Rancher = integration
				hasChange = true
			}
		}
	}

	if d.HasChange("elastic_eanstalk_integration") {
		if v, ok := d.GetOk("elastic_beanstalk_integration"); ok {
			if integration, err := expandAwsGroupElasticBeanstalkIntegration(v); err != nil {
				return err
			} else {
				if group.Integration == nil {
					group.Integration = &spotinst.AwsGroupIntegration{}
				}
				group.Integration.ElasticBeanstalk = integration
				hasChange = true
			}
		}
	}

	if d.HasChange("ec2_container_service_integration") {
		if v, ok := d.GetOk("ec2_container_service_integration"); ok {
			if integration, err := expandAwsGroupEC2ContainerServiceIntegration(v); err != nil {
				return err
			} else {
				if group.Integration == nil {
					group.Integration = &spotinst.AwsGroupIntegration{}
				}
				group.Integration.EC2ContainerService = integration
				hasChange = true
			}
		}
	}

	if d.HasChange("nirmata_integration") {
		if v, ok := d.GetOk("nirmata_integration"); ok {
			if integration, err := expandAwsGroupNirmataIntegration(v); err != nil {
				return err
			} else {
				if group.Integration == nil {
					group.Integration = &spotinst.AwsGroupIntegration{}
				}
				group.Integration.Nirmata = integration
				hasChange = true
			}
		}
	}

	if hasChange {
		log.Printf("[DEBUG] AwsGroup update configuration: %#v\n", group)
		_, _, err := client.AwsGroup.Update(group)
		if err != nil {
			return fmt.Errorf("[ERROR] Error updating group: %s", err)
		}
	}

	return resourceSpotinstAwsGroupRead(d, meta)
}

func resourceSpotinstAwsGroupDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*spotinst.Client)
	log.Printf("[INFO] Deleting group: %s\n", d.Id())
	group := &spotinst.AwsGroup{ID: spotinst.String(d.Id())}
	_, err := client.AwsGroup.Delete(group)
	if err != nil {
		return fmt.Errorf("[ERROR] Error deleting group: %s", err)
	}
	return nil
}

// buildAwsGroupOpts builds the Spotinst AWS Group options.
func buildAwsGroupOpts(d *schema.ResourceData, meta interface{}) (*spotinst.AwsGroup, error) {
	group := &spotinst.AwsGroup{
		Name:        spotinst.String(d.Get("name").(string)),
		Description: spotinst.String(d.Get("description").(string)),
		Scaling:     &spotinst.AwsGroupScaling{},
		Scheduling:  &spotinst.AwsGroupScheduling{},
		Integration: &spotinst.AwsGroupIntegration{},
		Compute: &spotinst.AwsGroupCompute{
			Product:             spotinst.String(d.Get("product").(string)),
			LaunchSpecification: &spotinst.AwsGroupComputeLaunchSpecification{},
		},
	}

	if v, ok := d.GetOk("capacity"); ok {
		if capacity, err := expandAwsGroupCapacity(v); err != nil {
			return nil, err
		} else {
			group.Capacity = capacity
		}
	}

	if v, ok := d.GetOk("strategy"); ok {
		if strategy, err := expandAwsGroupStrategy(v); err != nil {
			return nil, err
		} else {
			group.Strategy = strategy
		}
	}

	if v, ok := d.GetOk("scaling_up_policy"); ok {
		if policies, err := expandAwsGroupScalingPolicies(v); err != nil {
			return nil, err
		} else {
			group.Scaling.Up = policies
		}
	}

	if v, ok := d.GetOk("scaling_down_policy"); ok {
		if policies, err := expandAwsGroupScalingPolicies(v); err != nil {
			return nil, err
		} else {
			group.Scaling.Down = policies
		}
	}

	if v, ok := d.GetOk("scheduled_task"); ok {
		if tasks, err := expandAwsGroupScheduledTasks(v); err != nil {
			return nil, err
		} else {
			group.Scheduling.Tasks = tasks
		}
	}

	if v, ok := d.GetOk("instance_types"); ok {
		if types, err := expandAwsGroupInstanceTypes(v); err != nil {
			return nil, err
		} else {
			group.Compute.InstanceTypes = types
		}
	}

	if v, ok := d.GetOk("elastic_ips"); ok {
		if eips, err := expandAwsGroupElasticIPs(v); err != nil {
			return nil, err
		} else {
			group.Compute.ElasticIPs = eips
		}
	}

	if v, ok := d.GetOk("availability_zone"); ok {
		if zones, err := expandAwsGroupAvailabilityZones(v); err != nil {
			return nil, err
		} else {
			group.Compute.AvailabilityZones = zones
		}
	}

	if v, ok := d.GetOk("signal"); ok {
		if signals, err := expandAwsGroupSignals(v); err != nil {
			return nil, err
		} else {
			group.Strategy.Signals = signals
		}
	}

	if v, ok := d.GetOk("launch_specification"); ok {
		if lc, err := expandAwsGroupLaunchSpecification(v); err != nil {
			return nil, err
		} else {
			group.Compute.LaunchSpecification = lc
		}
	}

	if v, ok := d.GetOk("tags"); ok {
		if tags, err := expandAwsGroupTags(v); err != nil {
			return nil, err
		} else {
			group.Compute.LaunchSpecification.Tags = tags
		}
	}

	if v, ok := d.GetOk("network_interface"); ok {
		if interfaces, err := expandAwsGroupNetworkInterfaces(v); err != nil {
			return nil, err
		} else {
			group.Compute.LaunchSpecification.NetworkInterfaces = interfaces
		}
	}

	if v, ok := d.GetOk("ebs_block_device"); ok {
		if devices, err := expandAwsGroupEBSBlockDevices(v); err != nil {
			return nil, err
		} else {
			group.Compute.LaunchSpecification.BlockDevices = devices
		}
	}

	if v, ok := d.GetOk("ephemeral_block_device"); ok {
		if devices, err := expandAwsGroupEphemeralBlockDevices(v); err != nil {
			return nil, err
		} else {
			if len(group.Compute.LaunchSpecification.BlockDevices) > 0 {
				for _, d := range devices {
					group.Compute.LaunchSpecification.BlockDevices = append(group.Compute.LaunchSpecification.BlockDevices, d)
				}
			} else {
				group.Compute.LaunchSpecification.BlockDevices = devices
			}
		}
	}

	if v, ok := d.GetOk("rancher_integration"); ok {
		if integration, err := expandAwsGroupRancherIntegration(v); err != nil {
			return nil, err
		} else {
			group.Integration.Rancher = integration
		}
	}

	if v, ok := d.GetOk("elastic_beanstalk_integration"); ok {
		if integration, err := expandAwsGroupElasticBeanstalkIntegration(v); err != nil {
			return nil, err
		} else {
			group.Integration.ElasticBeanstalk = integration
		}
	}

	if v, ok := d.GetOk("ec2_container_service_integration"); ok {
		if integration, err := expandAwsGroupEC2ContainerServiceIntegration(v); err != nil {
			return nil, err
		} else {
			group.Integration.EC2ContainerService = integration
		}
	}

	if v, ok := d.GetOk("nirmata_integration"); ok {
		if integration, err := expandAwsGroupNirmataIntegration(v); err != nil {
			return nil, err
		} else {
			group.Integration.Nirmata = integration
		}
	}

	return group, nil
}

// expandAwsGroupCapacity expands the Capacity block.
func expandAwsGroupCapacity(data interface{}) (*spotinst.AwsGroupCapacity, error) {
	if list := data.(*schema.Set).List(); len(list) != 1 {
		return nil, fmt.Errorf("Only a single capacity block is expected")
	} else {
		m := list[0].(map[string]interface{})
		capacity := &spotinst.AwsGroupCapacity{}

		if v, ok := m["minimum"].(int); ok && v >= 0 {
			capacity.Minimum = spotinst.Int(v)
		}

		if v, ok := m["maximum"].(int); ok && v >= 0 {
			capacity.Maximum = spotinst.Int(v)
		}

		if v, ok := m["target"].(int); ok && v >= 0 {
			capacity.Target = spotinst.Int(v)
		}

		log.Printf("[DEBUG] AwsGroup capacity configuration: %#v\n", capacity)
		return capacity, nil
	}
}

// expandAwsGroupStrategy expands the Strategy block.
func expandAwsGroupStrategy(data interface{}) (*spotinst.AwsGroupStrategy, error) {
	if list := data.(*schema.Set).List(); len(list) != 1 {
		return nil, fmt.Errorf("Only a single strategy block is expected")
	} else {
		m := list[0].(map[string]interface{})
		strategy := &spotinst.AwsGroupStrategy{}

		if v, ok := m["risk"].(float64); ok && v >= 0 {
			strategy.Risk = spotinst.Float64(v)
		}

		if v, ok := m["ondemand_count"].(int); ok && v > 0 {
			strategy.OnDemandCount = spotinst.Int(v)
		}

		if v, ok := m["availability_vs_cost"].(string); ok && v != "" {
			strategy.AvailabilityVsCost = spotinst.String(v)
		}

		if v, ok := m["draining_timeout"].(int); ok && v > 0 {
			strategy.DrainingTimeout = spotinst.Int(v)
		}

		log.Printf("[DEBUG] AwsGroup strategy configuration: %#v\n", strategy)
		return strategy, nil
	}
}

// expandAwsGroupScalingPolicies expands the Scaling Policy block.
func expandAwsGroupScalingPolicies(data interface{}) ([]*spotinst.AwsGroupScalingPolicy, error) {
	list := data.(*schema.Set).List()
	policies := make([]*spotinst.AwsGroupScalingPolicy, 0, len(list))
	for _, item := range list {
		m := item.(map[string]interface{})
		policy := &spotinst.AwsGroupScalingPolicy{}

		if v, ok := m["policy_name"].(string); ok && v != "" {
			policy.PolicyName = spotinst.String(v)
		}

		if v, ok := m["metric_name"].(string); ok && v != "" {
			policy.MetricName = spotinst.String(v)
		}

		if v, ok := m["statistic"].(string); ok && v != "" {
			policy.Statistic = spotinst.String(v)
		}

		if v, ok := m["unit"].(string); ok && v != "" {
			policy.Unit = spotinst.String(v)
		}

		if v, ok := m["threshold"].(float64); ok && v > 0 {
			policy.Threshold = spotinst.Float64(v)
		}

		if v, ok := m["adjustment"].(int); ok && v > 0 {
			policy.Adjustment = spotinst.Int(v)
		}

		if v, ok := m["min_target_capacity"].(int); ok && v > 0 {
			policy.MinTargetCapacity = spotinst.Int(v)
		}

		if v, ok := m["max_target_capacity"].(int); ok && v > 0 {
			policy.MaxTargetCapacity = spotinst.Int(v)
		}

		if v, ok := m["namespace"].(string); ok && v != "" {
			policy.Namespace = spotinst.String(v)
		}

		if v, ok := m["period"].(int); ok && v > 0 {
			policy.Period = spotinst.Int(v)
		}

		if v, ok := m["evaluation_periods"].(int); ok && v > 0 {
			policy.EvaluationPeriods = spotinst.Int(v)
		}

		if v, ok := m["cooldown"].(int); ok {
			policy.Cooldown = spotinst.Int(v)
		}

		if v, ok := m["dimensions"].(map[string]interface{}); ok {
			dimensions := make([]*spotinst.AwsGroupScalingPolicyDimension, 0, len(v))
			for i, k := range v {
				dimensions = append(dimensions, &spotinst.AwsGroupScalingPolicyDimension{
					Name:  spotinst.String(i),
					Value: spotinst.String(k.(string)),
				})
			}

			policy.Dimensions = dimensions
		}

		log.Printf("[DEBUG] AwsGroup scaling policy configuration: %#v\n", policy)
		policies = append(policies, policy)
	}

	return policies, nil
}

// expandAwsGroupScheduledTasks expands the Scheduled Task block.
func expandAwsGroupScheduledTasks(data interface{}) ([]*spotinst.AwsGroupScheduledTask, error) {
	list := data.(*schema.Set).List()
	tasks := make([]*spotinst.AwsGroupScheduledTask, 0, len(list))
	for _, item := range list {
		m := item.(map[string]interface{})
		task := &spotinst.AwsGroupScheduledTask{}

		if v, ok := m["task_type"].(string); ok && v != "" {
			task.TaskType = spotinst.String(v)
		}

		if v, ok := m["frequency"].(string); ok && v != "" {
			task.Frequency = spotinst.String(v)
		}

		if v, ok := m["cron_expression"].(string); ok && v != "" {
			task.CronExpression = spotinst.String(v)
		}

		if v, ok := m["scale_target_capacity"].(int); ok && v > 0 {
			task.ScaleTargetCapacity = spotinst.Int(v)
		}

		if v, ok := m["scale_min_capacity"].(int); ok && v > 0 {
			task.ScaleMinCapacity = spotinst.Int(v)
		}

		if v, ok := m["scale_max_capacity"].(int); ok && v > 0 {
			task.ScaleMaxCapacity = spotinst.Int(v)
		}

		log.Printf("[DEBUG] AwsGroup scheduled task configuration: %#v\n", task)
		tasks = append(tasks, task)
	}

	return tasks, nil
}

// expandAwsGroupAvailabilityZones expands the Availability Zone block.
func expandAwsGroupAvailabilityZones(data interface{}) ([]*spotinst.AwsGroupComputeAvailabilityZone, error) {
	list := data.(*schema.Set).List()
	zones := make([]*spotinst.AwsGroupComputeAvailabilityZone, 0, len(list))
	for _, item := range list {
		m := item.(map[string]interface{})
		zone := &spotinst.AwsGroupComputeAvailabilityZone{}

		if v, ok := m["name"].(string); ok && v != "" {
			zone.Name = spotinst.String(v)
		}

		if v, ok := m["subnet_id"].(string); ok && v != "" {
			zone.SubnetID = spotinst.String(v)
		}

		log.Printf("[DEBUG] AwsGroup availability zone configuration: %#v\n", zone)
		zones = append(zones, zone)
	}

	return zones, nil
}

// expandAwsGroupSignals expands the Signal block.
func expandAwsGroupSignals(data interface{}) ([]*spotinst.AwsGroupStrategySignal, error) {
	list := data.(*schema.Set).List()
	signals := make([]*spotinst.AwsGroupStrategySignal, 0, len(list))
	for _, item := range list {
		m := item.(map[string]interface{})
		signal := &spotinst.AwsGroupStrategySignal{}

		if v, ok := m["name"].(string); ok && v != "" {
			signal.Name = spotinst.String(strings.ToUpper(v))
		}

		log.Printf("[DEBUG] AwsGroup signal configuration: %#v\n", signal)
		signals = append(signals, signal)
	}

	return signals, nil
}

// expandAwsGroupInstanceTypes expands the Instance Types block.
func expandAwsGroupInstanceTypes(data interface{}) (*spotinst.AwsGroupComputeInstanceType, error) {
	if list := data.(*schema.Set).List(); len(list) != 1 {
		return nil, fmt.Errorf("Only a single instance_types block is expected")
	} else {
		m := list[0].(map[string]interface{})
		types := &spotinst.AwsGroupComputeInstanceType{}
		if v, ok := m["ondemand"].(string); ok && v != "" {
			types.OnDemand = spotinst.String(v)
		}
		if v, ok := m["spot"].([]interface{}); ok {
			it := make([]string, len(v))
			for i, j := range v {
				it[i] = j.(string)
			}
			types.Spot = it
		}

		log.Printf("[DEBUG] AwsGroup instance types configuration: %#v\n", types)
		return types, nil
	}
}

// expandAwsGroupNetworkInterfaces expands the Elastic Network Interface block.
func expandAwsGroupNetworkInterfaces(data interface{}) ([]*spotinst.AwsGroupComputeNetworkInterface, error) {
	list := data.(*schema.Set).List()
	interfaces := make([]*spotinst.AwsGroupComputeNetworkInterface, 0, len(list))
	for _, item := range list {
		m := item.(map[string]interface{})
		iface := &spotinst.AwsGroupComputeNetworkInterface{}

		if v, ok := m["network_interface_id"].(string); ok && v != "" {
			iface.ID = spotinst.String(v)
		}

		if v, ok := m["description"].(string); ok && v != "" {
			iface.Description = spotinst.String(v)
		}

		if v, ok := m["device_index"].(int); ok && v >= 0 {
			iface.DeviceIndex = spotinst.Int(v)
		}

		if v, ok := m["secondary_private_ip_address_count"].(int); ok && v > 0 {
			iface.SecondaryPrivateIPAddressCount = spotinst.Int(v)
		}

		if v, ok := m["associate_public_ip_address"].(bool); ok {
			iface.AssociatePublicIPAddress = spotinst.Bool(v)
		}

		if v, ok := m["delete_on_termination"].(bool); ok {
			iface.DeleteOnTermination = spotinst.Bool(v)
		}

		if v, ok := m["private_ip_address"].(string); ok && v != "" {
			iface.PrivateIPAddress = spotinst.String(v)
		}

		if v, ok := m["subnet_id"].(string); ok && v != "" {
			iface.SubnetID = spotinst.String(v)
		}

		if v := m["security_group_ids"]; v != nil {
			var groups []string
			sgs := v.(*schema.Set).List()
			if len(sgs) > 0 {
				for _, v := range sgs {
					if s, ok := v.(string); ok && s != "" {
						groups = append(groups, s)
					}
				}
				iface.SecurityGroupsIDs = groups
			}
		}

		log.Printf("[DEBUG] AwsGroup network interface configuration: %#v\n", iface)
		interfaces = append(interfaces, iface)
	}

	return interfaces, nil
}

// expandAwsGroupEphemeralBlockDevice expands the Ephemeral Block Device block.
func expandAwsGroupEphemeralBlockDevices(data interface{}) ([]*spotinst.AwsGroupComputeBlockDevice, error) {
	list := data.(*schema.Set).List()
	devices := make([]*spotinst.AwsGroupComputeBlockDevice, 0, len(list))
	for _, item := range list {
		m := item.(map[string]interface{})
		device := &spotinst.AwsGroupComputeBlockDevice{}

		if v, ok := m["device_name"].(string); ok && v != "" {
			device.DeviceName = spotinst.String(v)
		}

		if v, ok := m["virtual_name"].(string); ok && v != "" {
			device.VirtualName = spotinst.String(v)
		}

		log.Printf("[DEBUG] AwsGroup ephemeral block device configuration: %#v\n", device)
		devices = append(devices, device)
	}

	return devices, nil
}

// expandAwsGroupEBSBlockDevices expands the EBS Block Device block.
func expandAwsGroupEBSBlockDevices(data interface{}) ([]*spotinst.AwsGroupComputeBlockDevice, error) {
	list := data.(*schema.Set).List()
	devices := make([]*spotinst.AwsGroupComputeBlockDevice, 0, len(list))
	for _, item := range list {
		m := item.(map[string]interface{})
		device := &spotinst.AwsGroupComputeBlockDevice{EBS: &spotinst.AwsGroupComputeEBS{}}

		if v, ok := m["device_name"].(string); ok && v != "" {
			device.DeviceName = spotinst.String(v)
		}

		if v, ok := m["delete_on_termination"].(bool); ok {
			device.EBS.DeleteOnTermination = spotinst.Bool(v)
		}

		if v, ok := m["encrypted"].(bool); ok {
			device.EBS.Encrypted = spotinst.Bool(v)
		}

		if v, ok := m["snapshot_id"].(string); ok && v != "" {
			device.EBS.SnapshotID = spotinst.String(v)
		}

		if v, ok := m["volume_type"].(string); ok && v != "" {
			device.EBS.VolumeType = spotinst.String(v)
		}

		if v, ok := m["volume_size"].(int); ok && v > 0 {
			device.EBS.VolumeSize = spotinst.Int(v)
		}

		if v, ok := m["iops"].(int); ok && v > 0 {
			device.EBS.IOPS = spotinst.Int(v)
		}

		log.Printf("[DEBUG] AwsGroup elastic block device configuration: %#v\n", device)
		devices = append(devices, device)
	}

	return devices, nil
}

// expandAwsGroupLaunchSpecification expands the launch Specification block.
func expandAwsGroupLaunchSpecification(data interface{}) (*spotinst.AwsGroupComputeLaunchSpecification, error) {
	if list := data.(*schema.Set).List(); len(list) != 1 {
		return nil, fmt.Errorf("Only a single launch_specification block is expected")
	} else {
		m := list[0].(map[string]interface{})
		lc := &spotinst.AwsGroupComputeLaunchSpecification{}

		if v, ok := m["monitoring"].(bool); ok {
			lc.Monitoring = spotinst.Bool(v)
		}

		if v, ok := m["image_id"].(string); ok && v != "" {
			lc.ImageID = spotinst.String(v)
		}

		if v, ok := m["key_pair"].(string); ok && v != "" {
			lc.KeyPair = spotinst.String(v)
		}

		if v, ok := m["health_check_type"].(string); ok && v != "" {
			lc.HealthCheckType = spotinst.String(v)
		}

		if v, ok := m["health_check_grace_period"].(int); ok && v > 0 {
			lc.HealthCheckGracePeriod = spotinst.Int(v)
		}

		if v, ok := m["iam_instance_profile"].(string); ok && v != "" {
			lc.IamInstanceProfile = &spotinst.AwsGroupComputeIamInstanceProfile{Arn: spotinst.String(v)}
		}

		if v, ok := m["user_data"].(string); ok && v != "" {
			lc.UserData = spotinst.String(base64.StdEncoding.EncodeToString([]byte(v)))
		}

		if v := m["security_group_ids"]; v != nil {
			var groups []string
			sgs := v.(*schema.Set).List()
			if len(sgs) > 0 {
				for _, v := range sgs {
					if s, ok := v.(string); ok && s != "" {
						groups = append(groups, s)
					}
				}
				lc.SecurityGroupIDs = groups
			}
		}

		if v := m["load_balancer_names"]; v != nil {
			var names []string
			elbs := v.(*schema.Set).List()
			if len(elbs) > 0 {
				for _, v := range elbs {
					if s, ok := v.(string); ok && s != "" {
						names = append(names, s)
					}
				}
				lc.LoadBalancerNames = names
			}
		}

		log.Printf("[DEBUG] AwsGroup launch specification configuration: %#v\n", lc)
		return lc, nil
	}
}

// expandAwsGroupRancherIntegration expands the Rancher Integration block.
func expandAwsGroupRancherIntegration(data interface{}) (*spotinst.AwsGroupRancherIntegration, error) {
	if list := data.(*schema.Set).List(); len(list) != 1 {
		return nil, fmt.Errorf("Only a single rancher_integration block is expected")
	} else {
		m := list[0].(map[string]interface{})
		i := &spotinst.AwsGroupRancherIntegration{}

		if v, ok := m["master_host"].(string); ok && v != "" {
			i.MasterHost = spotinst.String(v)
		}

		if v, ok := m["access_key"].(string); ok && v != "" {
			i.AccessKey = spotinst.String(v)
		}

		if v, ok := m["secret_key"].(string); ok && v != "" {
			i.SecretKey = spotinst.String(v)
		}

		log.Printf("[DEBUG] AwsGroup Rancher integration configuration: %#v\n", i)
		return i, nil
	}
}

// expandAwsGroupElasticBeanstalkIntegration expands the Elastic Beanstalk Integration block.
func expandAwsGroupElasticBeanstalkIntegration(data interface{}) (*spotinst.AwsGroupElasticBeanstalkIntegration, error) {
	if list := data.(*schema.Set).List(); len(list) != 1 {
		return nil, fmt.Errorf("Only a single elastic_beanstalk_integration block is expected")
	} else {
		m := list[0].(map[string]interface{})
		i := &spotinst.AwsGroupElasticBeanstalkIntegration{}

		if v, ok := m["environment_id"].(string); ok && v != "" {
			i.EnvironmentID = spotinst.String(v)
		}

		log.Printf("[DEBUG] AwsGroup Elastic Beanstalk integration configuration: %#v\n", i)
		return i, nil
	}
}

// expandAwsGroupEC2ContainerServiceIntegration expands the EC2 Container Service Integration block.
func expandAwsGroupEC2ContainerServiceIntegration(data interface{}) (*spotinst.AwsGroupEC2ContainerServiceIntegration, error) {
	if list := data.(*schema.Set).List(); len(list) != 1 {
		return nil, fmt.Errorf("Only a single ec2_container_service_integration block is expected")
	} else {
		m := list[0].(map[string]interface{})
		i := &spotinst.AwsGroupEC2ContainerServiceIntegration{}

		if v, ok := m["cluster_name"].(string); ok && v != "" {
			i.ClusterName = spotinst.String(v)
		}

		log.Printf("[DEBUG] AwsGroup ECS integration configuration: %#v\n", i)
		return i, nil
	}
}

// expandAwsGroupNirmataIntegration expands the Nirmata Integration block.
func expandAwsGroupNirmataIntegration(data interface{}) (*spotinst.AwsGroupNirmataIntegration, error) {
	if list := data.(*schema.Set).List(); len(list) != 1 {
		return nil, fmt.Errorf("Only a single nirmata_integration block is expected")
	} else {
		m := list[0].(map[string]interface{})
		i := &spotinst.AwsGroupNirmataIntegration{}

		if v, ok := m["api_key"].(string); ok && v != "" {
			i.APIKey = spotinst.String(v)
		}

		log.Printf("[DEBUG] AwsGroup Nirmata integration configuration: %#v\n", i)
		return i, nil
	}
}

// expandAwsGroupElasticIPs expands the Elastic IPs block.
func expandAwsGroupElasticIPs(data interface{}) ([]string, error) {
	list := data.(*schema.Set).List()
	eips := make([]string, 0, len(list))
	for _, str := range list {
		if eip, ok := str.(string); ok {
			log.Printf("[DEBUG] AwsGroup elastic IP configuration: %#v\n", eip)
			eips = append(eips, eip)
		}
	}

	return eips, nil
}

// expandAwsGroupTags expands the Tags block.
func expandAwsGroupTags(data interface{}) ([]*spotinst.AwsGroupComputeTag, error) {
	list := data.(map[string]interface{})
	tags := make([]*spotinst.AwsGroupComputeTag, 0, len(list))
	for k, v := range list {
		tag := &spotinst.AwsGroupComputeTag{
			Key:   spotinst.String(k),
			Value: spotinst.String(v.(string)),
		}

		log.Printf("[DEBUG] AwsGroup tag configuration: %#v\n", tag)
		tags = append(tags, tag)
	}

	return tags, nil
}

// tagsToMap turns the list of tags into a map.
func tagsToMap(ts []*spotinst.AwsGroupComputeTag) map[string]string {
	result := make(map[string]string)
	for _, t := range ts {
		result[*t.Key] = *t.Value
	}
	return result
}
