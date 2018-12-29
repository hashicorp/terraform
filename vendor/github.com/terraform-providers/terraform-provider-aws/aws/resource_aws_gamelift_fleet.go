package aws

import (
	"fmt"
	"log"
	"reflect"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/gamelift"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
)

func resourceAwsGameliftFleet() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsGameliftFleetCreate,
		Read:   resourceAwsGameliftFleetRead,
		Update: resourceAwsGameliftFleetUpdate,
		Delete: resourceAwsGameliftFleetDelete,

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(70 * time.Minute),
			Delete: schema.DefaultTimeout(5 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"build_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"ec2_instance_type": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"name": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.StringLenBetween(1, 1024),
			},
			"description": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.StringLenBetween(1, 1024),
			},
			"ec2_inbound_permission": {
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 50,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"from_port": {
							Type:         schema.TypeInt,
							Required:     true,
							ValidateFunc: validation.IntBetween(1, 60000),
						},
						"ip_range": {
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: validateCIDRNetworkAddress,
						},
						"protocol": {
							Type:     schema.TypeString,
							Required: true,
							ValidateFunc: validation.StringInSlice([]string{
								gamelift.IpProtocolTcp,
								gamelift.IpProtocolUdp,
							}, false),
						},
						"to_port": {
							Type:         schema.TypeInt,
							Required:     true,
							ValidateFunc: validation.IntBetween(1, 60000),
						},
					},
				},
			},
			"log_paths": {
				Type:     schema.TypeList,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"metric_groups": {
				Type:     schema.TypeList,
				Optional: true,
				Computed: true,
				Elem: &schema.Schema{
					Type:         schema.TypeString,
					ValidateFunc: validation.StringLenBetween(1, 255),
				},
			},
			"new_game_session_protection_policy": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  gamelift.ProtectionPolicyNoProtection,
				ValidateFunc: validation.StringInSlice([]string{
					gamelift.ProtectionPolicyNoProtection,
					gamelift.ProtectionPolicyFullProtection,
				}, false),
			},
			"operating_system": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"resource_creation_limit_policy": {
				Type:     schema.TypeList,
				MaxItems: 1,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"new_game_sessions_per_creator": {
							Type:         schema.TypeInt,
							Optional:     true,
							ValidateFunc: validation.IntAtLeast(0),
						},
						"policy_period_in_minutes": {
							Type:         schema.TypeInt,
							Optional:     true,
							ValidateFunc: validation.IntAtLeast(0),
						},
					},
				},
			},
			"runtime_configuration": {
				Type:     schema.TypeList,
				MaxItems: 1,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"game_session_activation_timeout_seconds": {
							Type:         schema.TypeInt,
							Optional:     true,
							ValidateFunc: validation.IntBetween(1, 600),
						},
						"max_concurrent_game_session_activations": {
							Type:         schema.TypeInt,
							Optional:     true,
							ValidateFunc: validation.IntBetween(1, 2147483647),
						},
						"server_process": {
							Type:     schema.TypeList,
							Optional: true,
							MaxItems: 50,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"concurrent_executions": {
										Type:         schema.TypeInt,
										Required:     true,
										ValidateFunc: validation.IntAtLeast(1),
									},
									"launch_path": {
										Type:         schema.TypeString,
										Required:     true,
										ValidateFunc: validation.StringLenBetween(1, 1024),
									},
									"parameters": {
										Type:         schema.TypeString,
										Optional:     true,
										ValidateFunc: validation.StringLenBetween(1, 1024),
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func resourceAwsGameliftFleetCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).gameliftconn

	input := gamelift.CreateFleetInput{
		BuildId:         aws.String(d.Get("build_id").(string)),
		EC2InstanceType: aws.String(d.Get("ec2_instance_type").(string)),
		Name:            aws.String(d.Get("name").(string)),
	}

	if v, ok := d.GetOk("description"); ok {
		input.Description = aws.String(v.(string))
	}
	if v, ok := d.GetOk("ec2_inbound_permission"); ok {
		input.EC2InboundPermissions = expandGameliftIpPermissions(v.([]interface{}))
	}
	if v, ok := d.GetOk("metric_groups"); ok {
		input.MetricGroups = expandStringList(v.([]interface{}))
	}
	if v, ok := d.GetOk("new_game_session_protection_policy"); ok {
		input.NewGameSessionProtectionPolicy = aws.String(v.(string))
	}
	if v, ok := d.GetOk("resource_creation_limit_policy"); ok {
		input.ResourceCreationLimitPolicy = expandGameliftResourceCreationLimitPolicy(v.([]interface{}))
	}
	if v, ok := d.GetOk("runtime_configuration"); ok {
		input.RuntimeConfiguration = expandGameliftRuntimeConfiguration(v.([]interface{}))
	}

	log.Printf("[INFO] Creating Gamelift Fleet: %s", input)
	out, err := conn.CreateFleet(&input)
	if err != nil {
		return err
	}

	d.SetId(*out.FleetAttributes.FleetId)

	stateConf := &resource.StateChangeConf{
		Pending: []string{
			gamelift.FleetStatusActivating,
			gamelift.FleetStatusBuilding,
			gamelift.FleetStatusDownloading,
			gamelift.FleetStatusNew,
			gamelift.FleetStatusValidating,
		},
		Target:  []string{gamelift.FleetStatusActive},
		Timeout: d.Timeout(schema.TimeoutCreate),
		Refresh: func() (interface{}, string, error) {
			out, err := conn.DescribeFleetAttributes(&gamelift.DescribeFleetAttributesInput{
				FleetIds: aws.StringSlice([]string{d.Id()}),
			})
			if err != nil {
				return 42, "", err
			}

			attributes := out.FleetAttributes
			if len(attributes) < 1 {
				return nil, "", nil
			}
			if len(attributes) != 1 {
				return 42, "", fmt.Errorf("Expected exactly 1 Gamelift fleet, found %d under %q",
					len(attributes), d.Id())
			}

			fleet := attributes[0]
			return fleet, *fleet.Status, nil
		},
	}
	_, err = stateConf.WaitForState()
	if err != nil {
		events, fErr := getGameliftFleetFailures(conn, d.Id())
		if fErr != nil {
			log.Printf("[WARN] Failed to poll fleet failures: %s", fErr)
		}
		if len(events) > 0 {
			return fmt.Errorf("%s Recent failures:\n%+v", err, events)
		}

		return err
	}

	return resourceAwsGameliftFleetRead(d, meta)
}

func resourceAwsGameliftFleetRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).gameliftconn

	log.Printf("[INFO] Describing Gamelift Fleet: %s", d.Id())
	out, err := conn.DescribeFleetAttributes(&gamelift.DescribeFleetAttributesInput{
		FleetIds: aws.StringSlice([]string{d.Id()}),
	})
	if err != nil {
		return err
	}
	attributes := out.FleetAttributes
	if len(attributes) < 1 {
		log.Printf("[WARN] Gamelift Fleet (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}
	if len(attributes) != 1 {
		return fmt.Errorf("Expected exactly 1 Gamelift fleet, found %d under %q",
			len(attributes), d.Id())
	}
	fleet := attributes[0]

	d.Set("build_id", fleet.BuildId)
	d.Set("description", fleet.Description)
	d.Set("arn", fleet.FleetArn)
	d.Set("log_paths", aws.StringValueSlice(fleet.LogPaths))
	d.Set("metric_groups", flattenStringList(fleet.MetricGroups))
	d.Set("name", fleet.Name)
	d.Set("new_game_session_protection_policy", fleet.NewGameSessionProtectionPolicy)
	d.Set("operating_system", fleet.OperatingSystem)
	d.Set("resource_creation_limit_policy", flattenGameliftResourceCreationLimitPolicy(fleet.ResourceCreationLimitPolicy))

	return nil
}

func resourceAwsGameliftFleetUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).gameliftconn

	log.Printf("[INFO] Updating Gamelift Fleet: %s", d.Id())

	if d.HasChange("description") || d.HasChange("metric_groups") || d.HasChange("name") ||
		d.HasChange("new_game_session_protection_policy") || d.HasChange("resource_creation_limit_policy") {
		_, err := conn.UpdateFleetAttributes(&gamelift.UpdateFleetAttributesInput{
			Description:                    aws.String(d.Get("description").(string)),
			FleetId:                        aws.String(d.Id()),
			MetricGroups:                   expandStringList(d.Get("metric_groups").([]interface{})),
			Name:                           aws.String(d.Get("name").(string)),
			NewGameSessionProtectionPolicy: aws.String(d.Get("new_game_session_protection_policy").(string)),
			ResourceCreationLimitPolicy:    expandGameliftResourceCreationLimitPolicy(d.Get("resource_creation_limit_policy").([]interface{})),
		})
		if err != nil {
			return err
		}
	}

	if d.HasChange("ec2_inbound_permission") {
		oldPerms, newPerms := d.GetChange("ec2_inbound_permission")
		authorizations, revocations := diffGameliftPortSettings(oldPerms.([]interface{}), newPerms.([]interface{}))

		_, err := conn.UpdateFleetPortSettings(&gamelift.UpdateFleetPortSettingsInput{
			FleetId:                         aws.String(d.Id()),
			InboundPermissionAuthorizations: authorizations,
			InboundPermissionRevocations:    revocations,
		})
		if err != nil {
			return err
		}
	}

	if d.HasChange("runtime_configuration") {
		_, err := conn.UpdateRuntimeConfiguration(&gamelift.UpdateRuntimeConfigurationInput{
			FleetId:              aws.String(d.Id()),
			RuntimeConfiguration: expandGameliftRuntimeConfiguration(d.Get("runtime_configuration").([]interface{})),
		})
		if err != nil {
			return err
		}
	}

	return resourceAwsGameliftFleetRead(d, meta)
}

func resourceAwsGameliftFleetDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).gameliftconn

	log.Printf("[INFO] Deleting Gamelift Fleet: %s", d.Id())
	// It can take ~ 1 hr as Gamelift will keep retrying on errors like
	// invalid launch path and remain in state when it can't be deleted :/
	err := resource.Retry(60*time.Minute, func() *resource.RetryError {
		_, err := conn.DeleteFleet(&gamelift.DeleteFleetInput{
			FleetId: aws.String(d.Id()),
		})
		if err != nil {
			msg := fmt.Sprintf("Cannot delete fleet %s that is in status of ", d.Id())
			if isAWSErr(err, gamelift.ErrCodeInvalidRequestException, msg) {
				return resource.RetryableError(err)
			}
			return resource.NonRetryableError(err)
		}
		return nil
	})
	if err != nil {
		return err
	}

	return waitForGameliftFleetToBeDeleted(conn, d.Id(), d.Timeout(schema.TimeoutDelete))
}

func waitForGameliftFleetToBeDeleted(conn *gamelift.GameLift, id string, timeout time.Duration) error {
	stateConf := resource.StateChangeConf{
		Pending: []string{
			gamelift.FleetStatusActive,
			gamelift.FleetStatusDeleting,
			gamelift.FleetStatusError,
			gamelift.FleetStatusTerminated,
		},
		Target:  []string{},
		Timeout: timeout,
		Refresh: func() (interface{}, string, error) {
			out, err := conn.DescribeFleetAttributes(&gamelift.DescribeFleetAttributesInput{
				FleetIds: aws.StringSlice([]string{id}),
			})
			if err != nil {
				return 42, "", err
			}

			attributes := out.FleetAttributes
			if len(attributes) < 1 {
				return nil, "", nil
			}
			if len(attributes) != 1 {
				return 42, "", fmt.Errorf("Expected exactly 1 Gamelift fleet, found %d under %q",
					len(attributes), id)
			}

			fleet := attributes[0]
			return fleet, *fleet.Status, nil
		},
	}
	_, err := stateConf.WaitForState()
	if err != nil {
		events, fErr := getGameliftFleetFailures(conn, id)
		if fErr != nil {
			log.Printf("[WARN] Failed to poll fleet failures: %s", fErr)
		}
		if len(events) > 0 {
			return fmt.Errorf("%s Recent failures:\n%+v", err, events)
		}
	}
	return err
}

func expandGameliftIpPermissions(cfgs []interface{}) []*gamelift.IpPermission {
	if len(cfgs) < 1 {
		return []*gamelift.IpPermission{}
	}

	perms := make([]*gamelift.IpPermission, len(cfgs), len(cfgs))
	for i, rawCfg := range cfgs {
		cfg := rawCfg.(map[string]interface{})
		perms[i] = expandGameliftIpPermission(cfg)
	}
	return perms
}

func expandGameliftIpPermission(cfg map[string]interface{}) *gamelift.IpPermission {
	return &gamelift.IpPermission{
		FromPort: aws.Int64(int64(cfg["from_port"].(int))),
		IpRange:  aws.String(cfg["ip_range"].(string)),
		Protocol: aws.String(cfg["protocol"].(string)),
		ToPort:   aws.Int64(int64(cfg["to_port"].(int))),
	}
}

func expandGameliftResourceCreationLimitPolicy(cfg []interface{}) *gamelift.ResourceCreationLimitPolicy {
	if len(cfg) < 1 {
		return nil
	}
	out := gamelift.ResourceCreationLimitPolicy{}
	m := cfg[0].(map[string]interface{})

	if v, ok := m["new_game_sessions_per_creator"]; ok {
		out.NewGameSessionsPerCreator = aws.Int64(int64(v.(int)))
	}
	if v, ok := m["policy_period_in_minutes"]; ok {
		out.PolicyPeriodInMinutes = aws.Int64(int64(v.(int)))
	}

	return &out
}

func flattenGameliftResourceCreationLimitPolicy(policy *gamelift.ResourceCreationLimitPolicy) []interface{} {
	if policy == nil {
		return []interface{}{}
	}

	m := make(map[string]interface{}, 0)
	m["new_game_sessions_per_creator"] = *policy.NewGameSessionsPerCreator
	m["policy_period_in_minutes"] = *policy.PolicyPeriodInMinutes

	return []interface{}{m}
}

func expandGameliftRuntimeConfiguration(cfg []interface{}) *gamelift.RuntimeConfiguration {
	if len(cfg) < 1 {
		return nil
	}
	out := gamelift.RuntimeConfiguration{}
	m := cfg[0].(map[string]interface{})

	if v, ok := m["game_session_activation_timeout_seconds"].(int); ok && v > 0 {
		out.GameSessionActivationTimeoutSeconds = aws.Int64(int64(v))
	}
	if v, ok := m["max_concurrent_game_session_activations"].(int); ok && v > 0 {
		out.MaxConcurrentGameSessionActivations = aws.Int64(int64(v))
	}
	if v, ok := m["server_process"]; ok {
		out.ServerProcesses = expandGameliftServerProcesses(v.([]interface{}))
	}

	return &out
}

func expandGameliftServerProcesses(cfgs []interface{}) []*gamelift.ServerProcess {
	if len(cfgs) < 1 {
		return []*gamelift.ServerProcess{}
	}

	processes := make([]*gamelift.ServerProcess, len(cfgs), len(cfgs))
	for i, rawCfg := range cfgs {
		cfg := rawCfg.(map[string]interface{})
		process := &gamelift.ServerProcess{
			ConcurrentExecutions: aws.Int64(int64(cfg["concurrent_executions"].(int))),
			LaunchPath:           aws.String(cfg["launch_path"].(string)),
		}
		if v, ok := cfg["parameters"].(string); ok && len(v) > 0 {
			process.Parameters = aws.String(v)
		}
		processes[i] = process
	}
	return processes
}

func getGameliftFleetFailures(conn *gamelift.GameLift, id string) ([]*gamelift.Event, error) {
	var events []*gamelift.Event
	err := _getGameliftFleetFailures(conn, id, nil, &events)
	return events, err
}

func _getGameliftFleetFailures(conn *gamelift.GameLift, id string, nextToken *string, events *[]*gamelift.Event) error {
	eOut, err := conn.DescribeFleetEvents(&gamelift.DescribeFleetEventsInput{
		FleetId:   aws.String(id),
		NextToken: nextToken,
	})
	if err != nil {
		return err
	}

	for _, e := range eOut.Events {
		if isGameliftEventFailure(e) {
			*events = append(*events, e)
		}
	}

	if eOut.NextToken != nil {
		err := _getGameliftFleetFailures(conn, id, nextToken, events)
		if err != nil {
			return err
		}
	}

	return nil
}

func isGameliftEventFailure(event *gamelift.Event) bool {
	failureCodes := []string{
		gamelift.EventCodeFleetActivationFailed,
		gamelift.EventCodeFleetActivationFailedNoInstances,
		gamelift.EventCodeFleetBinaryDownloadFailed,
		gamelift.EventCodeFleetInitializationFailed,
		gamelift.EventCodeFleetStateError,
		gamelift.EventCodeFleetValidationExecutableRuntimeFailure,
		gamelift.EventCodeFleetValidationLaunchPathNotFound,
		gamelift.EventCodeFleetValidationTimedOut,
		gamelift.EventCodeFleetVpcPeeringFailed,
		gamelift.EventCodeGameSessionActivationTimeout,
		gamelift.EventCodeServerProcessCrashed,
		gamelift.EventCodeServerProcessForceTerminated,
		gamelift.EventCodeServerProcessInvalidPath,
		gamelift.EventCodeServerProcessProcessExitTimeout,
		gamelift.EventCodeServerProcessProcessReadyTimeout,
		gamelift.EventCodeServerProcessSdkInitializationTimeout,
		gamelift.EventCodeServerProcessTerminatedUnhealthy,
	}
	for _, fc := range failureCodes {
		if *event.EventCode == fc {
			return true
		}
	}
	return false
}

func diffGameliftPortSettings(oldPerms, newPerms []interface{}) (a []*gamelift.IpPermission, r []*gamelift.IpPermission) {
OUTER:
	for i, op := range oldPerms {
		oldPerm := op.(map[string]interface{})
		for j, np := range newPerms {
			newPerm := np.(map[string]interface{})

			// Remove permissions which have not changed so we're not wasting
			// API calls for removal & subseq. addition of the same ones
			if reflect.DeepEqual(oldPerm, newPerm) {
				oldPerms = append(oldPerms[:i], oldPerms[i+1:]...)
				newPerms = append(newPerms[:j], newPerms[j+1:]...)
				continue OUTER
			}
		}

		// Add what's left for revocation
		r = append(r, expandGameliftIpPermission(oldPerm))
	}

	for _, np := range newPerms {
		newPerm := np.(map[string]interface{})
		// Add what's left for authorization
		a = append(a, expandGameliftIpPermission(newPerm))
	}
	return
}
