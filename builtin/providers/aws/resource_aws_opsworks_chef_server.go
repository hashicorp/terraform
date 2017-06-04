package aws

import (
	"errors"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/opsworkscm"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsOpsworksChefServer() *schema.Resource {
	// Maximum length for the "name" attribute
	maxNameLength := 40

	// The length of the suffix appended to "name_prefix"
	nameSuffixLength := 26

	return &schema.Resource{
		Create: resourceAwsOpsworksChefServerCreate,
		Read:   resourceAwsOpsworksChefServerRead,
		Update: resourceAwsOpsworksChefServerUpdate,
		Delete: resourceAwsOpsworksChefServerDelete,

		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(40 * time.Minute),
			Update: schema.DefaultTimeout(80 * time.Minute),
			Delete: schema.DefaultTimeout(40 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"cloudformation_stack_arn": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"endpoint": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"name": {
				Type:          schema.TypeString,
				Optional:      true,
				Computed:      true,
				ForceNew:      true,
				ConflictsWith: []string{"name_prefix"},
				ValidateFunc:  validateName(maxNameLength),
			},

			"name_prefix": {
				Type:         schema.TypeString,
				Optional:     true,
				ForceNew:     true,
				ValidateFunc: validateName(maxNameLength - nameSuffixLength),
			},

			"instance_type": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"instance_profile_arn": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validateArn,
			},

			"service_role_arn": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validateArn,
			},

			"key_name": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},

			"security_group_ids": {
				Type:     schema.TypeSet,
				Optional: true,
				Computed: true,
				ForceNew: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},

			"subnet_ids": {
				Type:     schema.TypeSet,
				Optional: true,
				Computed: true,
				ForceNew: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},

			"associate_public_ip_address": {
				Type:     schema.TypeBool,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},

			"backup_automatically": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},

			"backup_id": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"backup_retention_count": {
				Type:     schema.TypeInt,
				Optional: true,
				Computed: true,
			},

			"preferred_backup_window": {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				ValidateFunc: validateOnceADayWindowFormat,
			},

			"preferred_maintenance_window": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				StateFunc: func(v interface{}) string {
					if v != nil {
						value := v.(string)
						return strings.ToLower(value)
					}
					return ""
				},
				ValidateFunc: validateOnceAWeekWindowFormat,
			},

			"chef_delivery_admin_password": {
				Type:      schema.TypeString,
				Optional:  true,
				ForceNew:  true,
				Sensitive: true,
			},

			"chef_pivotal_key": {
				Type:      schema.TypeString,
				Optional:  true,
				Computed:  true,
				ForceNew:  true,
				Sensitive: true,
			},

			"chef_starter_kit": {
				Type:      schema.TypeString,
				Computed:  true,
				Sensitive: true,
			},

			"engine": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "Chef",
				ForceNew: true,
			},

			"engine_model": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "Single",
				ForceNew: true,
			},

			"engine_version": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "12",
				ForceNew: true,
				DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
					// Suppress diff if old is a prefix of the new (eg. "12" vs. "12.5.1")
					return len(old) > 0 && strings.HasPrefix(old, new)
				},
			},
		},
	}
}

func validateName(maxNameLength int) schema.SchemaValidateFunc {
	return func(i interface{}, k string) (ws []string, errors []error) {
		v, ok := i.(string)
		if !ok {
			errors = append(errors, fmt.Errorf("expected type of %s to be string", k))
			return
		}

		if len(v) > maxNameLength {
			message := "expected length of %s to be less than %d characters, got %s"
			errors = append(errors, fmt.Errorf(message, k, maxNameLength, v))
		}

		pattern := `[a-zA-Z][a-zA-Z0-9\-]*`
		if !regexp.MustCompile(pattern).MatchString(v) {
			message := "expected %s to satisfy regular expression pattern: %s"
			errors = append(errors, fmt.Errorf(message, k, pattern))
		}
		return
	}
}

func resourceAwsOpsworksChefServerCreate(d *schema.ResourceData, meta interface{}) error {
	var serverName string
	if v, ok := d.GetOk("name"); ok {
		serverName = v.(string)
	} else {
		if v, ok := d.GetOk("name_prefix"); ok {
			serverName = resource.PrefixedUniqueId(v.(string))
			if err := d.Set("name", serverName); err != nil {
				return fmt.Errorf("Error setting 'name': %s", err)
			}
		} else {
			return errors.New("One of 'name' or 'name_prefix' is required")
		}
	}

	request := &opsworkscm.CreateServerInput{
		DisableAutomatedBackup: aws.Bool(!(d.Get("backup_automatically").(bool))),
		Engine:                 aws.String(d.Get("engine").(string)),
		EngineModel:            aws.String(d.Get("engine_model").(string)),
		EngineVersion:          aws.String(d.Get("engine_version").(string)),
		InstanceProfileArn:     aws.String(d.Get("instance_profile_arn").(string)),
		InstanceType:           aws.String(d.Get("instance_type").(string)),
		ServerName:             aws.String(serverName),
		ServiceRoleArn:         aws.String(d.Get("service_role_arn").(string)),
	}

	if v, ok := d.GetOk("associate_public_ip_address"); ok {
		request.AssociatePublicIpAddress = aws.Bool(v.(bool))
	}

	if v, ok := d.GetOk("backup_id"); ok {
		request.BackupId = aws.String(v.(string))
	}

	if v, ok := d.GetOk("backup_retention_count"); ok {
		request.BackupRetentionCount = aws.Int64(int64(v.(int)))
	}

	if v, ok := d.GetOk("key_name"); ok {
		request.KeyPair = aws.String(v.(string))
	}

	var engineAttributes []*opsworkscm.EngineAttribute

	if v, ok := d.GetOk("chef_pivotal_key"); ok {
		engineAttributes = append(engineAttributes, &opsworkscm.EngineAttribute{
			Name:  aws.String("CHEF_PIVOTAL_KEY"),
			Value: aws.String(v.(string)),
		})
	}

	if v, ok := d.GetOk("chef_delivery_admin_password"); ok {
		engineAttributes = append(engineAttributes, &opsworkscm.EngineAttribute{
			Name: aws.String("CHEF_DELIVERY_ADMIN_PASSWORD"),
			// TODO: does this need base64 encoding?
			Value: aws.String(v.(string)),
		})
	}

	if len(engineAttributes) > 0 {
		request.EngineAttributes = engineAttributes
	}

	if v, ok := d.GetOk("preferred_backup_window"); ok {
		request.PreferredBackupWindow = aws.String(v.(string))
	}

	if v, ok := d.GetOk("preferred_maintenance_window"); ok {
		request.PreferredMaintenanceWindow = aws.String(v.(string))
	}

	if v, ok := d.GetOk("security_group_ids"); ok {
		request.SecurityGroupIds = expandStringList(v.(*schema.Set).List())
	}

	if v, ok := d.GetOk("subnet_ids"); ok {
		request.SubnetIds = expandStringList(v.(*schema.Set).List())
	}

	log.Printf("[DEBUG] Creating OpsWorks server for Chef Automate (%s), options:\n%s", d.Id(), request)

	client := opsworkscmClient(meta)
	var response *opsworkscm.CreateServerOutput
	response, err := client.CreateServer(request)
	if err != nil {
		return fmt.Errorf("Error creating OpsWorks server for Chef Automate: %s", err)
	}

	d.SetId(serverName)
	if err := d.Set("arn", response.Server.ServerArn); err != nil {
		return fmt.Errorf("Error setting '%s': %s", "arn", err)
	}
	if err := d.Set("cloudformation_stack_arn", response.Server.CloudFormationStackArn); err != nil {
		return fmt.Errorf("Error setting '%s': %s", "cloudformation_stack_arn", err)
	}
	if err := d.Set("endpoint", response.Server.Endpoint); err != nil {
		return fmt.Errorf("Error setting '%s': %s", "endpoint", err)
	}
	if err := d.Set("engine_version", response.Server.EngineVersion); err != nil {
		return fmt.Errorf("Error setting '%s': %s", "engine_version", err)
	}

	for _, attribute := range response.Server.EngineAttributes {
		key := strings.ToLower(aws.StringValue(attribute.Name))
		if key == "chef_pivotal_key" || key == "chef_starter_kit" {
			if err := d.Set(key, attribute.Value); err != nil {
				return fmt.Errorf("Error setting '%s': %s", key, err)
			}
		}
	}

	stateConf := &resource.StateChangeConf{
		Pending:      []string{"BACKING_UP", "CONNECTION_LOST", "CREATING",
			"DELETING", "MODIFYING", "FAILED", "HEALTHY", "RUNNING", "RESTORING",
			"SETUP", "UNDER_MAINTENANCE", "UNHEALTHY", "TERMINATED"},
		Target:       []string{"HEALTHY", "RUNNING"},
		Refresh:      resourceAwsOpsworksChefServerStateRefreshFunc(client, serverName, "FAILED"),
		Timeout:      40 * time.Minute,
		MinTimeout:   10 * time.Second,
		PollInterval: 1 * time.Minute,
		Delay:        30 * time.Second,
	}

	log.Printf("[DEBUG] Waiting for state to become available: %v", d.Id())
	_, sterr := stateConf.WaitForState()
	if sterr != nil {
		return fmt.Errorf("Error waiting for Chef Server (%s) to be created: %s", d.Id(), sterr)
	}

	return resourceAwsOpsworksChefServerRead(d, meta)
}

func resourceAwsOpsworksChefServerRead(d *schema.ResourceData, meta interface{}) error {
	v, err := resourceAwsOpsworksChefServerRetrieve(d.Id(), opsworkscmClient(meta))

	if err != nil {
		return err
	}
	if v == nil {
		d.SetId("")
		return nil
	}

	d.SetId(aws.StringValue(v.ServerName))
	if err := d.Set("arn", v.ServerArn); err != nil {
		return fmt.Errorf("Error setting '%s': %s", "arn", err)
	}
	if err := d.Set("cloudformation_stack_arn", v.CloudFormationStackArn); err != nil {
		return fmt.Errorf("Error setting '%s': %s", "cloudformation_stack_arn", err)
	}
	if err := d.Set("endpoint", v.Endpoint); err != nil {
		return fmt.Errorf("Error setting '%s': %s", "endpoint", err)
	}
	if err := d.Set("name", v.ServerName); err != nil {
		return fmt.Errorf("Error setting '%s': %s", "name", err)
	}
	if err := d.Set("instance_type", v.InstanceType); err != nil {
		return fmt.Errorf("Error setting '%s': %s", "instance_type", err)
	}
	if err := d.Set("instance_profile_arn", v.InstanceProfileArn); err != nil {
		return fmt.Errorf("Error setting '%s': %s", "instance_profile_arn", err)
	}
	if err := d.Set("service_role_arn", v.ServiceRoleArn); err != nil {
		return fmt.Errorf("Error setting '%s': %s", "service_role_arn", err)
	}
	if err := d.Set("key_name", v.KeyPair); err != nil {
		return fmt.Errorf("Error setting '%s': %s", "key_name", err)
	}
	if err := d.Set("associate_public_ip_address", v.AssociatePublicIpAddress); err != nil {
		return fmt.Errorf("Error setting '%s': %s", "associate_public_ip_address", err)
	}
	if err := d.Set("backup_automatically", !aws.BoolValue(v.DisableAutomatedBackup)); err != nil {
		return fmt.Errorf("Error setting '%s': %s", "backup_automatically", err)
	}
	if err := d.Set("backup_retention_count", v.BackupRetentionCount); err != nil {
		return fmt.Errorf("Error setting '%s': %s", "backup_retention_count", err)
	}
	if err := d.Set("preferred_backup_window", v.PreferredBackupWindow); err != nil {
		return fmt.Errorf("Error setting '%s': %s", "preferred_backup_window", err)
	}
	if err := d.Set("preferred_maintenance_window", v.PreferredMaintenanceWindow); err != nil {
		return fmt.Errorf("Error setting '%s': %s", "preferred_maintenance_window", err)
	}
	if err := d.Set("engine", v.Engine); err != nil {
		return fmt.Errorf("Error setting '%s': %s", "engine", err)
	}
	if err := d.Set("engine_model", v.EngineModel); err != nil {
		return fmt.Errorf("Error setting '%s': %s", "engine_model", err)
	}
	if err := d.Set("engine_version", v.EngineVersion); err != nil {
		return fmt.Errorf("Error setting '%s': %s", "engine_version", err)
	}

	for _, attribute := range v.EngineAttributes {
		key := strings.ToLower(aws.StringValue(attribute.Name))
		if key == "chef_pivotal_key" || key == "chef_starter_kit" {
			if err := d.Set(key, attribute.Value); err != nil {
				return fmt.Errorf("Error setting '%s': %s", key, err)
			}
		}
	}

	// Create an empty schema.Set to hold all security group ids
	securityGroupIds := &schema.Set{F: schema.HashString}
	for _, securityGroupId := range v.SecurityGroupIds {
		securityGroupIds.Add(*securityGroupId)
	}
	if err := d.Set("security_group_ids", securityGroupIds); err != nil {
		return fmt.Errorf("Error setting '%s': %s", "security_group_ids", err)
	}

	// Create an empty schema.Set to hold all subnet IDs
	subnetIds := &schema.Set{F: schema.HashString}
	for _, subnetId := range v.SubnetIds {
		subnetIds.Add(*subnetId)
	}
	if err := d.Set("subnet_ids", subnetIds); err != nil {
		return fmt.Errorf("Error setting '%s': %s", "subnet_ids", err)
	}

	return nil
}

// resourceAwsOpsworksChefServerRetrieve fetches resource information from the
// AWS API. It returns an error if there is a communication problem or
// unexpected error with AWS. When the resource is not found, it returns no
// error and a nil pointer.
func resourceAwsOpsworksChefServerRetrieve(serverName string,
	client *opsworkscm.OpsWorksCM) (*opsworkscm.Server, error) {
	request := &opsworkscm.DescribeServersInput{
		ServerName: aws.String(serverName),
	}

	log.Printf("[DEBUG] Retrieving OpsWorks Chef Server: %v", serverName)

	response, err := client.DescribeServers(request)
	if err != nil {
		e, ok := err.(awserr.Error)
		if ok && e.Code() == "ResourceNotFoundException" {
			log.Printf("[DEBUG] OpsWorks Chef Server %v not found: %s", serverName, err)
			return nil, nil
		}
		log.Printf("[DEBUG] OpsWorks Chef Server %v error: %s", serverName, err)
		return nil, fmt.Errorf("Error retrieving OpsWorks Servers for Chef Automate: %s", err)
	}

	if len(response.Servers) != 1 || *response.Servers[0].ServerName != serverName {
		log.Printf("[DEBUG] OpsWorks Chef Server %v not found in response: %s", serverName, err)
		return nil, nil
	}

	log.Printf("[DEBUG] OpsWorks Chef Server %v found", serverName)
	return response.Servers[0], nil
}

func resourceAwsOpsworksChefServerUpdate(d *schema.ResourceData, meta interface{}) error {
	client := opsworkscmClient(meta)
	request := &opsworkscm.UpdateServerInput{ServerName: aws.String(d.Id())}

	// Determine which fields, if any, require updating
	requestUpdate := false

	if d.HasChange("backup_automatically") {
		request.DisableAutomatedBackup = aws.Bool(!(d.Get("backup_automatically").(bool)))
		requestUpdate = true
	}

	if d.HasChange("backup_retention_count") {
		request.BackupRetentionCount = aws.Int64(int64(d.Get("backup_retention_count").(int)))
		requestUpdate = true
	}

	if d.HasChange("preferred_backup_window") {
		request.PreferredBackupWindow = aws.String(d.Get("preferred_backup_window").(string))
		requestUpdate = true
	}

	if d.HasChange("preferred_maintenance_window") {
		request.PreferredMaintenanceWindow = aws.String(d.Get("preferred_maintenance_window").(string))
		requestUpdate = true
	}

	if requestUpdate {
		log.Printf("[DEBUG] Updating OpsWorks Server for Chef Automate (%s), options:\n%s", d.Id(), request)
		log.Printf("[DEBUG] Updating OpsWorks Server for Chef Automate (%s), options:\n%s", d.Id(), request)
		_, err := client.UpdateServer(request)
		if err != nil {
			return fmt.Errorf("[WARN] Error updating OpsWorks Server for Chef Automate (%s), error: %s", d.Id(), err)
		}

		log.Printf("[DEBUG] Waiting for update: %s", d.Id())
		stateConf := &resource.StateChangeConf{
			Pending:    []string{"BACKING_UP", "CONNECTION_LOST", "CREATING",
				"DELETING", "MODIFYING", "RESTORING", "SETUP", "UNDER_MAINTENANCE",
				"UNHEALTHY", "TERMINATED"},
			Target:     []string{"HEALTHY", "RUNNING"},
			Refresh:    resourceAwsOpsworksChefServerStateRefreshFunc(client, d.Id(), "FAILED"),
			Timeout:    10 * time.Minute,
			MinTimeout: 10 * time.Second,
			Delay:      30 * time.Second,
		}

		_, sterr := stateConf.WaitForState()
		if sterr != nil {
			return fmt.Errorf("Error waiting for OpsWorks Server for Chef Automate (%s) to update: %s", d.Id(), sterr)
		}
	}

	return resourceAwsOpsworksChefServerRead(d, meta)
}

func resourceAwsOpsworksChefServerDelete(d *schema.ResourceData, meta interface{}) error {
	client := opsworkscmClient(meta)

	request := &opsworkscm.DeleteServerInput{ServerName: aws.String(d.Id())}
	_, err := client.DeleteServer(request)
	if err != nil {
		return err
	}

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"BACKING_UP", "CONNECTION_LOST", "CREATING",
			"DELETING", "MODIFYING", "HEALTHY", "RUNNING", "RESTORING",
			"SETUP", "UNDER_MAINTENANCE", "UNHEALTHY", "TERMINATED"},
		Target:     []string{},
		Refresh:    resourceAwsOpsworksChefServerStateRefreshFunc(client, d.Id(), "FAILED"),
		Timeout:    40 * time.Minute,
		MinTimeout: 30 * time.Second,
		Delay:      30 * time.Second,
	}

	log.Printf("[DEBUG] Waiting for deletion: %v", d.Id())
	_, sterr := stateConf.WaitForState()
	if sterr != nil {
		return fmt.Errorf("Error waiting for Chef Server (%s) to delete: %s", d.Id(), sterr)
	}

	d.SetId("")

	return nil
}

// Creates a resource.StateRefreshFunc that is used to watch a server
func resourceAwsOpsworksChefServerStateRefreshFunc(client *opsworkscm.OpsWorksCM,
	serverName string, failState string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		v, err := resourceAwsOpsworksChefServerRetrieve(serverName, client)

		// Handle unexpected errors
		if err != nil {
			log.Printf("Error on OpsworksChefServerStateRefresh: %s", err)
			return nil, "", err
		}

		// Handle the case where the server isn't found
		if v == nil {
			return nil, "", nil
		}

		state := aws.StringValue(v.Status)

		if state == failState {
			return v, state, fmt.Errorf("Failed to reach target state. Reason: %s",
				aws.StringValue(v.StatusReason))
		}

		return v, state, nil
	}
}

// Retrieves and configures the opsworkscm connection from the configuration
func opsworkscmClient(meta interface{}) *opsworkscm.OpsWorksCM {
	// logger := log.New(os.Stderr, "aws-sdk-go", log.Llongfile)
	client := meta.(*AWSClient).opsworkscmconn
	// logger := aws.LoggerFunc(func(args ...interface{}) {
	//     fmt.Fprintln(os.Stderr, args...)
	// })

	// TODO: manipulate client before returning
	client.Config.LogLevel = aws.LogLevel(aws.LogDebugWithHTTPBody)
	// client.Config.Logger = logger

	return client
}
