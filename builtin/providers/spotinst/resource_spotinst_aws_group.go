package spotinst

import (
	"bytes"
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"log"
	"regexp"
	"strings"

	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/spotinst/spotinst-sdk-go/spotinst"
	"github.com/spotinst/spotinst-sdk-go/spotinst/util/stringutil"
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
			},

			"description": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"capacity": &schema.Schema{
				Type:     schema.TypeSet,
				Required: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"target": &schema.Schema{
							Type:     schema.TypeInt,
							Required: true,
						},

						"minimum": &schema.Schema{
							Type:     schema.TypeInt,
							Optional: true,
							Computed: true,
						},

						"maximum": &schema.Schema{
							Type:     schema.TypeInt,
							Optional: true,
							Computed: true,
						},

						"unit": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							Computed: true,
						},
					},
				},
				Set: hashAwsGroupCapacity,
			},

			"strategy": &schema.Schema{
				Type:     schema.TypeSet,
				Required: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"risk": &schema.Schema{
							Type:     schema.TypeFloat,
							Optional: true,
						},

						"ondemand_count": &schema.Schema{
							Type:     schema.TypeInt,
							Optional: true,
						},

						"availability_vs_cost": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							Computed: true,
						},

						"draining_timeout": &schema.Schema{
							Type:     schema.TypeInt,
							Optional: true,
							Computed: true,
						},

						"utilize_reserved_instances": &schema.Schema{
							Type:     schema.TypeBool,
							Optional: true,
							Computed: true,
						},

						"fallback_to_ondemand": &schema.Schema{
							Type:     schema.TypeBool,
							Optional: true,
							Computed: true,
						},
					},
				},
				Set: hashAwsGroupStrategy,
			},

			"scheduled_task": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"task_type": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},

						"frequency": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},

						"cron_expression": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},

						"scale_target_capacity": &schema.Schema{
							Type:     schema.TypeInt,
							Optional: true,
						},

						"scale_min_capacity": &schema.Schema{
							Type:     schema.TypeInt,
							Optional: true,
						},

						"scale_max_capacity": &schema.Schema{
							Type:     schema.TypeInt,
							Optional: true,
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
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"ondemand": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},

						"spot": &schema.Schema{
							Type:     schema.TypeList,
							Required: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},
					},
				},
			},

			"signal": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},

			"availability_zone": &schema.Schema{
				Type:          schema.TypeSet,
				Optional:      true,
				ConflictsWith: []string{"availability_zones"},
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},

						"subnet_id": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
					},
				},
			},

			"availability_zones": &schema.Schema{
				Type:          schema.TypeList,
				Optional:      true,
				Elem:          &schema.Schema{Type: schema.TypeString},
				ConflictsWith: []string{"availability_zone"},
			},

			"hot_ebs_volume": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"device_name": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},

						"volume_ids": &schema.Schema{
							Type:     schema.TypeList,
							Required: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},
					},
				},
			},

			"load_balancer": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},

						"arn": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},

						"type": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
				Set: hashAwsGroupLoadBalancer,
			},

			"launch_specification": &schema.Schema{
				Type:     schema.TypeSet,
				Required: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"load_balancer_names": &schema.Schema{
							Type:     schema.TypeList,
							Optional: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},

						"monitoring": &schema.Schema{
							Type:     schema.TypeBool,
							Optional: true,
							Default:  false,
						},

						"ebs_optimized": &schema.Schema{
							Type:     schema.TypeBool,
							Optional: true,
							Computed: true,
						},

						"image_id": &schema.Schema{
							Type:          schema.TypeString,
							Optional:      true,
							ConflictsWith: []string{"image_id"},
						},

						"key_pair": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},

						"health_check_type": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},

						"health_check_grace_period": &schema.Schema{
							Type:     schema.TypeInt,
							Optional: true,
						},

						"security_group_ids": &schema.Schema{
							Type:     schema.TypeList,
							Required: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},

						"user_data": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
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
							Deprecated: "Attribute iam_role is deprecated. Use iam_instance_profile instead",
						},

						"iam_instance_profile": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
					},
				},
			},

			"image_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},

			"elastic_ips": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},

			"tags": &schema.Schema{
				Type:     schema.TypeMap,
				Optional: true,
			},

			"ebs_block_device": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"delete_on_termination": &schema.Schema{
							Type:     schema.TypeBool,
							Optional: true,
							Computed: true,
						},

						"device_name": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},

						"encrypted": &schema.Schema{
							Type:     schema.TypeBool,
							Optional: true,
							Computed: true,
						},

						"iops": &schema.Schema{
							Type:     schema.TypeInt,
							Optional: true,
						},

						"snapshot_id": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},

						"volume_size": &schema.Schema{
							Type:     schema.TypeInt,
							Optional: true,
						},

						"volume_type": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							Computed: true,
						},
					},
				},
				Set: hashAwsGroupEBSBlockDevice,
			},

			"ephemeral_block_device": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"device_name": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},

						"virtual_name": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},

			"network_interface": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"description": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},

						"device_index": &schema.Schema{
							Type:     schema.TypeInt,
							Required: true,
						},

						"secondary_private_ip_address_count": &schema.Schema{
							Type:     schema.TypeInt,
							Optional: true,
						},

						"associate_public_ip_address": &schema.Schema{
							Type:     schema.TypeBool,
							Optional: true,
						},

						"delete_on_termination": &schema.Schema{
							Type:     schema.TypeBool,
							Optional: true,
							Computed: true,
						},

						"security_group_ids": &schema.Schema{
							Type:     schema.TypeList,
							Optional: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},

						"network_interface_id": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},

						"private_ip_address": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},

						"subnet_id": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
					},
				},
			},

			"scaling_up_policy": scalingPolicySchema(),

			"scaling_down_policy": scalingPolicySchema(),

			"rancher_integration": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"master_host": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},

						"access_key": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},

						"secret_key": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},

			"elastic_beanstalk_integration": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"environment_id": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},

			"ec2_container_service_integration": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"cluster_name": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},

			"kubernetes_integration": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"api_server": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},

						"token": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},

			"mesosphere_integration": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"api_server": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
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
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"policy_name": &schema.Schema{
					Type:     schema.TypeString,
					Required: true,
				},

				"metric_name": &schema.Schema{
					Type:     schema.TypeString,
					Required: true,
				},

				"statistic": &schema.Schema{
					Type:     schema.TypeString,
					Required: true,
				},

				"unit": &schema.Schema{
					Type:     schema.TypeString,
					Required: true,
				},

				"threshold": &schema.Schema{
					Type:     schema.TypeFloat,
					Required: true,
				},

				"adjustment": &schema.Schema{
					Type:     schema.TypeInt,
					Optional: true,
				},

				"min_target_capacity": &schema.Schema{
					Type:     schema.TypeInt,
					Optional: true,
				},

				"max_target_capacity": &schema.Schema{
					Type:     schema.TypeInt,
					Optional: true,
				},

				"namespace": &schema.Schema{
					Type:     schema.TypeString,
					Required: true,
				},

				"operator": &schema.Schema{
					Type:     schema.TypeString,
					Optional: true,
					Computed: true,
				},

				"evaluation_periods": &schema.Schema{
					Type:     schema.TypeInt,
					Required: true,
				},

				"period": &schema.Schema{
					Type:     schema.TypeInt,
					Required: true,
				},

				"cooldown": &schema.Schema{
					Type:     schema.TypeInt,
					Required: true,
				},

				"dimensions": &schema.Schema{
					Type:     schema.TypeMap,
					Optional: true,
				},
			},
		},
		Set: hashAwsGroupScalingPolicy,
	}
}

func resourceSpotinstAwsGroupCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*spotinst.Client)
	newAwsGroup, err := buildAwsGroupOpts(d, meta)
	if err != nil {
		return err
	}
	log.Printf("[DEBUG] AwsGroup create configuration: %s\n", stringutil.Stringify(newAwsGroup))
	input := &spotinst.CreateAwsGroupInput{Group: newAwsGroup}
	resp, err := client.AwsGroupService.Create(input)
	if err != nil {
		return fmt.Errorf("Error creating group: %s", err)
	}
	d.SetId(spotinst.StringValue(resp.Group.ID))
	log.Printf("[INFO] AwsGroup created successfully: %s\n", d.Id())
	return resourceSpotinstAwsGroupRead(d, meta)
}

func resourceSpotinstAwsGroupRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*spotinst.Client)
	input := &spotinst.ReadAwsGroupInput{ID: spotinst.String(d.Id())}
	resp, err := client.AwsGroupService.Read(input)
	if err != nil {
		return fmt.Errorf("Error retrieving group: %s", err)
	}
	if g := resp.Group; g != nil {
		d.Set("name", g.Name)
		d.Set("description", g.Description)
		d.Set("product", g.Compute.Product)
		d.Set("tags", tagsToMap(g.Compute.LaunchSpecification.Tags))
		d.Set("elastic_ips", g.Compute.ElasticIPs)

		// Set capacity.
		if g.Capacity != nil {
			if err := d.Set("capacity", flattenAwsGroupCapacity(g.Capacity)); err != nil {
				return fmt.Errorf("Error setting capacity onfiguration: %#v", err)
			}
		}

		// Set strategy.
		if g.Strategy != nil {
			if err := d.Set("strategy", flattenAwsGroupStrategy(g.Strategy)); err != nil {
				return fmt.Errorf("Error setting strategy configuration: %#v", err)
			}
		}

		// Set signals.
		if g.Strategy.Signals != nil {
			if err := d.Set("signal", flattenAwsGroupSignals(g.Strategy.Signals)); err != nil {
				return fmt.Errorf("Error setting signals configuration: %#v", err)
			}
		}

		// Set scaling up policies.
		if g.Scaling.Up != nil {
			if err := d.Set("scaling_up_policy", flattenAwsGroupScalingPolicies(g.Scaling.Up)); err != nil {
				return fmt.Errorf("Error setting scaling up policies configuration: %#v", err)
			}
		}

		// Set scaling down policies.
		if g.Scaling.Down != nil {
			if err := d.Set("scaling_down_policy", flattenAwsGroupScalingPolicies(g.Scaling.Down)); err != nil {
				return fmt.Errorf("Error setting scaling down policies configuration: %#v", err)
			}
		}

		// Set scheduled tasks.
		if g.Scheduling.Tasks != nil {
			if err := d.Set("scheduled_task", flattenAwsGroupScheduledTasks(g.Scheduling.Tasks)); err != nil {
				return fmt.Errorf("Error setting scheduled tasks configuration: %#v", err)
			}
		}

		// Set launch specification.
		if g.Compute.LaunchSpecification != nil {
			imageIDSetInLaunchSpec := true
			if v, ok := d.GetOk("image_id"); ok && v != "" {
				imageIDSetInLaunchSpec = false
			}
			if err := d.Set("launch_specification", flattenAwsGroupLaunchSpecification(g.Compute.LaunchSpecification, imageIDSetInLaunchSpec)); err != nil {
				return fmt.Errorf("Error setting launch specification configuration: %#v", err)
			}
		}

		// Set image ID.
		if g.Compute.LaunchSpecification.ImageID != nil {
			if d.Get("image_id") != nil && d.Get("image_id") != "" {
				d.Set("image_id", g.Compute.LaunchSpecification.ImageID)
			}
		}

		// Set load balancers.
		if g.Compute.LaunchSpecification.LoadBalancersConfig != nil {
			if err := d.Set("load_balancer", flattenAwsGroupLoadBalancers(g.Compute.LaunchSpecification.LoadBalancersConfig.LoadBalancers)); err != nil {
				return fmt.Errorf("Error setting load balancers configuration: %#v", err)
			}
		}

		// Set EBS volume pool.
		if g.Compute.EBSVolumePool != nil {
			if err := d.Set("hot_ebs_volume", flattenAwsGroupEBSVolumePool(g.Compute.EBSVolumePool)); err != nil {
				return fmt.Errorf("Error setting EBS volume pool configuration: %#v", err)
			}
		}

		// Set network interfaces.
		if g.Compute.LaunchSpecification.NetworkInterfaces != nil {
			if err := d.Set("network_interface", flattenAwsGroupNetworkInterfaces(g.Compute.LaunchSpecification.NetworkInterfaces)); err != nil {
				return fmt.Errorf("Error setting network interfaces configuration: %#v", err)
			}
		}

		// Set block devices.
		if g.Compute.LaunchSpecification.BlockDevices != nil {
			if err := d.Set("ebs_block_device", flattenAwsGroupEBSBlockDevices(g.Compute.LaunchSpecification.BlockDevices)); err != nil {
				return fmt.Errorf("Error setting EBS block devices configuration: %#v", err)
			}
			if err := d.Set("ephemeral_block_device", flattenAwsGroupEphemeralBlockDevices(g.Compute.LaunchSpecification.BlockDevices)); err != nil {
				return fmt.Errorf("Error setting Ephemeral block devices configuration: %#v", err)
			}
		}

		// Set Rancher integration.
		if g.Integration.Rancher != nil {
			if err := d.Set("rancher_integration", flattenAwsGroupRancherIntegration(g.Integration.Rancher)); err != nil {
				return fmt.Errorf("Error setting Rancher configuration: %#v", err)
			}
		}

		// Set Elastic Beanstalk integration.
		if g.Integration.ElasticBeanstalk != nil {
			if err := d.Set("elastic_beanstalk_integration", flattenAwsGroupElasticBeanstalkIntegration(g.Integration.ElasticBeanstalk)); err != nil {
				return fmt.Errorf("Error setting Elastic Beanstalk configuration: %#v", err)
			}
		}

		// Set EC2 Container Service integration.
		if g.Integration.EC2ContainerService != nil {
			if err := d.Set("ec2_container_service_integration", flattenAwsGroupEC2ContainerServiceIntegration(g.Integration.EC2ContainerService)); err != nil {
				return fmt.Errorf("Error setting EC2 Container Service configuration: %#v", err)
			}
		}

		// Set Kubernetes integration.
		if g.Integration.Kubernetes != nil {
			if err := d.Set("kubernetes_integration", flattenAwsGroupKubernetesIntegration(g.Integration.Kubernetes)); err != nil {
				return fmt.Errorf("Error setting Kubernetes configuration: %#v", err)
			}
		}

		// Set Mesosphere integration.
		if g.Integration.Mesosphere != nil {
			if err := d.Set("mesosphere_integration", flattenAwsGroupMesosphereIntegration(g.Integration.Mesosphere)); err != nil {
				return fmt.Errorf("Error setting Mesosphere configuration: %#v", err)
			}
		}
	} else {
		d.SetId("")
	}
	return nil
}

func resourceSpotinstAwsGroupUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*spotinst.Client)
	group := &spotinst.AwsGroup{ID: spotinst.String(d.Id())}
	update := false

	if d.HasChange("name") {
		group.Name = spotinst.String(d.Get("name").(string))
		update = true
	}

	if d.HasChange("description") {
		group.Description = spotinst.String(d.Get("description").(string))
		update = true
	}

	if d.HasChange("capacity") {
		if v, ok := d.GetOk("capacity"); ok {
			if capacity, err := expandAwsGroupCapacity(v); err != nil {
				return err
			} else {
				group.Capacity = capacity
				update = true
			}
		}
	}

	if d.HasChange("strategy") {
		if v, ok := d.GetOk("strategy"); ok {
			if strategy, err := expandAwsGroupStrategy(v); err != nil {
				return err
			} else {
				group.Strategy = strategy
				if v, ok := d.GetOk("signal"); ok {
					if signals, err := expandAwsGroupSignals(v); err != nil {
						return err
					} else {
						group.Strategy.Signals = signals
					}
				}
				update = true
			}
		}
	}

	if d.HasChange("launch_specification") {
		if v, ok := d.GetOk("launch_specification"); ok {
			lc, err := expandAwsGroupLaunchSpecification(v)
			if err != nil {
				return err
			}
			if group.Compute == nil {
				group.Compute = &spotinst.AwsGroupCompute{}
			}
			group.Compute.LaunchSpecification = lc
			update = true
		}
	}

	if d.HasChange("image_id") {
		if d.Get("image_id") != nil && d.Get("image_id") != "" {
			if group.Compute == nil {
				group.Compute = &spotinst.AwsGroupCompute{}
			}
			if group.Compute.LaunchSpecification == nil {
				group.Compute.LaunchSpecification = &spotinst.AwsGroupComputeLaunchSpecification{}
			}
			group.Compute.LaunchSpecification.ImageID = spotinst.String(d.Get("image_id").(string))
			update = true
		}
	}

	if d.HasChange("load_balancer") {
		if v, ok := d.GetOk("load_balancer"); ok {
			if lbs, err := expandAwsGroupLoadBalancer(v); err != nil {
				return err
			} else {
				if group.Compute == nil {
					group.Compute = &spotinst.AwsGroupCompute{}
				}
				if group.Compute.LaunchSpecification == nil {
					group.Compute.LaunchSpecification = &spotinst.AwsGroupComputeLaunchSpecification{}
				}
				if group.Compute.LaunchSpecification.LoadBalancersConfig == nil {
					group.Compute.LaunchSpecification.LoadBalancersConfig = &spotinst.AwsGroupComputeLoadBalancersConfig{}
					group.Compute.LaunchSpecification.LoadBalancersConfig.LoadBalancers = lbs
					update = true
				}
			}
		}
	}

	var blockDevicesExpanded bool

	if d.HasChange("ebs_block_device") {
		if v, ok := d.GetOk("ebs_block_device"); ok {
			if devices, err := expandAwsGroupEBSBlockDevices(v); err != nil {
				return err
			} else {
				if group.Compute == nil {
					group.Compute = &spotinst.AwsGroupCompute{}
				}
				if group.Compute.LaunchSpecification == nil {
					group.Compute.LaunchSpecification = &spotinst.AwsGroupComputeLaunchSpecification{}
				}
				if len(group.Compute.LaunchSpecification.BlockDevices) > 0 {
					group.Compute.LaunchSpecification.BlockDevices = append(group.Compute.LaunchSpecification.BlockDevices, devices...)
				} else {
					if v, ok := d.GetOk("ephemeral_block_device"); ok {
						if ephemeral, err := expandAwsGroupEphemeralBlockDevices(v); err != nil {
							return err
						} else {
							devices = append(devices, ephemeral...)
							blockDevicesExpanded = true
						}
					}
					group.Compute.LaunchSpecification.BlockDevices = devices
				}
				update = true
			}
		}
	}

	if d.HasChange("ephemeral_block_device") && !blockDevicesExpanded {
		if v, ok := d.GetOk("ephemeral_block_device"); ok {
			if devices, err := expandAwsGroupEphemeralBlockDevices(v); err != nil {
				return err
			} else {
				if group.Compute == nil {
					group.Compute = &spotinst.AwsGroupCompute{}
				}
				if group.Compute.LaunchSpecification == nil {
					group.Compute.LaunchSpecification = &spotinst.AwsGroupComputeLaunchSpecification{}
				}
				if len(group.Compute.LaunchSpecification.BlockDevices) > 0 {
					group.Compute.LaunchSpecification.BlockDevices = append(group.Compute.LaunchSpecification.BlockDevices, devices...)
				} else {
					if v, ok := d.GetOk("ebs_block_device"); ok {
						if ebs, err := expandAwsGroupEBSBlockDevices(v); err != nil {
							return err
						} else {
							devices = append(devices, ebs...)
						}
					}
					group.Compute.LaunchSpecification.BlockDevices = devices
				}
				update = true
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
				}
				if group.Compute.LaunchSpecification == nil {
					group.Compute.LaunchSpecification = &spotinst.AwsGroupComputeLaunchSpecification{}
				}
				group.Compute.LaunchSpecification.NetworkInterfaces = interfaces
				update = true
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
				update = true
			}
		}
	}

	if d.HasChange("availability_zones") {
		if v, ok := d.GetOk("availability_zones"); ok {
			if zones, err := expandAwsGroupAvailabilityZonesSlice(v); err != nil {
				return err
			} else {
				if group.Compute == nil {
					group.Compute = &spotinst.AwsGroupCompute{}
				}
				group.Compute.AvailabilityZones = zones
				update = true
			}
		}
	}

	if d.HasChange("hot_ebs_volume") {
		if v, ok := d.GetOk("hot_ebs_volume"); ok {
			if ebsVolumePool, err := expandAwsGroupEBSVolumePool(v); err != nil {
				return err
			} else {
				if group.Compute == nil {
					group.Compute = &spotinst.AwsGroupCompute{}
				}
				group.Compute.EBSVolumePool = ebsVolumePool
				update = true
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
				update = true
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
				update = true
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
				}
				if group.Compute.LaunchSpecification == nil {
					group.Compute.LaunchSpecification = &spotinst.AwsGroupComputeLaunchSpecification{}
				}
				group.Compute.LaunchSpecification.Tags = tags
				update = true
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
				update = true
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
				update = true
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
				update = true
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
				update = true
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
				update = true
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
				update = true
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
				update = true
			}
		}
	}

	if d.HasChange("kubernetes_integration") {
		if v, ok := d.GetOk("kubernetes_integration"); ok {
			if integration, err := expandAwsGroupKubernetesIntegration(v); err != nil {
				return err
			} else {
				if group.Integration == nil {
					group.Integration = &spotinst.AwsGroupIntegration{}
				}
				group.Integration.Kubernetes = integration
				update = true
			}
		}
	}

	if d.HasChange("mesosphere_integration") {
		if v, ok := d.GetOk("mesosphere_integration"); ok {
			if integration, err := expandAwsGroupMesosphereIntegration(v); err != nil {
				return err
			} else {
				if group.Integration == nil {
					group.Integration = &spotinst.AwsGroupIntegration{}
				}
				group.Integration.Mesosphere = integration
				update = true
			}
		}
	}

	if update {
		log.Printf("[DEBUG] AwsGroup update configuration: %s\n", stringutil.Stringify(group))
		input := &spotinst.UpdateAwsGroupInput{Group: group}
		if _, err := client.AwsGroupService.Update(input); err != nil {
			return fmt.Errorf("Error updating group %s: %s", d.Id(), err)
		}
	}

	return resourceSpotinstAwsGroupRead(d, meta)
}

func resourceSpotinstAwsGroupDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*spotinst.Client)
	log.Printf("[INFO] Deleting group: %s\n", d.Id())
	input := &spotinst.DeleteAwsGroupInput{ID: spotinst.String(d.Id())}
	if _, err := client.AwsGroupService.Delete(input); err != nil {
		return fmt.Errorf("Error deleting group: %s", err)
	}
	d.SetId("")
	return nil
}

func flattenAwsGroupCapacity(capacity *spotinst.AwsGroupCapacity) []interface{} {
	result := make(map[string]interface{})
	result["target"] = spotinst.IntValue(capacity.Target)
	result["minimum"] = spotinst.IntValue(capacity.Minimum)
	result["maximum"] = spotinst.IntValue(capacity.Maximum)
	result["unit"] = spotinst.StringValue(capacity.Unit)
	return []interface{}{result}
}

func flattenAwsGroupStrategy(strategy *spotinst.AwsGroupStrategy) []interface{} {
	result := make(map[string]interface{})
	result["risk"] = spotinst.Float64Value(strategy.Risk)
	result["ondemand_count"] = spotinst.IntValue(strategy.OnDemandCount)
	result["availability_vs_cost"] = spotinst.StringValue(strategy.AvailabilityVsCost)
	result["draining_timeout"] = spotinst.IntValue(strategy.DrainingTimeout)
	result["utilize_reserved_instances"] = spotinst.BoolValue(strategy.UtilizeReservedInstances)
	result["fallback_to_ondemand"] = spotinst.BoolValue(strategy.FallbackToOnDemand)
	return []interface{}{result}
}

func flattenAwsGroupLaunchSpecification(lspec *spotinst.AwsGroupComputeLaunchSpecification, includeImageID bool) []interface{} {
	result := make(map[string]interface{})
	result["health_check_grace_period"] = spotinst.IntValue(lspec.HealthCheckGracePeriod)
	result["health_check_type"] = spotinst.StringValue(lspec.HealthCheckType)
	if includeImageID {
		result["image_id"] = spotinst.StringValue(lspec.ImageID)
	}
	result["key_pair"] = spotinst.StringValue(lspec.KeyPair)
	if lspec.UserData != nil && spotinst.StringValue(lspec.UserData) != "" {
		decodedUserData, _ := base64.StdEncoding.DecodeString(spotinst.StringValue(lspec.UserData))
		result["user_data"] = string(decodedUserData)
	} else {
		result["user_data"] = ""
	}
	result["monitoring"] = spotinst.BoolValue(lspec.Monitoring)
	result["ebs_optimized"] = spotinst.BoolValue(lspec.EBSOptimized)
	result["load_balancer_names"] = lspec.LoadBalancerNames
	result["security_group_ids"] = lspec.SecurityGroupIDs
	if lspec.IamInstanceProfile != nil {
		if lspec.IamInstanceProfile.Arn != nil {
			result["iam_instance_profile"] = spotinst.StringValue(lspec.IamInstanceProfile.Arn)
		} else {
			result["iam_instance_profile"] = spotinst.StringValue(lspec.IamInstanceProfile.Name)
		}
	}
	return []interface{}{result}
}

func flattenAwsGroupLoadBalancers(balancers []*spotinst.AwsGroupComputeLoadBalancer) []interface{} {
	result := make([]interface{}, 0, len(balancers))
	for _, b := range balancers {
		m := make(map[string]interface{})
		m["name"] = spotinst.StringValue(b.Name)
		m["arn"] = spotinst.StringValue(b.Arn)
		m["type"] = strings.ToLower(spotinst.StringValue(b.Type))
		result = append(result, m)
	}
	return result
}

func flattenAwsGroupEBSVolumePool(volumes []*spotinst.AwsGroupComputeEBSVolume) []interface{} {
	result := make([]interface{}, 0, len(volumes))
	for _, v := range volumes {
		m := make(map[string]interface{})
		m["device_name"] = spotinst.StringValue(v.DeviceName)
		m["volume_ids"] = v.VolumeIDs
		result = append(result, m)
	}
	return result
}

func flattenAwsGroupSignals(signals []*spotinst.AwsGroupStrategySignal) []interface{} {
	result := make([]interface{}, 0, len(signals))
	for _, s := range signals {
		m := make(map[string]interface{})
		m["name"] = strings.ToLower(spotinst.StringValue(s.Name))
		result = append(result, m)
	}
	return result
}

func flattenAwsGroupScheduledTasks(tasks []*spotinst.AwsGroupScheduledTask) []interface{} {
	result := make([]interface{}, 0, len(tasks))
	for _, t := range tasks {
		m := make(map[string]interface{})
		m["task_type"] = spotinst.StringValue(t.TaskType)
		m["cron_expression"] = spotinst.StringValue(t.CronExpression)
		m["frequency"] = spotinst.StringValue(t.Frequency)
		m["scale_target_capacity"] = spotinst.IntValue(t.ScaleTargetCapacity)
		m["scale_min_capacity"] = spotinst.IntValue(t.ScaleMinCapacity)
		m["scale_max_capacity"] = spotinst.IntValue(t.ScaleMaxCapacity)
		result = append(result, m)
	}
	return result
}

func flattenAwsGroupScalingPolicies(policies []*spotinst.AwsGroupScalingPolicy) []interface{} {
	result := make([]interface{}, 0, len(policies))
	for _, p := range policies {
		m := make(map[string]interface{})
		m["adjustment"] = spotinst.IntValue(p.Adjustment)
		m["cooldown"] = spotinst.IntValue(p.Cooldown)
		m["evaluation_periods"] = spotinst.IntValue(p.EvaluationPeriods)
		m["min_target_capacity"] = spotinst.IntValue(p.MinTargetCapacity)
		m["max_target_capacity"] = spotinst.IntValue(p.MaxTargetCapacity)
		m["metric_name"] = spotinst.StringValue(p.MetricName)
		m["namespace"] = spotinst.StringValue(p.Namespace)
		m["operator"] = spotinst.StringValue(p.Operator)
		m["period"] = spotinst.IntValue(p.Period)
		m["policy_name"] = spotinst.StringValue(p.PolicyName)
		m["statistic"] = spotinst.StringValue(p.Statistic)
		m["threshold"] = spotinst.Float64Value(p.Threshold)
		m["unit"] = spotinst.StringValue(p.Unit)
		if len(p.Dimensions) > 0 {
			flatDims := make(map[string]interface{})
			for _, d := range p.Dimensions {
				flatDims[spotinst.StringValue(d.Name)] = *d.Value
			}
			m["dimensions"] = flatDims
		}
		result = append(result, m)
	}
	return result
}

func flattenAwsGroupNetworkInterfaces(ifaces []*spotinst.AwsGroupComputeNetworkInterface) []interface{} {
	result := make([]interface{}, 0, len(ifaces))
	for _, iface := range ifaces {
		m := make(map[string]interface{})
		m["associate_public_ip_address"] = spotinst.BoolValue(iface.AssociatePublicIPAddress)
		m["delete_on_termination"] = spotinst.BoolValue(iface.DeleteOnTermination)
		m["description"] = spotinst.StringValue(iface.Description)
		m["device_index"] = spotinst.IntValue(iface.DeviceIndex)
		m["network_interface_id"] = spotinst.StringValue(iface.ID)
		m["private_ip_address"] = spotinst.StringValue(iface.PrivateIPAddress)
		m["secondary_private_ip_address_count"] = spotinst.IntValue(iface.SecondaryPrivateIPAddressCount)
		m["subnet_id"] = spotinst.StringValue(iface.SubnetID)
		m["security_group_ids"] = iface.SecurityGroupsIDs
		result = append(result, m)
	}
	return result
}

func flattenAwsGroupEBSBlockDevices(devices []*spotinst.AwsGroupComputeBlockDevice) []interface{} {
	result := make([]interface{}, 0, len(devices))
	for _, dev := range devices {
		if dev.EBS != nil {
			m := make(map[string]interface{})
			m["device_name"] = spotinst.StringValue(dev.DeviceName)
			m["delete_on_termination"] = spotinst.BoolValue(dev.EBS.DeleteOnTermination)
			m["encrypted"] = spotinst.BoolValue(dev.EBS.Encrypted)
			m["iops"] = spotinst.IntValue(dev.EBS.IOPS)
			m["snapshot_id"] = spotinst.StringValue(dev.EBS.SnapshotID)
			m["volume_type"] = spotinst.StringValue(dev.EBS.VolumeType)
			m["volume_size"] = spotinst.IntValue(dev.EBS.VolumeSize)
			result = append(result, m)
		}
	}
	return result
}

func flattenAwsGroupEphemeralBlockDevices(devices []*spotinst.AwsGroupComputeBlockDevice) []interface{} {
	result := make([]interface{}, 0, len(devices))
	for _, dev := range devices {
		if dev.EBS == nil {
			m := make(map[string]interface{})
			m["device_name"] = spotinst.StringValue(dev.DeviceName)
			m["virtual_name"] = spotinst.StringValue(dev.VirtualName)
			result = append(result, m)
		}
	}
	return result
}

func flattenAwsGroupRancherIntegration(integration *spotinst.AwsGroupRancherIntegration) []interface{} {
	result := make(map[string]interface{})
	result["master_host"] = spotinst.StringValue(integration.MasterHost)
	result["access_key"] = spotinst.StringValue(integration.AccessKey)
	result["secret_key"] = spotinst.StringValue(integration.SecretKey)
	return []interface{}{result}
}

func flattenAwsGroupElasticBeanstalkIntegration(integration *spotinst.AwsGroupElasticBeanstalkIntegration) []interface{} {
	result := make(map[string]interface{})
	result["environment_id"] = spotinst.StringValue(integration.EnvironmentID)
	return []interface{}{result}
}

func flattenAwsGroupEC2ContainerServiceIntegration(integration *spotinst.AwsGroupEC2ContainerServiceIntegration) []interface{} {
	result := make(map[string]interface{})
	result["cluster_name"] = spotinst.StringValue(integration.ClusterName)
	return []interface{}{result}
}

func flattenAwsGroupKubernetesIntegration(integration *spotinst.AwsGroupKubernetesIntegration) []interface{} {
	result := make(map[string]interface{})
	result["api_server"] = spotinst.StringValue(integration.Server)
	result["token"] = spotinst.StringValue(integration.Token)
	return []interface{}{result}
}

func flattenAwsGroupMesosphereIntegration(integration *spotinst.AwsGroupMesosphereIntegration) []interface{} {
	result := make(map[string]interface{})
	result["api_server"] = spotinst.StringValue(integration.Server)
	return []interface{}{result}
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

	if v, ok := d.GetOk("availability_zones"); ok {
		if zones, err := expandAwsGroupAvailabilityZonesSlice(v); err != nil {
			return nil, err
		} else {
			group.Compute.AvailabilityZones = zones
		}
	}

	if v, ok := d.GetOk("hot_ebs_volume"); ok {
		if ebsVolumePool, err := expandAwsGroupEBSVolumePool(v); err != nil {
			return nil, err
		} else {
			group.Compute.EBSVolumePool = ebsVolumePool
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

	if v, ok := d.GetOk("image_id"); ok {
		group.Compute.LaunchSpecification.ImageID = spotinst.String(v.(string))
	}

	if v, ok := d.GetOk("load_balancer"); ok {
		if lbs, err := expandAwsGroupLoadBalancer(v); err != nil {
			return nil, err
		} else {
			if group.Compute.LaunchSpecification.LoadBalancersConfig == nil {
				group.Compute.LaunchSpecification.LoadBalancersConfig = &spotinst.AwsGroupComputeLoadBalancersConfig{}
			}
			group.Compute.LaunchSpecification.LoadBalancersConfig.LoadBalancers = lbs
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
				group.Compute.LaunchSpecification.BlockDevices = append(group.Compute.LaunchSpecification.BlockDevices, devices...)
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

	if v, ok := d.GetOk("kubernetes_integration"); ok {
		if integration, err := expandAwsGroupKubernetesIntegration(v); err != nil {
			return nil, err
		} else {
			group.Integration.Kubernetes = integration
		}
	}

	return group, nil
}

// expandAwsGroupCapacity expands the Capacity block.
func expandAwsGroupCapacity(data interface{}) (*spotinst.AwsGroupCapacity, error) {
	list := data.(*schema.Set).List()
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

	if v, ok := m["unit"].(string); ok && v != "" {
		capacity.Unit = spotinst.String(v)
	}

	log.Printf("[DEBUG] AwsGroup capacity configuration: %s\n", stringutil.Stringify(capacity))
	return capacity, nil
}

// expandAwsGroupStrategy expands the Strategy block.
func expandAwsGroupStrategy(data interface{}) (*spotinst.AwsGroupStrategy, error) {
	list := data.(*schema.Set).List()
	m := list[0].(map[string]interface{})
	strategy := &spotinst.AwsGroupStrategy{}

	if v, ok := m["risk"].(float64); ok && v >= 0 {
		strategy.Risk = spotinst.Float64(v)
	}

	if v, ok := m["ondemand_count"].(int); ok && v >= 0 && spotinst.Float64Value(strategy.Risk) == 0 {
		strategy.OnDemandCount = spotinst.Int(v)
		strategy.Risk = nil
	}

	if v, ok := m["availability_vs_cost"].(string); ok && v != "" {
		strategy.AvailabilityVsCost = spotinst.String(v)
	}

	if v, ok := m["draining_timeout"].(int); ok && v > 0 {
		strategy.DrainingTimeout = spotinst.Int(v)
	}

	if v, ok := m["utilize_reserved_instances"].(bool); ok {
		strategy.UtilizeReservedInstances = spotinst.Bool(v)
	}

	if v, ok := m["fallback_to_ondemand"].(bool); ok {
		strategy.FallbackToOnDemand = spotinst.Bool(v)
	}

	log.Printf("[DEBUG] AwsGroup strategy configuration: %s\n", stringutil.Stringify(strategy))
	return strategy, nil
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

		if v, ok := m["operator"].(string); ok && v != "" {
			policy.Operator = spotinst.String(v)
		}

		if v, ok := m["period"].(int); ok && v > 0 {
			policy.Period = spotinst.Int(v)
		}

		if v, ok := m["evaluation_periods"].(int); ok && v > 0 {
			policy.EvaluationPeriods = spotinst.Int(v)
		}

		if v, ok := m["cooldown"].(int); ok && v > 0 {
			policy.Cooldown = spotinst.Int(v)
		}

		if v, ok := m["dimensions"]; ok {
			dimensions := expandAwsGroupScalingPolicyDimensions(v.(map[string]interface{}))
			policy.Dimensions = dimensions
		}

		if v, ok := m["namespace"].(string); ok && v != "" {
			log.Printf("[DEBUG] AwsGroup scaling policy configuration: %s\n", stringutil.Stringify(policy))
			policies = append(policies, policy)
		}
	}

	return policies, nil
}

func expandAwsGroupScalingPolicyDimensions(list map[string]interface{}) []*spotinst.AwsGroupScalingPolicyDimension {
	dimensions := make([]*spotinst.AwsGroupScalingPolicyDimension, 0, len(list))
	for name, val := range list {
		dimension := &spotinst.AwsGroupScalingPolicyDimension{
			Name:  spotinst.String(name),
			Value: spotinst.String(val.(string)),
		}
		log.Printf("[DEBUG] AwsGroup scaling policy dimension: %s\n", stringutil.Stringify(dimension))
		dimensions = append(dimensions, dimension)
	}
	return dimensions
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

		log.Printf("[DEBUG] AwsGroup scheduled task configuration: %s\n", stringutil.Stringify(task))
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

		log.Printf("[DEBUG] AwsGroup availability zone configuration: %s\n", stringutil.Stringify(zone))
		zones = append(zones, zone)
	}

	return zones, nil
}

// expandAwsGroupAvailabilityZonesSlice expands the Availability Zone block when provided as a slice.
func expandAwsGroupAvailabilityZonesSlice(data interface{}) ([]*spotinst.AwsGroupComputeAvailabilityZone, error) {
	list := data.([]interface{})
	zones := make([]*spotinst.AwsGroupComputeAvailabilityZone, 0, len(list))
	for _, str := range list {
		if s, ok := str.(string); ok {
			parts := strings.Split(s, ":")
			zone := &spotinst.AwsGroupComputeAvailabilityZone{}
			if len(parts) >= 1 && parts[0] != "" {
				zone.Name = spotinst.String(parts[0])
			}
			if len(parts) == 2 && parts[1] != "" {
				zone.SubnetID = spotinst.String(parts[1])
			}
			log.Printf("[DEBUG] AwsGroup availability zone configuration: %s\n", stringutil.Stringify(zone))
			zones = append(zones, zone)
		}
	}

	return zones, nil
}

// expandAwsGroupEBSVolumePool expands the EBS Volume Pool block.
func expandAwsGroupEBSVolumePool(data interface{}) ([]*spotinst.AwsGroupComputeEBSVolume, error) {
	list := data.(*schema.Set).List()
	volumes := make([]*spotinst.AwsGroupComputeEBSVolume, 0, len(list))
	for _, item := range list {
		m := item.(map[string]interface{})
		volume := &spotinst.AwsGroupComputeEBSVolume{}

		if v, ok := m["device_name"].(string); ok && v != "" {
			volume.DeviceName = spotinst.String(v)
		}

		if v, ok := m["volume_ids"].([]interface{}); ok {
			ids := make([]string, len(v))
			for i, j := range v {
				ids[i] = j.(string)
			}
			volume.VolumeIDs = ids
		}

		log.Printf("[DEBUG] AwsGroup EBS volume (pool) configuration: %s\n", stringutil.Stringify(volume))
		volumes = append(volumes, volume)
	}

	return volumes, nil
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

		log.Printf("[DEBUG] AwsGroup signal configuration: %s\n", stringutil.Stringify(signal))
		signals = append(signals, signal)
	}

	return signals, nil
}

// expandAwsGroupInstanceTypes expands the Instance Types block.
func expandAwsGroupInstanceTypes(data interface{}) (*spotinst.AwsGroupComputeInstanceType, error) {
	list := data.(*schema.Set).List()
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

	log.Printf("[DEBUG] AwsGroup instance types configuration: %s\n", stringutil.Stringify(types))
	return types, nil
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

		if v, ok := m["security_group_ids"].([]interface{}); ok {
			ids := make([]string, len(v))
			for i, j := range v {
				ids[i] = j.(string)
			}
			iface.SecurityGroupsIDs = ids
		}

		log.Printf("[DEBUG] AwsGroup network interface configuration: %s\n", stringutil.Stringify(iface))
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

		log.Printf("[DEBUG] AwsGroup ephemeral block device configuration: %s\n", stringutil.Stringify(device))
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

		log.Printf("[DEBUG] AwsGroup elastic block device configuration: %s\n", stringutil.Stringify(device))
		devices = append(devices, device)
	}

	return devices, nil
}

// iprofArnRE is a regular expression for matching IAM instance profile ARNs.
var iprofArnRE = regexp.MustCompile(`arn:aws:iam::\d{12}:instance-profile/?[a-zA-Z_0-9+=,.@\-_/]+`)

// expandAwsGroupLaunchSpecification expands the launch Specification block.
func expandAwsGroupLaunchSpecification(data interface{}) (*spotinst.AwsGroupComputeLaunchSpecification, error) {
	list := data.(*schema.Set).List()
	m := list[0].(map[string]interface{})
	lc := &spotinst.AwsGroupComputeLaunchSpecification{}

	if v, ok := m["monitoring"].(bool); ok {
		lc.Monitoring = spotinst.Bool(v)
	}

	if v, ok := m["ebs_optimized"].(bool); ok {
		lc.EBSOptimized = spotinst.Bool(v)
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
		iprof := &spotinst.AwsGroupComputeIamInstanceProfile{}
		if iprofArnRE.MatchString(v) {
			iprof.Arn = spotinst.String(v)
		} else {
			iprof.Name = spotinst.String(v)
		}
		lc.IamInstanceProfile = iprof
	}

	if v, ok := m["user_data"].(string); ok && v != "" {
		lc.UserData = spotinst.String(base64.StdEncoding.EncodeToString([]byte(v)))
	}

	if v, ok := m["security_group_ids"].([]interface{}); ok {
		ids := make([]string, len(v))
		for i, j := range v {
			ids[i] = j.(string)
		}
		lc.SecurityGroupIDs = ids
	}

	if v, ok := m["load_balancer_names"].([]interface{}); ok {
		var names []string
		for _, j := range v {
			if name, ok := j.(string); ok && name != "" {
				names = append(names, name)
			}
		}
		lc.LoadBalancerNames = names
	}

	log.Printf("[DEBUG] AwsGroup launch specification configuration: %s\n", stringutil.Stringify(lc))
	return lc, nil
}

// expandAwsGroupLoadBalancer expands the Load Balancer block.
func expandAwsGroupLoadBalancer(data interface{}) ([]*spotinst.AwsGroupComputeLoadBalancer, error) {
	list := data.(*schema.Set).List()
	lbs := make([]*spotinst.AwsGroupComputeLoadBalancer, 0, len(list))
	for _, item := range list {
		m := item.(map[string]interface{})
		lb := &spotinst.AwsGroupComputeLoadBalancer{}

		if v, ok := m["name"].(string); ok && v != "" {
			lb.Name = spotinst.String(v)
		}

		if v, ok := m["arn"].(string); ok && v != "" {
			lb.Arn = spotinst.String(v)
		}

		if v, ok := m["type"].(string); ok && v != "" {
			lb.Type = spotinst.String(strings.ToUpper(v))
		}

		log.Printf("[DEBUG] AwsGroup load balancer configuration: %s\n", stringutil.Stringify(lb))
		lbs = append(lbs, lb)
	}

	return lbs, nil
}

// expandAwsGroupRancherIntegration expands the Rancher Integration block.
func expandAwsGroupRancherIntegration(data interface{}) (*spotinst.AwsGroupRancherIntegration, error) {
	list := data.(*schema.Set).List()
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

	log.Printf("[DEBUG] AwsGroup Rancher integration configuration: %s\n", stringutil.Stringify(i))
	return i, nil
}

// expandAwsGroupElasticBeanstalkIntegration expands the Elastic Beanstalk Integration block.
func expandAwsGroupElasticBeanstalkIntegration(data interface{}) (*spotinst.AwsGroupElasticBeanstalkIntegration, error) {
	list := data.(*schema.Set).List()
	m := list[0].(map[string]interface{})
	i := &spotinst.AwsGroupElasticBeanstalkIntegration{}

	if v, ok := m["environment_id"].(string); ok && v != "" {
		i.EnvironmentID = spotinst.String(v)
	}

	log.Printf("[DEBUG] AwsGroup Elastic Beanstalk integration configuration:  %s\n", stringutil.Stringify(i))
	return i, nil
}

// expandAwsGroupEC2ContainerServiceIntegration expands the EC2 Container Service Integration block.
func expandAwsGroupEC2ContainerServiceIntegration(data interface{}) (*spotinst.AwsGroupEC2ContainerServiceIntegration, error) {
	list := data.(*schema.Set).List()
	m := list[0].(map[string]interface{})
	i := &spotinst.AwsGroupEC2ContainerServiceIntegration{}

	if v, ok := m["cluster_name"].(string); ok && v != "" {
		i.ClusterName = spotinst.String(v)
	}

	log.Printf("[DEBUG] AwsGroup ECS integration configuration:  %s\n", stringutil.Stringify(i))
	return i, nil
}

// expandAwsGroupKubernetesIntegration expands the Kubernetes Integration block.
func expandAwsGroupKubernetesIntegration(data interface{}) (*spotinst.AwsGroupKubernetesIntegration, error) {
	list := data.(*schema.Set).List()
	m := list[0].(map[string]interface{})
	i := &spotinst.AwsGroupKubernetesIntegration{}

	if v, ok := m["api_server"].(string); ok && v != "" {
		i.Server = spotinst.String(v)
	}

	if v, ok := m["token"].(string); ok && v != "" {
		i.Token = spotinst.String(v)
	}

	log.Printf("[DEBUG] AwsGroup Kubernetes integration configuration:  %s\n", stringutil.Stringify(i))
	return i, nil
}

// expandAwsGroupMesosphereIntegration expands the Mesosphere Integration block.
func expandAwsGroupMesosphereIntegration(data interface{}) (*spotinst.AwsGroupMesosphereIntegration, error) {
	list := data.(*schema.Set).List()
	m := list[0].(map[string]interface{})
	i := &spotinst.AwsGroupMesosphereIntegration{}

	if v, ok := m["api_server"].(string); ok && v != "" {
		i.Server = spotinst.String(v)
	}

	log.Printf("[DEBUG] AwsGroup Mesosphere integration configuration: %s\n", stringutil.Stringify(i))
	return i, nil
}

// expandAwsGroupElasticIPs expands the Elastic IPs block.
func expandAwsGroupElasticIPs(data interface{}) ([]string, error) {
	list := data.([]interface{})
	eips := make([]string, 0, len(list))
	for _, str := range list {
		if eip, ok := str.(string); ok {
			log.Printf("[DEBUG] AwsGroup elastic IP configuration: %s\n", stringutil.Stringify(eip))
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

		log.Printf("[DEBUG] AwsGroup tag configuration: %s\n", stringutil.Stringify(tag))
		tags = append(tags, tag)
	}

	return tags, nil
}

func hashAwsGroupCapacity(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})

	buf.WriteString(fmt.Sprintf("%d-", m["target"].(int)))
	buf.WriteString(fmt.Sprintf("%d-", m["minimum"].(int)))
	buf.WriteString(fmt.Sprintf("%d-", m["maximum"].(int)))

	return hashcode.String(buf.String())
}

func hashAwsGroupStrategy(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})

	buf.WriteString(fmt.Sprintf("%f-", m["risk"].(float64)))
	buf.WriteString(fmt.Sprintf("%d-", m["ondemand_count"].(int)))
	buf.WriteString(fmt.Sprintf("%t-", m["utilize_reserved_instances"].(bool)))
	buf.WriteString(fmt.Sprintf("%t-", m["fallback_to_ondemand"].(bool)))

	return hashcode.String(buf.String())
}

func hashAwsGroupLoadBalancer(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})

	buf.WriteString(fmt.Sprintf("%s-", m["name"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["type"].(string)))
	if v, ok := m["arn"].(string); ok && len(v) > 0 {
		buf.WriteString(fmt.Sprintf("%s-", v))
	}

	return hashcode.String(buf.String())
}

func hashAwsGroupEBSBlockDevice(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})

	buf.WriteString(fmt.Sprintf("%s-", m["device_name"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["snapshot_id"].(string)))
	buf.WriteString(fmt.Sprintf("%d-", m["volume_size"].(int)))
	buf.WriteString(fmt.Sprintf("%t-", m["delete_on_termination"].(bool)))
	buf.WriteString(fmt.Sprintf("%t-", m["encrypted"].(bool)))
	buf.WriteString(fmt.Sprintf("%d-", m["iops"].(int)))

	return hashcode.String(buf.String())
}

func hashAwsGroupScalingPolicy(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})

	buf.WriteString(fmt.Sprintf("%d-", m["adjustment"].(int)))
	buf.WriteString(fmt.Sprintf("%d-", m["cooldown"].(int)))
	buf.WriteString(fmt.Sprintf("%d-", m["evaluation_periods"].(int)))
	buf.WriteString(fmt.Sprintf("%s-", m["metric_name"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["namespace"].(string)))
	buf.WriteString(fmt.Sprintf("%d-", m["period"].(int)))
	buf.WriteString(fmt.Sprintf("%s-", m["policy_name"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["statistic"].(string)))
	buf.WriteString(fmt.Sprintf("%f-", m["threshold"].(float64)))
	buf.WriteString(fmt.Sprintf("%s-", m["unit"].(string)))
	buf.WriteString(fmt.Sprintf("%d-", m["min_target_capacity"].(int)))
	buf.WriteString(fmt.Sprintf("%d-", m["max_target_capacity"].(int)))

	// if v, ok := m["operator"].(string); ok && len(v) > 0 {
	// 	buf.WriteString(fmt.Sprintf("%s-", v))
	// }

	if d, ok := m["dimensions"]; ok {
		if len(d.(map[string]interface{})) > 0 {
			e := d.(map[string]interface{})
			for k, v := range e {
				buf.WriteString(fmt.Sprintf("%s:%s-", k, v.(string)))
			}
		}
	}

	return hashcode.String(buf.String())
}

// tagsToMap turns the list of tags into a map.
func tagsToMap(ts []*spotinst.AwsGroupComputeTag) map[string]string {
	result := make(map[string]string)
	for _, t := range ts {
		result[spotinst.StringValue(t.Key)] = spotinst.StringValue(t.Value)
	}
	return result
}
