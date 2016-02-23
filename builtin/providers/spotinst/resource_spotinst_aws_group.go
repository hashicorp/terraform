package spotinst

import (
	"bytes"
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"log"

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
							Type:     schema.TypeList,
							Optional: true,
							ForceNew: false,
							Elem:     &schema.Schema{Type: schema.TypeString},
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
							Type:     schema.TypeList,
							Required: true,
							ForceNew: false,
							Elem:     &schema.Schema{Type: schema.TypeString},
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
							Type:     schema.TypeList,
							Optional: true,
							ForceNew: false,
							Elem:     &schema.Schema{Type: schema.TypeString},
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
	d.SetId(res[0].ID)
	log.Printf("[INFO] AwsGroup created successfully: %s\n", d.Id())
	return resourceSpotinstAwsGroupRead(d, meta)
}

func resourceSpotinstAwsGroupRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*spotinst.Client)
	groups, _, err := client.AwsGroup.Get(d.Id())
	if err != nil {
		serr, ok := err.(*spotinst.ErrorResponse)
		if ok {
			for _, r := range serr.Errors {
				if r.Code == "400" {
					d.SetId("")
					return nil
				} else {
					return fmt.Errorf("[ERROR] Error retrieving group: %s", err)
				}
			}
		} else {
			return fmt.Errorf("[ERROR] Error retrieving group: %s", err)
		}
	}
	if len(groups) == 0 {
		return fmt.Errorf("[ERROR] No matching group %s", d.Id())
	} else if len(groups) > 1 {
		return fmt.Errorf("[ERROR] Got %d results, only one is allowed", len(groups))
	} else {
		g := groups[0]
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
		lspec = append(lspec, map[string]interface{}{
			"health_check_grace_period": g.Compute.LaunchSpecification.HealthCheckGracePeriod,
			"health_check_type":         g.Compute.LaunchSpecification.HealthCheckType,
			"iam_instance_profile":      g.Compute.LaunchSpecification.IamInstanceProfile.Arn,
			"image_id":                  g.Compute.LaunchSpecification.ImageID,
			"key_pair":                  g.Compute.LaunchSpecification.KeyPair,
			"load_balancer_names":       g.Compute.LaunchSpecification.LoadBalancerNames,
			"monitoring":                g.Compute.LaunchSpecification.Monitoring,
			"security_group_ids":        g.Compute.LaunchSpecification.SecurityGroupIDs,
			"user_data":                 g.Compute.LaunchSpecification.UserData,
		})
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

		// Set the Nirmata integration.
		nirmata := make([]map[string]interface{}, 0, 1)
		if g.Integration.Nirmata != nil {
			nirmata = append(nirmata, map[string]interface{}{
				"api_key": g.Integration.Nirmata.APIKey,
			})
		}
		d.Set("nirmata_integration", nirmata)

	}
	return nil
}

func resourceSpotinstAwsGroupUpdate(d *schema.ResourceData, meta interface{}) error {
	hasChange := false
	client := meta.(*spotinst.Client)
	update := &spotinst.AwsGroup{ID: d.Id()}

	if d.HasChange("name") {
		update.Name = d.Get("name").(string)
		hasChange = true
	}

	if d.HasChange("description") {
		update.Description = d.Get("description").(string)
		hasChange = true
	}

	if d.HasChange("capacity") {
		vL := d.Get("capacity").(*schema.Set).List()
		if len(vL) > 1 {
			return fmt.Errorf("Only a single capacity block is expected")
		} else if len(vL) == 1 {
			c := vL[0].(map[string]interface{})
			update.Capacity = expandAwsGroupCapacity(c)
			hasChange = true
		}
	}

	if d.HasChange("strategy") {
		vL := d.Get("strategy").(*schema.Set).List()
		if len(vL) > 1 {
			return fmt.Errorf("Only a single strategy block is expected")
		} else if len(vL) == 1 {
			c := vL[0].(map[string]interface{})
			update.Strategy = expandAwsGroupStrategy(c)
			hasChange = true
		}
	}

	if d.HasChange("launch_specification") {
		vL := d.Get("launch_specification").(*schema.Set).List()
		if len(vL) > 1 {
			return fmt.Errorf("Only a single launch_specification block is expected")
		} else if len(vL) == 1 {
			c := vL[0].(map[string]interface{})
			if update.Compute == nil {
				update.Compute = &spotinst.AwsGroupCompute{}
			}
			update.Compute.LaunchSpecification = expandAwsGroupLaunchSpecification(c)
			hasChange = true
		}
	}

	if d.HasChange("network_interface") {
		vL := d.Get("network_interface").(*schema.Set).List()
		interfaces := make([]*spotinst.AwsGroupComputeNetworkInterface, 0, len(vL))
		for _, c := range vL {
			if i, ok := c.(map[string]interface{}); ok {
				iface := expandAwsGroupNetworkInterface(i)
				interfaces = append(interfaces, iface)
			}
		}
		if update.Compute == nil {
			update.Compute = &spotinst.AwsGroupCompute{}
		} else if update.Compute.LaunchSpecification == nil {
			update.Compute.LaunchSpecification = &spotinst.AwsGroupComputeLaunchSpecification{}
		}
		update.Compute.LaunchSpecification.NetworkInterfaces = interfaces
		hasChange = true
	}

	if d.HasChange("availability_zone") {
		vL := d.Get("availability_zone").(*schema.Set).List()
		zones := make([]*spotinst.AwsGroupComputeAvailabilityZone, 0, len(vL))
		for _, c := range vL {
			if z, ok := c.(map[string]interface{}); ok {
				zone := expandAwsGroupAvailabilityZone(z)
				zones = append(zones, zone)
			}
		}
		if update.Compute == nil {
			update.Compute = &spotinst.AwsGroupCompute{}
		}
		update.Compute.AvailabilityZones = zones
		hasChange = true
	}

	if d.HasChange("instance_types") {
		vL := d.Get("instance_types").(*schema.Set).List()
		if len(vL) > 1 {
			return fmt.Errorf("Only a single instance_types block is expected")
		} else if len(vL) == 1 {
			c := vL[0].(map[string]interface{})
			it := &spotinst.AwsGroupComputeInstanceType{}
			if v, ok := c["ondemand"].(string); ok && v != "" {
				it.OnDemand = v
			}
			if v, ok := c["spot"].([]interface{}); ok {
				types := make([]string, len(v))
				for i, j := range v {
					types[i] = j.(string)
				}
				it.Spot = types
			}
			if update.Compute == nil {
				update.Compute = &spotinst.AwsGroupCompute{}
			}
			update.Compute.InstanceTypes = it
			hasChange = true
		}
	}

	if d.HasChange("tags") {
		if v, ok := d.GetOk("tags"); ok {
			c := v.(map[string]interface{})
			tags := make([]*spotinst.AwsGroupComputeTag, 0, len(c))
			for i, k := range c {
				tags = append(tags, &spotinst.AwsGroupComputeTag{
					Key:   i,
					Value: k.(string),
				})
			}
			if update.Compute == nil {
				update.Compute = &spotinst.AwsGroupCompute{}
			} else if update.Compute.LaunchSpecification == nil {
				update.Compute.LaunchSpecification = &spotinst.AwsGroupComputeLaunchSpecification{}
			}
			update.Compute.LaunchSpecification.Tags = tags
			hasChange = true
		}
	}

	if d.HasChange("elastic_ips") {
		if v, ok := d.GetOk("elastic_ips"); ok {
			c := v.(*schema.Set).List()
			eips := make([]string, 0, len(c))
			for _, e := range c {
				if eip, ok := e.(string); ok {
					eips = append(eips, eip)
				}
			}
			if len(eips) > 0 {
				if update.Compute == nil {
					update.Compute = &spotinst.AwsGroupCompute{}
				}
				update.Compute.ElasticIPs = make([]string, len(eips))
				copy(update.Compute.ElasticIPs, eips)
				hasChange = true
			}
		}
	}

	if d.HasChange("scheduled_task") {
		vL := d.Get("scheduled_task").(*schema.Set).List()
		tasks := make([]*spotinst.AwsGroupScheduledTask, 0, len(vL))
		for _, c := range vL {
			if t, ok := c.(map[string]interface{}); ok {
				task := expandAwsGroupScheduledTask(t)
				tasks = append(tasks, task)
			}
		}
		if update.Scheduling == nil {
			update.Scheduling = &spotinst.AwsGroupScheduling{}
		}
		update.Scheduling.Tasks = tasks
		hasChange = true
	}

	if d.HasChange("scaling_up_policy") {
		vL := d.Get("scaling_up_policy").(*schema.Set).List()
		policies := make([]*spotinst.AwsGroupScalingPolicy, 0, len(vL))
		for _, c := range vL {
			if p, ok := c.(map[string]interface{}); ok {
				policy := expandAwsGroupScalingPolicy(p)
				policies = append(policies, policy)
			}
		}
		if update.Scaling == nil {
			update.Scaling = &spotinst.AwsGroupScaling{}
		}
		update.Scaling.Up = policies
		hasChange = true
	}

	if d.HasChange("scaling_down_policy") {
		vL := d.Get("scaling_down_policy").(*schema.Set).List()
		policies := make([]*spotinst.AwsGroupScalingPolicy, 0, len(vL))
		for _, c := range vL {
			if p, ok := c.(map[string]interface{}); ok {
				policy := expandAwsGroupScalingPolicy(p)
				policies = append(policies, policy)
			}
		}
		if update.Scaling == nil {
			update.Scaling = &spotinst.AwsGroupScaling{}
		}
		update.Scaling.Down = policies
		hasChange = true
	}

	if d.HasChange("rancher_integration") {
		vL := d.Get("rancher_integration").(*schema.Set).List()
		if len(vL) > 1 {
			return fmt.Errorf("Only a single rancher_integration block is expected")
		} else if len(vL) == 1 {
			c := vL[0].(map[string]interface{})
			if update.Integration == nil {
				update.Integration = &spotinst.AwsGroupIntegration{}
			}
			update.Integration.Rancher = expandAwsGroupRancherIntegration(c)
			hasChange = true
		}
	}

	if d.HasChange("elastic_eanstalk_integration") {
		vL := d.Get("elastic_eanstalk_integration").(*schema.Set).List()
		if len(vL) > 1 {
			return fmt.Errorf("Only a single elastic_eanstalk_integration block is expected")
		} else if len(vL) == 1 {
			c := vL[0].(map[string]interface{})
			if update.Integration == nil {
				update.Integration = &spotinst.AwsGroupIntegration{}
			}
			update.Integration.ElasticBeanstalk = expandAwsGroupElasticBeanstalkIntegration(c)
			hasChange = true
		}
	}

	if d.HasChange("nirmata_integration") {
		vL := d.Get("nirmata_integration").(*schema.Set).List()
		if len(vL) > 1 {
			return fmt.Errorf("Only a single nirmata_integration block is expected")
		} else if len(vL) == 1 {
			c := vL[0].(map[string]interface{})
			if update.Integration == nil {
				update.Integration = &spotinst.AwsGroupIntegration{}
			}
			update.Integration.Nirmata = expandAwsGroupNirmataIntegration(c)
			hasChange = true
		}
	}

	if hasChange {
		log.Printf("[DEBUG] AwsGroup update configuration: %#v\n", update)
		_, _, err := client.AwsGroup.Update(update)
		if err != nil {
			return fmt.Errorf("[ERROR] Error updating group: %s", err)
		}
	}

	return resourceSpotinstAwsGroupRead(d, meta)
}

func resourceSpotinstAwsGroupDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*spotinst.Client)
	log.Printf("[INFO] Deleting group: %s\n", d.Id())
	group := &spotinst.AwsGroup{ID: d.Id()}
	_, err := client.AwsGroup.Delete(group)
	if err != nil {
		return fmt.Errorf("[ERROR] Error deleting group: %s", err)
	}
	return nil
}

// buildAwsGroupOpts builds the Spotinst AWS Group options.
func buildAwsGroupOpts(d *schema.ResourceData, meta interface{}) (*spotinst.AwsGroup, error) {
	group := &spotinst.AwsGroup{
		Name:        d.Get("name").(string),
		Description: d.Get("description").(string),
		Scaling:     &spotinst.AwsGroupScaling{},
		Scheduling:  &spotinst.AwsGroupScheduling{},
		Integration: &spotinst.AwsGroupIntegration{},
		Compute: &spotinst.AwsGroupCompute{
			Product:             d.Get("product").(string),
			LaunchSpecification: &spotinst.AwsGroupComputeLaunchSpecification{},
		},
	}

	if v, ok := d.GetOk("capacity"); ok {
		vL := v.(*schema.Set).List()
		if len(vL) > 1 {
			return nil, fmt.Errorf("Only a single capacity block is expected")
		} else if len(vL) == 1 {
			c := vL[0].(map[string]interface{})
			group.Capacity = expandAwsGroupCapacity(c)
		}
	}

	if v, ok := d.GetOk("strategy"); ok {
		vL := v.(*schema.Set).List()
		if len(vL) > 1 {
			return nil, fmt.Errorf("Only a single strategy block is expected")
		} else if len(vL) == 1 {
			c := vL[0].(map[string]interface{})
			group.Strategy = expandAwsGroupStrategy(c)
		}
	}

	if v, ok := d.GetOk("scaling_up_policy"); ok {
		vL := v.(*schema.Set).List()
		policies := make([]*spotinst.AwsGroupScalingPolicy, 0, len(vL))
		for _, c := range vL {
			if p, ok := c.(map[string]interface{}); ok {
				policy := expandAwsGroupScalingPolicy(p)
				policies = append(policies, policy)
			}
		}
		group.Scaling.Up = policies
	}

	if v, ok := d.GetOk("scaling_down_policy"); ok {
		vL := v.(*schema.Set).List()
		policies := make([]*spotinst.AwsGroupScalingPolicy, 0, len(vL))
		for _, c := range vL {
			if p, ok := c.(map[string]interface{}); ok {
				policy := expandAwsGroupScalingPolicy(p)
				policies = append(policies, policy)
			}
		}
		group.Scaling.Down = policies
	}

	if v, ok := d.GetOk("scheduled_task"); ok {
		vL := v.(*schema.Set).List()
		tasks := make([]*spotinst.AwsGroupScheduledTask, 0, len(vL))
		for _, c := range vL {
			if t, ok := c.(map[string]interface{}); ok {
				task := expandAwsGroupScheduledTask(t)
				tasks = append(tasks, task)
			}
		}
		group.Scheduling.Tasks = tasks
	}

	if v, ok := d.GetOk("instance_types"); ok {
		vL := v.(*schema.Set).List()
		if len(vL) > 1 {
			return nil, fmt.Errorf("Only a single instance_types block is expected")
		} else if len(vL) == 1 {
			c := vL[0].(map[string]interface{})
			it := &spotinst.AwsGroupComputeInstanceType{}
			if v, ok := c["ondemand"].(string); ok && v != "" {
				it.OnDemand = v
			}
			if v, ok := c["spot"].([]interface{}); ok {
				types := make([]string, len(v))
				for i, j := range v {
					types[i] = j.(string)
				}
				it.Spot = types
			}
			group.Compute.InstanceTypes = it
		}
	}

	if v, ok := d.GetOk("elastic_ips"); ok {
		c := v.(*schema.Set).List()
		eips := make([]string, 0, len(c))
		for _, e := range c {
			if eip, ok := e.(string); ok {
				eips = append(eips, eip)
			}
		}
		if len(eips) > 0 {
			group.Compute.ElasticIPs = make([]string, len(eips))
			copy(group.Compute.ElasticIPs, eips)
		}
	}

	if v, ok := d.GetOk("availability_zone"); ok {
		vL := v.(*schema.Set).List()
		zones := make([]*spotinst.AwsGroupComputeAvailabilityZone, 0, len(vL))
		for _, c := range vL {
			if z, ok := c.(map[string]interface{}); ok {
				zone := expandAwsGroupAvailabilityZone(z)
				zones = append(zones, zone)
			}
		}
		group.Compute.AvailabilityZones = zones
	}

	if v, ok := d.GetOk("launch_specification"); ok {
		vL := v.(*schema.Set).List()
		if len(vL) > 1 {
			return nil, fmt.Errorf("Only a single launch_specification block is expected")
		} else if len(vL) == 1 {
			c := vL[0].(map[string]interface{})
			lc := expandAwsGroupLaunchSpecification(c)
			group.Compute.LaunchSpecification = lc
		}
	}

	if v, ok := d.GetOk("tags"); ok {
		c := v.(map[string]interface{})
		tags := make([]*spotinst.AwsGroupComputeTag, 0, len(c))
		for i, k := range c {
			tags = append(tags, &spotinst.AwsGroupComputeTag{
				Key:   i,
				Value: k.(string),
			})
		}
		group.Compute.LaunchSpecification.Tags = tags
	}

	if v, ok := d.GetOk("network_interface"); ok {
		vL := v.(*schema.Set).List()
		interfaces := make([]*spotinst.AwsGroupComputeNetworkInterface, 0, len(vL))
		for _, c := range vL {
			if i, ok := c.(map[string]interface{}); ok {
				iface := expandAwsGroupNetworkInterface(i)
				interfaces = append(interfaces, iface)
			}
		}
		group.Compute.LaunchSpecification.NetworkInterfaces = interfaces
	}

	if v, ok := d.GetOk("ebs_block_device"); ok {
		vL := v.(*schema.Set).List()
		devices := make([]*spotinst.AwsGroupComputeBlockDevice, 0, len(vL))
		for _, c := range vL {
			if d, ok := c.(map[string]interface{}); ok {
				dev := expandAwsGroupEBSBlockDevice(d)
				devices = append(devices, dev)
			}
		}
		group.Compute.LaunchSpecification.BlockDevices = devices
	}

	if v, ok := d.GetOk("ephemeral_block_device"); ok {
		vL := v.(*schema.Set).List()
		devices := make([]*spotinst.AwsGroupComputeBlockDevice, 0, len(vL))
		for _, c := range vL {
			if d, ok := c.(map[string]interface{}); ok {
				dev := expandAwsGroupEphemeralBlockDevice(d)
				devices = append(devices, dev)
			}
		}
		if len(group.Compute.LaunchSpecification.BlockDevices) > 0 {
			for _, d := range devices {
				group.Compute.LaunchSpecification.BlockDevices = append(group.Compute.LaunchSpecification.BlockDevices, d)
			}
		} else {
			group.Compute.LaunchSpecification.BlockDevices = devices
		}
	}

	if v, ok := d.GetOk("rancher_integration"); ok {
		vL := v.(*schema.Set).List()
		if len(vL) > 1 {
			return nil, fmt.Errorf("Only a single rancher_integration block is expected")
		} else if len(vL) == 1 {
			if c, ok := vL[0].(map[string]interface{}); ok {
				i := expandAwsGroupRancherIntegration(c)
				group.Integration.Rancher = i
			}
		}
	}

	if v, ok := d.GetOk("elastic_beanstalk_integration"); ok {
		vL := v.(*schema.Set).List()
		if len(vL) > 1 {
			return nil, fmt.Errorf("Only a single elastic_beanstalk_integration block is expected")
		} else if len(vL) == 1 {
			if c, ok := vL[0].(map[string]interface{}); ok {
				i := expandAwsGroupElasticBeanstalkIntegration(c)
				group.Integration.ElasticBeanstalk = i
			}
		}
	}

	if v, ok := d.GetOk("nirmata_integration"); ok {
		vL := v.(*schema.Set).List()
		if len(vL) > 1 {
			return nil, fmt.Errorf("Only a single nirmata_integration block is expected")
		} else if len(vL) == 1 {
			if c, ok := vL[0].(map[string]interface{}); ok {
				i := expandAwsGroupNirmataIntegration(c)
				group.Integration.Nirmata = i
			}
		}
	}

	return group, nil
}

// expandAwsGroupCapacity expands the Capacity block.
func expandAwsGroupCapacity(m map[string]interface{}) *spotinst.AwsGroupCapacity {
	capacity := &spotinst.AwsGroupCapacity{}

	if v, ok := m["minimum"].(int); ok && v > 0 {
		capacity.Minimum = v
	}

	if v, ok := m["maximum"].(int); ok && v > 0 {
		capacity.Maximum = v
	}

	if v, ok := m["target"].(int); ok && v > 0 {
		capacity.Target = v
	}

	log.Printf("[DEBUG] AwsGroup capacity configuration: %#v\n", capacity)
	return capacity
}

// expandAwsGroupStrategy expands the Strategy block.
func expandAwsGroupStrategy(m map[string]interface{}) *spotinst.AwsGroupStrategy {
	strategy := &spotinst.AwsGroupStrategy{}

	if v, ok := m["risk"].(float64); ok && v >= 0 {
		strategy.Risk = v
	}

	if v, ok := m["ondemand_count"].(int); ok && v >= 0 {
		strategy.OnDemandCount = v
	}

	if v, ok := m["availability_vs_cost"].(string); ok && v != "" {
		strategy.AvailabilityVsCost = v
	}

	if v, ok := m["draining_timeout"].(int); ok && v >= 0 {
		strategy.DrainingTimeout = v
	}

	log.Printf("[DEBUG] AwsGroup strategy configuration: %#v\n", strategy)
	return strategy
}

// expandAwsGroupScalingPolicy expands the Scaling Policy block.
func expandAwsGroupScalingPolicy(m map[string]interface{}) *spotinst.AwsGroupScalingPolicy {
	p := &spotinst.AwsGroupScalingPolicy{}

	if v, ok := m["policy_name"].(string); ok && v != "" {
		p.PolicyName = v
	}

	if v, ok := m["metric_name"].(string); ok && v != "" {
		p.MetricName = v
	}

	if v, ok := m["statistic"].(string); ok && v != "" {
		p.Statistic = v
	}

	if v, ok := m["unit"].(string); ok && v != "" {
		p.Unit = v
	}

	if v, ok := m["threshold"].(float64); ok && v > 0 {
		p.Threshold = v
	}

	if v, ok := m["adjustment"].(int); ok && v > 0 {
		p.Adjustment = v
	}

	if v, ok := m["min_target_capacity"].(int); ok && v >= 0 {
		p.MinTargetCapacity = v
	}

	if v, ok := m["max_target_capacity"].(int); ok && v >= 0 {
		p.MaxTargetCapacity = v
	}

	if v, ok := m["namespace"].(string); ok && v != "" {
		p.Namespace = v
	}

	if v, ok := m["period"].(int); ok && v > 0 {
		p.Period = v
	}

	if v, ok := m["evaluation_periods"].(int); ok && v > 0 {
		p.EvaluationPeriods = v
	}

	if v, ok := m["cooldown"].(int); ok {
		p.Cooldown = v
	}

	if v, ok := m["dimensions"].(map[string]interface{}); ok {
		dimensions := make([]*spotinst.AwsGroupScalingPolicyDimension, 0, len(v))
		for i, k := range v {
			dimensions = append(dimensions, &spotinst.AwsGroupScalingPolicyDimension{
				Name:  i,
				Value: k.(string),
			})
		}

		p.Dimensions = dimensions
	}

	log.Printf("[DEBUG] AwsGroup scaling policy configuration: %#v\n", p)
	return p
}

// expandAwsGroupScheduledTask expands the Scheduled Task block.
func expandAwsGroupScheduledTask(m map[string]interface{}) *spotinst.AwsGroupScheduledTask {
	t := &spotinst.AwsGroupScheduledTask{}

	if v, ok := m["task_type"].(string); ok && v != "" {
		t.TaskType = v
	}

	if v, ok := m["frequency"].(string); ok && v != "" {
		t.Frequency = v
	}

	if v, ok := m["cron_expression"].(string); ok && v != "" {
		t.CronExpression = v
	}

	if v, ok := m["scale_target_capacity"].(int); ok && v >= 0 {
		t.ScaleTargetCapacity = v
	}

	if v, ok := m["scale_min_capacity"].(int); ok && v >= 0 {
		t.ScaleMinCapacity = v
	}

	if v, ok := m["scale_max_capacity"].(int); ok && v >= 0 {
		t.ScaleMaxCapacity = v
	}

	log.Printf("[DEBUG] AwsGroup scheduled task configuration: %#v\n", t)
	return t
}

// expandAwsGroupAvailabilityZone expands the Availability Zone block.
func expandAwsGroupAvailabilityZone(m map[string]interface{}) *spotinst.AwsGroupComputeAvailabilityZone {
	z := &spotinst.AwsGroupComputeAvailabilityZone{}

	if v, ok := m["name"].(string); ok && v != "" {
		z.Name = v
	}

	if v, ok := m["subnet_id"].(string); ok && v != "" {
		z.SubnetID = v
	}

	log.Printf("[DEBUG] AwsGroup availability zone configuration: %#v\n", z)
	return z
}

// expandAwsGroupNetworkInterface expands the Elastic Network Interface block.
func expandAwsGroupNetworkInterface(m map[string]interface{}) *spotinst.AwsGroupComputeNetworkInterface {
	i := &spotinst.AwsGroupComputeNetworkInterface{}

	if v, ok := m["network_interface_id"].(string); ok && v != "" {
		i.ID = v
	}

	if v, ok := m["description"].(string); ok && v != "" {
		i.Description = v
	}

	if v, ok := m["device_index"].(int); ok && v >= 0 {
		i.DeviceIndex = v
	}

	if v, ok := m["secondary_private_ip_address_count"].(int); ok && v >= 0 {
		i.SecondaryPrivateIPAddressCount = v
	}

	if v, ok := m["associate_public_ip_address"].(bool); ok {
		i.AssociatePublicIPAddress = v
	}

	if v, ok := m["delete_on_termination"].(bool); ok {
		i.DeleteOnTermination = v
	}

	if v, ok := m["private_ip_address"].(string); ok && v != "" {
		i.PrivateIPAddress = v
	}

	if v, ok := m["subnet_id"].(string); ok && v != "" {
		i.SubnetID = v
	}

	if v, ok := m["security_group_ids"].([]interface{}); ok {
		sids := make([]string, len(v))
		for i, j := range v {
			sids[i] = j.(string)
		}
		i.SecurityGroupsIDs = sids
	}

	log.Printf("[DEBUG] AwsGroup network interface configuration: %#v\n", i)
	return i
}

// expandAwsGroupEphemeralBlockDevice expands the Ephemeral Block Device block.
func expandAwsGroupEphemeralBlockDevice(m map[string]interface{}) *spotinst.AwsGroupComputeBlockDevice {
	b := &spotinst.AwsGroupComputeBlockDevice{}

	if v, ok := m["device_name"].(string); ok && v != "" {
		b.DeviceName = v
	}

	if v, ok := m["virtual_name"].(string); ok && v != "" {
		b.VirtualName = v
	}

	log.Printf("[DEBUG] AwsGroup ephemeral block device configuration: %#v\n", b)
	return b
}

// expandAwsGroupEBSBlockDevice expands the EBS Block Device block.
func expandAwsGroupEBSBlockDevice(m map[string]interface{}) *spotinst.AwsGroupComputeBlockDevice {
	b := &spotinst.AwsGroupComputeBlockDevice{EBS: &spotinst.AwsGroupComputeEBS{}}

	if v, ok := m["device_name"].(string); ok && v != "" {
		b.DeviceName = v
	}

	if v, ok := m["delete_on_termination"].(bool); ok {
		b.EBS.DeleteOnTermination = v
	}

	if v, ok := m["encrypted"].(bool); ok {
		b.EBS.Encrypted = v
	}

	if v, ok := m["snapshot_id"].(string); ok && v != "" {
		b.EBS.SnapshotID = v
	}

	if v, ok := m["volume_type"].(string); ok && v != "" {
		b.EBS.VolumeType = v
	}

	if v, ok := m["volume_size"].(int); ok && v >= 0 {
		b.EBS.VolumeSize = v
	}

	if v, ok := m["iops"].(int); ok && v >= 0 {
		b.EBS.IOPS = v
	}

	log.Printf("[DEBUG] AwsGroup EBS block device configuration: %#v\n", b)
	return b
}

// expandAwsGroupLaunchSpecification expands the launch Specification block.
func expandAwsGroupLaunchSpecification(m map[string]interface{}) *spotinst.AwsGroupComputeLaunchSpecification {
	lc := &spotinst.AwsGroupComputeLaunchSpecification{}

	if v, ok := m["monitoring"].(bool); ok {
		lc.Monitoring = v
	}

	if v, ok := m["image_id"].(string); ok && v != "" {
		lc.ImageID = v
	}

	if v, ok := m["key_pair"].(string); ok && v != "" {
		lc.KeyPair = v
	}

	if v, ok := m["health_check_type"].(string); ok && v != "" {
		lc.HealthCheckType = v
	}

	if v, ok := m["health_check_grace_period"].(int); ok && v >= 0 {
		lc.HealthCheckGracePeriod = v
	}

	if v, ok := m["iam_instance_profile"].(string); ok && v != "" {
		lc.IamRole = &spotinst.AwsGroupComputeIamInstanceProfile{Arn: v}
	}

	if v, ok := m["user_data"].(string); ok && v != "" {
		lc.UserData = base64.StdEncoding.EncodeToString([]byte(v))
	}

	if v, ok := m["security_group_ids"].([]interface{}); ok {
		sids := make([]string, len(v))
		for i, j := range v {
			sids[i] = j.(string)
		}
		lc.SecurityGroupIDs = sids
	}

	if v, ok := m["load_balancer_names"].([]interface{}); ok {
		elbs := make([]string, len(v))
		for i, j := range v {
			elbs[i] = j.(string)
		}
		lc.LoadBalancerNames = elbs
	}

	log.Printf("[DEBUG] AwsGroup launch specification configuration: %#v\n", lc)
	return lc
}

// expandAwsGroupRancherIntegration expands the Rancher Integration block.
func expandAwsGroupRancherIntegration(m map[string]interface{}) *spotinst.AwsGroupRancherIntegration {
	i := &spotinst.AwsGroupRancherIntegration{}

	if v, ok := m["master_host"].(string); ok && v != "" {
		i.MasterHost = v
	}

	if v, ok := m["access_key"].(string); ok && v != "" {
		i.AccessKey = v
	}

	if v, ok := m["secret_key"].(string); ok && v != "" {
		i.SecretKey = v
	}

	return i
}

// expandAwsGroupElasticBeanstalkIntegration expands the Elastic Beanstalk Integration block.
func expandAwsGroupElasticBeanstalkIntegration(m map[string]interface{}) *spotinst.AwsGroupElasticBeanstalkIntegration {
	i := &spotinst.AwsGroupElasticBeanstalkIntegration{}

	if v, ok := m["environment_id"].(string); ok && v != "" {
		i.EnvironmentID = v
	}

	return i
}

// expandAwsGroupNirmataIntegration expands the Nirmata Integration block.
func expandAwsGroupNirmataIntegration(m map[string]interface{}) *spotinst.AwsGroupNirmataIntegration {
	i := &spotinst.AwsGroupNirmataIntegration{}

	if v, ok := m["api_key"].(string); ok && v != "" {
		i.APIKey = v
	}

	return i
}

// tagsToMap turns the list of tags into a map.
func tagsToMap(ts []*spotinst.AwsGroupComputeTag) map[string]string {
	result := make(map[string]string)
	for _, t := range ts {
		result[t.Key] = t.Value
	}

	return result
}
