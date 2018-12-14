package aws

import (
	"errors"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/elbv2"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
)

func resourceAwsLbTargetGroup() *schema.Resource {
	return &schema.Resource{
		// NLBs have restrictions on them at this time
		CustomizeDiff: resourceAwsLbTargetGroupCustomizeDiff,

		Create: resourceAwsLbTargetGroupCreate,
		Read:   resourceAwsLbTargetGroupRead,
		Update: resourceAwsLbTargetGroupUpdate,
		Delete: resourceAwsLbTargetGroupDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"arn_suffix": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"name": {
				Type:          schema.TypeString,
				Optional:      true,
				Computed:      true,
				ForceNew:      true,
				ConflictsWith: []string{"name_prefix"},
				ValidateFunc:  validateLbTargetGroupName,
			},
			"name_prefix": {
				Type:          schema.TypeString,
				Optional:      true,
				ForceNew:      true,
				ConflictsWith: []string{"name"},
				ValidateFunc:  validateLbTargetGroupNamePrefix,
			},

			"port": {
				Type:         schema.TypeInt,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.IntBetween(1, 65535),
			},

			"protocol": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
				ValidateFunc: validation.StringInSlice([]string{
					elbv2.ProtocolEnumHttp,
					elbv2.ProtocolEnumHttps,
					elbv2.ProtocolEnumTcp,
				}, true),
			},

			"vpc_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"deregistration_delay": {
				Type:         schema.TypeInt,
				Optional:     true,
				Default:      300,
				ValidateFunc: validation.IntBetween(0, 3600),
			},

			"slow_start": {
				Type:         schema.TypeInt,
				Optional:     true,
				Default:      0,
				ValidateFunc: validateSlowStart,
			},

			"proxy_protocol_v2": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},

			"target_type": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "instance",
				ForceNew: true,
			},

			"stickiness": {
				Type:     schema.TypeList,
				Optional: true,
				Computed: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"enabled": {
							Type:     schema.TypeBool,
							Optional: true,
							Default:  true,
						},
						"type": {
							Type:     schema.TypeString,
							Required: true,
							ValidateFunc: validation.StringInSlice([]string{
								"lb_cookie",
							}, false),
						},
						"cookie_duration": {
							Type:         schema.TypeInt,
							Optional:     true,
							Default:      86400,
							ValidateFunc: validation.IntBetween(0, 604800),
						},
					},
				},
			},

			"health_check": {
				Type:     schema.TypeList,
				Optional: true,
				Computed: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"interval": {
							Type:     schema.TypeInt,
							Optional: true,
							Default:  30,
						},

						"path": {
							Type:         schema.TypeString,
							Optional:     true,
							Computed:     true,
							ValidateFunc: validateAwsLbTargetGroupHealthCheckPath,
						},

						"port": {
							Type:         schema.TypeString,
							Optional:     true,
							Default:      "traffic-port",
							ValidateFunc: validateAwsLbTargetGroupHealthCheckPort,
						},

						"protocol": {
							Type:     schema.TypeString,
							Optional: true,
							Default:  "HTTP",
							StateFunc: func(v interface{}) string {
								return strings.ToUpper(v.(string))
							},
							ValidateFunc: validation.StringInSlice([]string{
								elbv2.ProtocolEnumHttp,
								elbv2.ProtocolEnumHttps,
								elbv2.ProtocolEnumTcp,
							}, true),
						},

						"timeout": {
							Type:         schema.TypeInt,
							Optional:     true,
							Computed:     true,
							ValidateFunc: validation.IntBetween(2, 60),
						},

						"healthy_threshold": {
							Type:         schema.TypeInt,
							Optional:     true,
							Default:      3,
							ValidateFunc: validation.IntBetween(2, 10),
						},

						"matcher": {
							Type:     schema.TypeString,
							Computed: true,
							Optional: true,
						},

						"unhealthy_threshold": {
							Type:         schema.TypeInt,
							Optional:     true,
							Default:      3,
							ValidateFunc: validation.IntBetween(2, 10),
						},
					},
				},
			},

			"tags": tagsSchema(),
		},
	}
}

func resourceAwsLbTargetGroupCreate(d *schema.ResourceData, meta interface{}) error {
	elbconn := meta.(*AWSClient).elbv2conn

	var groupName string
	if v, ok := d.GetOk("name"); ok {
		groupName = v.(string)
	} else if v, ok := d.GetOk("name_prefix"); ok {
		groupName = resource.PrefixedUniqueId(v.(string))
	} else {
		groupName = resource.PrefixedUniqueId("tf-")
	}

	params := &elbv2.CreateTargetGroupInput{
		Name:       aws.String(groupName),
		Port:       aws.Int64(int64(d.Get("port").(int))),
		Protocol:   aws.String(d.Get("protocol").(string)),
		VpcId:      aws.String(d.Get("vpc_id").(string)),
		TargetType: aws.String(d.Get("target_type").(string)),
	}

	if healthChecks := d.Get("health_check").([]interface{}); len(healthChecks) == 1 {
		healthCheck := healthChecks[0].(map[string]interface{})

		params.HealthCheckIntervalSeconds = aws.Int64(int64(healthCheck["interval"].(int)))
		params.HealthCheckPort = aws.String(healthCheck["port"].(string))
		params.HealthCheckProtocol = aws.String(healthCheck["protocol"].(string))
		params.HealthyThresholdCount = aws.Int64(int64(healthCheck["healthy_threshold"].(int)))
		params.UnhealthyThresholdCount = aws.Int64(int64(healthCheck["unhealthy_threshold"].(int)))
		t := healthCheck["timeout"].(int)
		if t != 0 {
			params.HealthCheckTimeoutSeconds = aws.Int64(int64(t))
		}

		if *params.HealthCheckProtocol != "TCP" {
			p := healthCheck["path"].(string)
			if p != "" {
				params.HealthCheckPath = aws.String(p)
			}

			m := healthCheck["matcher"].(string)
			if m != "" {
				params.Matcher = &elbv2.Matcher{
					HttpCode: aws.String(m),
				}
			}
		}
	}

	resp, err := elbconn.CreateTargetGroup(params)
	if err != nil {
		return fmt.Errorf("Error creating LB Target Group: %s", err)
	}

	if len(resp.TargetGroups) == 0 {
		return errors.New("Error creating LB Target Group: no groups returned in response")
	}

	d.SetId(aws.StringValue(resp.TargetGroups[0].TargetGroupArn))

	return resourceAwsLbTargetGroupUpdate(d, meta)
}

func resourceAwsLbTargetGroupRead(d *schema.ResourceData, meta interface{}) error {
	elbconn := meta.(*AWSClient).elbv2conn

	resp, err := elbconn.DescribeTargetGroups(&elbv2.DescribeTargetGroupsInput{
		TargetGroupArns: []*string{aws.String(d.Id())},
	})
	if err != nil {
		if isAWSErr(err, elbv2.ErrCodeTargetGroupNotFoundException, "") {
			log.Printf("[DEBUG] DescribeTargetGroups - removing %s from state", d.Id())
			d.SetId("")
			return nil
		}
		return fmt.Errorf("Error retrieving Target Group: %s", err)
	}

	if len(resp.TargetGroups) != 1 {
		return fmt.Errorf("Error retrieving Target Group %q", d.Id())
	}

	return flattenAwsLbTargetGroupResource(d, meta, resp.TargetGroups[0])
}

func resourceAwsLbTargetGroupUpdate(d *schema.ResourceData, meta interface{}) error {
	elbconn := meta.(*AWSClient).elbv2conn

	if err := setElbV2Tags(elbconn, d); err != nil {
		return fmt.Errorf("Error Modifying Tags on LB Target Group: %s", err)
	}

	if d.HasChange("health_check") {
		var params *elbv2.ModifyTargetGroupInput
		healthChecks := d.Get("health_check").([]interface{})
		if len(healthChecks) == 1 {
			params = &elbv2.ModifyTargetGroupInput{
				TargetGroupArn: aws.String(d.Id()),
			}
			healthCheck := healthChecks[0].(map[string]interface{})

			params = &elbv2.ModifyTargetGroupInput{
				TargetGroupArn:          aws.String(d.Id()),
				HealthCheckPort:         aws.String(healthCheck["port"].(string)),
				HealthCheckProtocol:     aws.String(healthCheck["protocol"].(string)),
				HealthyThresholdCount:   aws.Int64(int64(healthCheck["healthy_threshold"].(int))),
				UnhealthyThresholdCount: aws.Int64(int64(healthCheck["unhealthy_threshold"].(int))),
			}

			t := healthCheck["timeout"].(int)
			if t != 0 {
				params.HealthCheckTimeoutSeconds = aws.Int64(int64(t))
			}

			healthCheckProtocol := strings.ToLower(healthCheck["protocol"].(string))

			if healthCheckProtocol != "tcp" && !d.IsNewResource() {
				params.Matcher = &elbv2.Matcher{
					HttpCode: aws.String(healthCheck["matcher"].(string)),
				}
				params.HealthCheckPath = aws.String(healthCheck["path"].(string))
				params.HealthCheckIntervalSeconds = aws.Int64(int64(healthCheck["interval"].(int)))
			}
		}

		if params != nil {
			_, err := elbconn.ModifyTargetGroup(params)
			if err != nil {
				return fmt.Errorf("Error modifying Target Group: %s", err)
			}
		}
	}

	var attrs []*elbv2.TargetGroupAttribute

	if d.HasChange("deregistration_delay") {
		attrs = append(attrs, &elbv2.TargetGroupAttribute{
			Key:   aws.String("deregistration_delay.timeout_seconds"),
			Value: aws.String(fmt.Sprintf("%d", d.Get("deregistration_delay").(int))),
		})
	}

	if d.HasChange("slow_start") {
		attrs = append(attrs, &elbv2.TargetGroupAttribute{
			Key:   aws.String("slow_start.duration_seconds"),
			Value: aws.String(fmt.Sprintf("%d", d.Get("slow_start").(int))),
		})
	}

	if d.HasChange("proxy_protocol_v2") {
		attrs = append(attrs, &elbv2.TargetGroupAttribute{
			Key:   aws.String("proxy_protocol_v2.enabled"),
			Value: aws.String(strconv.FormatBool(d.Get("proxy_protocol_v2").(bool))),
		})
	}

	// In CustomizeDiff we allow LB stickiness to be declared for TCP target
	// groups, so long as it's not enabled. This allows for better support for
	// modules, but also means we need to completely skip sending the data to the
	// API if it's defined on a TCP target group.
	if d.HasChange("stickiness") && d.Get("protocol") != "TCP" {
		stickinessBlocks := d.Get("stickiness").([]interface{})
		if len(stickinessBlocks) == 1 {
			stickiness := stickinessBlocks[0].(map[string]interface{})

			attrs = append(attrs,
				&elbv2.TargetGroupAttribute{
					Key:   aws.String("stickiness.enabled"),
					Value: aws.String(strconv.FormatBool(stickiness["enabled"].(bool))),
				},
				&elbv2.TargetGroupAttribute{
					Key:   aws.String("stickiness.type"),
					Value: aws.String(stickiness["type"].(string)),
				},
				&elbv2.TargetGroupAttribute{
					Key:   aws.String("stickiness.lb_cookie.duration_seconds"),
					Value: aws.String(fmt.Sprintf("%d", stickiness["cookie_duration"].(int))),
				})
		} else if len(stickinessBlocks) == 0 {
			attrs = append(attrs, &elbv2.TargetGroupAttribute{
				Key:   aws.String("stickiness.enabled"),
				Value: aws.String("false"),
			})
		}
	}

	if len(attrs) > 0 {
		params := &elbv2.ModifyTargetGroupAttributesInput{
			TargetGroupArn: aws.String(d.Id()),
			Attributes:     attrs,
		}

		_, err := elbconn.ModifyTargetGroupAttributes(params)
		if err != nil {
			return fmt.Errorf("Error modifying Target Group Attributes: %s", err)
		}
	}

	return resourceAwsLbTargetGroupRead(d, meta)
}

func resourceAwsLbTargetGroupDelete(d *schema.ResourceData, meta interface{}) error {
	elbconn := meta.(*AWSClient).elbv2conn

	_, err := elbconn.DeleteTargetGroup(&elbv2.DeleteTargetGroupInput{
		TargetGroupArn: aws.String(d.Id()),
	})
	if err != nil {
		return fmt.Errorf("Error deleting Target Group: %s", err)
	}

	return nil
}

func validateAwsLbTargetGroupHealthCheckPath(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)
	if len(value) > 1024 {
		errors = append(errors, fmt.Errorf(
			"%q cannot be longer than 1024 characters: %q", k, value))
	}
	if len(value) > 0 && !strings.HasPrefix(value, "/") {
		errors = append(errors, fmt.Errorf(
			"%q must begin with a '/' character: %q", k, value))
	}
	return
}

func validateSlowStart(v interface{}, k string) (ws []string, errors []error) {
	value := v.(int)

	// Check if the value is between 30-900 or 0 (seconds).
	if value != 0 && !(value >= 30 && value <= 900) {
		errors = append(errors, fmt.Errorf(
			"%q contains an invalid Slow Start Duration \"%d\". "+
				"Valid intervals are 30-900 or 0 to disable.",
			k, value))
	}
	return
}

func validateAwsLbTargetGroupHealthCheckPort(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)

	if value == "traffic-port" {
		return
	}

	port, err := strconv.Atoi(value)
	if err != nil {
		errors = append(errors, fmt.Errorf("%q must be a valid port number (1-65536) or %q", k, "traffic-port"))
	}

	if port < 1 || port > 65536 {
		errors = append(errors, fmt.Errorf("%q must be a valid port number (1-65536) or %q", k, "traffic-port"))
	}

	return
}

func lbTargetGroupSuffixFromARN(arn *string) string {
	if arn == nil {
		return ""
	}

	if arnComponents := regexp.MustCompile(`arn:.*:targetgroup/(.*)`).FindAllStringSubmatch(*arn, -1); len(arnComponents) == 1 {
		if len(arnComponents[0]) == 2 {
			return fmt.Sprintf("targetgroup/%s", arnComponents[0][1])
		}
	}

	return ""
}

// flattenAwsLbTargetGroupResource takes a *elbv2.TargetGroup and populates all respective resource fields.
func flattenAwsLbTargetGroupResource(d *schema.ResourceData, meta interface{}, targetGroup *elbv2.TargetGroup) error {
	elbconn := meta.(*AWSClient).elbv2conn

	d.Set("arn", targetGroup.TargetGroupArn)
	d.Set("arn_suffix", lbTargetGroupSuffixFromARN(targetGroup.TargetGroupArn))
	d.Set("name", targetGroup.TargetGroupName)
	d.Set("port", targetGroup.Port)
	d.Set("protocol", targetGroup.Protocol)
	d.Set("vpc_id", targetGroup.VpcId)
	d.Set("target_type", targetGroup.TargetType)

	healthCheck := make(map[string]interface{})
	healthCheck["interval"] = int(aws.Int64Value(targetGroup.HealthCheckIntervalSeconds))
	healthCheck["port"] = aws.StringValue(targetGroup.HealthCheckPort)
	healthCheck["protocol"] = aws.StringValue(targetGroup.HealthCheckProtocol)
	healthCheck["timeout"] = int(aws.Int64Value(targetGroup.HealthCheckTimeoutSeconds))
	healthCheck["healthy_threshold"] = int(aws.Int64Value(targetGroup.HealthyThresholdCount))
	healthCheck["unhealthy_threshold"] = int(aws.Int64Value(targetGroup.UnhealthyThresholdCount))

	if targetGroup.HealthCheckPath != nil {
		healthCheck["path"] = aws.StringValue(targetGroup.HealthCheckPath)
	}
	if targetGroup.Matcher != nil && targetGroup.Matcher.HttpCode != nil {
		healthCheck["matcher"] = aws.StringValue(targetGroup.Matcher.HttpCode)
	}

	if err := d.Set("health_check", []interface{}{healthCheck}); err != nil {
		return fmt.Errorf("error setting health_check: %s", err)
	}

	attrResp, err := elbconn.DescribeTargetGroupAttributes(&elbv2.DescribeTargetGroupAttributesInput{
		TargetGroupArn: aws.String(d.Id()),
	})
	if err != nil {
		return fmt.Errorf("Error retrieving Target Group Attributes: %s", err)
	}

	for _, attr := range attrResp.Attributes {
		switch aws.StringValue(attr.Key) {
		case "proxy_protocol_v2.enabled":
			enabled, err := strconv.ParseBool(aws.StringValue(attr.Value))
			if err != nil {
				return fmt.Errorf("Error converting proxy_protocol_v2.enabled to bool: %s", aws.StringValue(attr.Value))
			}
			d.Set("proxy_protocol_v2", enabled)
		case "slow_start.duration_seconds":
			slowStart, err := strconv.Atoi(aws.StringValue(attr.Value))
			if err != nil {
				return fmt.Errorf("Error converting slow_start.duration_seconds to int: %s", aws.StringValue(attr.Value))
			}
			d.Set("slow_start", slowStart)
		}
	}

	// We only read in the stickiness attributes if the target group is not
	// TCP-based. This ensures we don't end up causing a spurious diff if someone
	// has defined the stickiness block on a TCP target group (albeit with
	// false), for which this update would clobber the state coming from config
	// for.
	//
	// This is a workaround to support module design where the module needs to
	// support HTTP and TCP target groups.
	switch {
	case aws.StringValue(targetGroup.Protocol) != "TCP":
		if err = flattenAwsLbTargetGroupStickiness(d, attrResp.Attributes); err != nil {
			return err
		}
	case aws.StringValue(targetGroup.Protocol) == "TCP" && len(d.Get("stickiness").([]interface{})) < 1:
		if err = d.Set("stickiness", []interface{}{}); err != nil {
			return fmt.Errorf("error setting stickiness: %s", err)
		}
	}

	tagsResp, err := elbconn.DescribeTags(&elbv2.DescribeTagsInput{
		ResourceArns: []*string{aws.String(d.Id())},
	})
	if err != nil {
		return fmt.Errorf("Error retrieving Target Group Tags: %s", err)
	}
	for _, t := range tagsResp.TagDescriptions {
		if aws.StringValue(t.ResourceArn) == d.Id() {
			if err := d.Set("tags", tagsToMapELBv2(t.Tags)); err != nil {
				return fmt.Errorf("error setting tags: %s", err)
			}
		}
	}

	return nil
}

func flattenAwsLbTargetGroupStickiness(d *schema.ResourceData, attributes []*elbv2.TargetGroupAttribute) error {
	stickinessMap := map[string]interface{}{}
	for _, attr := range attributes {
		switch aws.StringValue(attr.Key) {
		case "stickiness.enabled":
			enabled, err := strconv.ParseBool(aws.StringValue(attr.Value))
			if err != nil {
				return fmt.Errorf("Error converting stickiness.enabled to bool: %s", aws.StringValue(attr.Value))
			}
			stickinessMap["enabled"] = enabled
		case "stickiness.type":
			stickinessMap["type"] = aws.StringValue(attr.Value)
		case "stickiness.lb_cookie.duration_seconds":
			duration, err := strconv.Atoi(aws.StringValue(attr.Value))
			if err != nil {
				return fmt.Errorf("Error converting stickiness.lb_cookie.duration_seconds to int: %s", aws.StringValue(attr.Value))
			}
			stickinessMap["cookie_duration"] = duration
		case "deregistration_delay.timeout_seconds":
			timeout, err := strconv.Atoi(aws.StringValue(attr.Value))
			if err != nil {
				return fmt.Errorf("Error converting deregistration_delay.timeout_seconds to int: %s", aws.StringValue(attr.Value))
			}
			d.Set("deregistration_delay", timeout)
		}
	}

	setStickyMap := []interface{}{}
	if len(stickinessMap) > 0 {
		setStickyMap = []interface{}{stickinessMap}
	}
	if err := d.Set("stickiness", setStickyMap); err != nil {
		return err
	}
	return nil
}

func resourceAwsLbTargetGroupCustomizeDiff(diff *schema.ResourceDiff, v interface{}) error {
	protocol := diff.Get("protocol").(string)
	if protocol == "TCP" {
		// TCP load balancers do not support stickiness
		if stickinessBlocks := diff.Get("stickiness").([]interface{}); len(stickinessBlocks) == 1 {
			stickiness := stickinessBlocks[0].(map[string]interface{})
			if val := stickiness["enabled"].(bool); val {
				return fmt.Errorf("Network Load Balancers do not support Stickiness")
			}
		}
	}

	// Network Load Balancers have many special qwirks to them.
	// See http://docs.aws.amazon.com/elasticloadbalancing/latest/APIReference/API_CreateTargetGroup.html
	if healthChecks := diff.Get("health_check").([]interface{}); len(healthChecks) == 1 {
		healthCheck := healthChecks[0].(map[string]interface{})
		protocol := healthCheck["protocol"].(string)

		if protocol == "TCP" {
			// Cannot set custom matcher on TCP health checks
			if m := healthCheck["matcher"].(string); m != "" {
				return fmt.Errorf("%s: custom matcher is not supported for target_groups with TCP protocol", diff.Id())
			}
			// Cannot set custom path on TCP health checks
			if m := healthCheck["path"].(string); m != "" {
				return fmt.Errorf("%s: custom path is not supported for target_groups with TCP protocol", diff.Id())
			}
			// Cannot set custom timeout on TCP health checks
			if t := healthCheck["timeout"].(int); t != 0 && diff.Id() == "" {
				// timeout has a default value, so only check this if this is a network
				// LB and is a first run
				return fmt.Errorf("%s: custom timeout is not supported for target_groups with TCP protocol", diff.Id())
			}
			if healthCheck["healthy_threshold"].(int) != healthCheck["unhealthy_threshold"].(int) {
				return fmt.Errorf("%s: healthy_threshold %d and unhealthy_threshold %d must be the same for target_groups with TCP protocol", diff.Id(), healthCheck["healthy_threshold"].(int), healthCheck["unhealthy_threshold"].(int))
			}
		}
	}

	if strings.Contains(protocol, "HTTP") {
		if healthChecks := diff.Get("health_check").([]interface{}); len(healthChecks) == 1 {
			healthCheck := healthChecks[0].(map[string]interface{})
			// HTTP(S) Target Groups cannot use TCP health checks
			if p := healthCheck["protocol"].(string); strings.ToLower(p) == "tcp" {
				return fmt.Errorf("HTTP Target Groups cannot use TCP health checks")
			}
		}
	}

	if diff.Id() == "" {
		return nil
	}

	if protocol == "TCP" {
		if diff.HasChange("health_check.0.interval") {
			old, new := diff.GetChange("health_check.0.interval")
			return fmt.Errorf("Health check interval cannot be updated from %d to %d for TCP based Target Group %s,"+
				" use 'terraform taint' to recreate the resource if you wish",
				old, new, diff.Id())
		}
		if diff.HasChange("health_check.0.timeout") {
			old, new := diff.GetChange("health_check.0.timeout")
			return fmt.Errorf("Health check timeout cannot be updated from %d to %d for TCP based Target Group %s,"+
				" use 'terraform taint' to recreate the resource if you wish",
				old, new, diff.Id())
		}
	}
	return nil
}
