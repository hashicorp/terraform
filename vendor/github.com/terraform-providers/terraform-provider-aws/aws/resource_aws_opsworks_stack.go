package aws

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/opsworks"
)

func resourceAwsOpsworksStack() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsOpsworksStackCreate,
		Read:   resourceAwsOpsworksStackRead,
		Update: resourceAwsOpsworksStackUpdate,
		Delete: resourceAwsOpsworksStackDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"agent_version": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"name": {
				Type:     schema.TypeString,
				Required: true,
			},

			"region": {
				Type:     schema.TypeString,
				ForceNew: true,
				Required: true,
			},

			"service_role_arn": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"default_instance_profile_arn": {
				Type:     schema.TypeString,
				Required: true,
			},

			"color": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"configuration_manager_name": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "Chef",
			},

			"configuration_manager_version": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "11.10",
			},

			"manage_berkshelf": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},

			"berkshelf_version": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "3.2.0",
			},

			"custom_cookbooks_source": {
				Type:     schema.TypeList,
				Optional: true,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"type": {
							Type:     schema.TypeString,
							Required: true,
						},

						"url": {
							Type:     schema.TypeString,
							Required: true,
						},

						"username": {
							Type:     schema.TypeString,
							Optional: true,
						},

						"password": {
							Type:      schema.TypeString,
							Optional:  true,
							Sensitive: true,
						},

						"revision": {
							Type:     schema.TypeString,
							Optional: true,
						},

						"ssh_key": {
							Type:     schema.TypeString,
							Optional: true,
						},
					},
				},
			},

			"custom_json": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"default_availability_zone": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"default_os": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "Ubuntu 12.04 LTS",
			},

			"default_root_device_type": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "instance-store",
			},

			"default_ssh_key_name": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"default_subnet_id": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"hostname_theme": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "Layer_Dependent",
			},

			"tags": tagsSchema(),

			"use_custom_cookbooks": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},

			"use_opsworks_security_groups": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},

			"vpc_id": {
				Type:     schema.TypeString,
				ForceNew: true,
				Computed: true,
				Optional: true,
			},

			"stack_endpoint": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceAwsOpsworksStackValidate(d *schema.ResourceData) error {
	cookbooksSourceCount := d.Get("custom_cookbooks_source.#").(int)
	if cookbooksSourceCount > 1 {
		return fmt.Errorf("Only one custom_cookbooks_source is permitted")
	}

	vpcId := d.Get("vpc_id").(string)
	if vpcId != "" {
		if d.Get("default_subnet_id").(string) == "" {
			return fmt.Errorf("default_subnet_id must be set if vpc_id is set")
		}
	} else {
		if d.Get("default_availability_zone").(string) == "" {
			return fmt.Errorf("either vpc_id or default_availability_zone must be set")
		}
	}

	return nil
}

func resourceAwsOpsworksStackCustomCookbooksSource(d *schema.ResourceData) *opsworks.Source {
	count := d.Get("custom_cookbooks_source.#").(int)
	if count == 0 {
		return nil
	}

	return &opsworks.Source{
		Type:     aws.String(d.Get("custom_cookbooks_source.0.type").(string)),
		Url:      aws.String(d.Get("custom_cookbooks_source.0.url").(string)),
		Username: aws.String(d.Get("custom_cookbooks_source.0.username").(string)),
		Password: aws.String(d.Get("custom_cookbooks_source.0.password").(string)),
		Revision: aws.String(d.Get("custom_cookbooks_source.0.revision").(string)),
		SshKey:   aws.String(d.Get("custom_cookbooks_source.0.ssh_key").(string)),
	}
}

func resourceAwsOpsworksSetStackCustomCookbooksSource(d *schema.ResourceData, v *opsworks.Source) {
	nv := make([]interface{}, 0, 1)
	if v != nil && v.Type != nil && *v.Type != "" {
		m := make(map[string]interface{})
		if v.Type != nil {
			m["type"] = *v.Type
		}
		if v.Url != nil {
			m["url"] = *v.Url
		}
		if v.Username != nil {
			m["username"] = *v.Username
		}
		if v.Revision != nil {
			m["revision"] = *v.Revision
		}
		// v.Password will, on read, contain the placeholder string
		// "*****FILTERED*****", so we ignore it on read and let persist
		// the value already in the state.
		nv = append(nv, m)
	}

	err := d.Set("custom_cookbooks_source", nv)
	if err != nil {
		// should never happen
		panic(err)
	}
}

func resourceAwsOpsworksStackRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*AWSClient).opsworksconn
	var conErr error
	if v := d.Get("stack_endpoint").(string); v != "" {
		client, conErr = opsworksConnForRegion(v, meta)
		if conErr != nil {
			return conErr
		}
	}

	req := &opsworks.DescribeStacksInput{
		StackIds: []*string{
			aws.String(d.Id()),
		},
	}

	log.Printf("[DEBUG] Reading OpsWorks stack: %s", d.Id())

	// notFound represents the number of times we've called DescribeStacks looking
	// for this Stack. If it's not found in the the default region we're in, we
	// check us-east-1 in the event this stack was created with Terraform before
	// version 0.9
	// See https://github.com/hashicorp/terraform/issues/12842
	var notFound int
	var resp *opsworks.DescribeStacksOutput
	var dErr error

	for {
		resp, dErr = client.DescribeStacks(req)
		if dErr != nil {
			if awserr, ok := dErr.(awserr.Error); ok {
				if awserr.Code() == "ResourceNotFoundException" {
					if notFound < 1 {
						// If we haven't already, try us-east-1, legacy connection
						notFound++
						var connErr error
						client, connErr = opsworksConnForRegion("us-east-1", meta)
						if connErr != nil {
							return connErr
						}
						// start again from the top of the FOR loop, but with a client
						// configured to talk to us-east-1
						continue
					}

					// We've tried both the original and us-east-1 endpoint, and the stack
					// is still not found
					log.Printf("[DEBUG] OpsWorks stack (%s) not found", d.Id())
					d.SetId("")
					return nil
				}
				// not ResoureNotFoundException, fall through to returning error
			}
			return dErr
		}
		// If the stack was found, set the stack_endpoint
		if client.Config.Region != nil && *client.Config.Region != "" {
			log.Printf("[DEBUG] Setting stack_endpoint for (%s) to (%s)", d.Id(), *client.Config.Region)
			if err := d.Set("stack_endpoint", *client.Config.Region); err != nil {
				log.Printf("[WARN] Error setting stack_endpoint: %s", err)
			}
		}
		log.Printf("[DEBUG] Breaking stack endpoint search, found stack for (%s)", d.Id())
		// Break the FOR loop
		break
	}

	stack := resp.Stacks[0]
	d.Set("arn", stack.Arn)
	d.Set("agent_version", stack.AgentVersion)
	d.Set("name", stack.Name)
	d.Set("region", stack.Region)
	d.Set("default_instance_profile_arn", stack.DefaultInstanceProfileArn)
	d.Set("service_role_arn", stack.ServiceRoleArn)
	d.Set("default_availability_zone", stack.DefaultAvailabilityZone)
	d.Set("default_os", stack.DefaultOs)
	d.Set("default_root_device_type", stack.DefaultRootDeviceType)
	d.Set("default_ssh_key_name", stack.DefaultSshKeyName)
	d.Set("default_subnet_id", stack.DefaultSubnetId)
	d.Set("hostname_theme", stack.HostnameTheme)
	d.Set("use_custom_cookbooks", stack.UseCustomCookbooks)
	if stack.CustomJson != nil {
		d.Set("custom_json", stack.CustomJson)
	}
	d.Set("use_opsworks_security_groups", stack.UseOpsworksSecurityGroups)
	d.Set("vpc_id", stack.VpcId)
	if color, ok := stack.Attributes["Color"]; ok {
		d.Set("color", color)
	}
	if stack.ConfigurationManager != nil {
		d.Set("configuration_manager_name", stack.ConfigurationManager.Name)
		d.Set("configuration_manager_version", stack.ConfigurationManager.Version)
	}
	if stack.ChefConfiguration != nil {
		d.Set("berkshelf_version", stack.ChefConfiguration.BerkshelfVersion)
		d.Set("manage_berkshelf", stack.ChefConfiguration.ManageBerkshelf)
	}
	resourceAwsOpsworksSetStackCustomCookbooksSource(d, stack.CustomCookbooksSource)

	return nil
}

// opsworksConn will return a connection for the stack_endpoint in the
// configuration. Stacks can only be accessed or managed within the endpoint
// in which they are created, so we allow users to specify an original endpoint
// for Stacks created before multiple endpoints were offered (Terraform v0.9.0).
// See:
//  - https://github.com/hashicorp/terraform/pull/12688
//  - https://github.com/hashicorp/terraform/issues/12842
func opsworksConnForRegion(region string, meta interface{}) (*opsworks.OpsWorks, error) {
	originalConn := meta.(*AWSClient).opsworksconn

	// Regions are the same, no need to reconfigure
	if originalConn.Config.Region != nil && *originalConn.Config.Region == region {
		return originalConn, nil
	}

	// Set up base session
	sess, err := session.NewSession(&originalConn.Config)
	if err != nil {
		return nil, errwrap.Wrapf("Error creating AWS session: {{err}}", err)
	}

	sess.Handlers.Build.PushBackNamed(addTerraformVersionToUserAgent)

	if extraDebug := os.Getenv("TERRAFORM_AWS_AUTHFAILURE_DEBUG"); extraDebug != "" {
		sess.Handlers.UnmarshalError.PushFrontNamed(debugAuthFailure)
	}

	newSession := sess.Copy(&aws.Config{Region: aws.String(region)})
	newOpsworksconn := opsworks.New(newSession)

	log.Printf("[DEBUG] Returning new OpsWorks client")
	return newOpsworksconn, nil
}

func resourceAwsOpsworksStackCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*AWSClient).opsworksconn

	err := resourceAwsOpsworksStackValidate(d)
	if err != nil {
		return err
	}

	req := &opsworks.CreateStackInput{
		DefaultInstanceProfileArn: aws.String(d.Get("default_instance_profile_arn").(string)),
		Name:                      aws.String(d.Get("name").(string)),
		Region:                    aws.String(d.Get("region").(string)),
		ServiceRoleArn:            aws.String(d.Get("service_role_arn").(string)),
		DefaultOs:                 aws.String(d.Get("default_os").(string)),
		UseOpsworksSecurityGroups: aws.Bool(d.Get("use_opsworks_security_groups").(bool)),
	}
	req.ConfigurationManager = &opsworks.StackConfigurationManager{
		Name:    aws.String(d.Get("configuration_manager_name").(string)),
		Version: aws.String(d.Get("configuration_manager_version").(string)),
	}
	inVpc := false
	if vpcId, ok := d.GetOk("vpc_id"); ok {
		req.VpcId = aws.String(vpcId.(string))
		inVpc = true
	}
	if defaultSubnetId, ok := d.GetOk("default_subnet_id"); ok {
		req.DefaultSubnetId = aws.String(defaultSubnetId.(string))
	}
	if defaultAvailabilityZone, ok := d.GetOk("default_availability_zone"); ok {
		req.DefaultAvailabilityZone = aws.String(defaultAvailabilityZone.(string))
	}
	if defaultRootDeviceType, ok := d.GetOk("default_root_device_type"); ok {
		req.DefaultRootDeviceType = aws.String(defaultRootDeviceType.(string))
	}

	log.Printf("[DEBUG] Creating OpsWorks stack: %s", req)

	var resp *opsworks.CreateStackOutput
	err = resource.Retry(20*time.Minute, func() *resource.RetryError {
		var cerr error
		resp, cerr = client.CreateStack(req)
		if cerr != nil {
			if opserr, ok := cerr.(awserr.Error); ok {
				// If Terraform is also managing the service IAM role,
				// it may have just been created and not yet be
				// propagated.
				// AWS doesn't provide a machine-readable code for this
				// specific error, so we're forced to do fragile message
				// matching.
				// The full error we're looking for looks something like
				// the following:
				// Service Role Arn: [...] is not yet propagated, please try again in a couple of minutes
				propErr := "not yet propagated"
				trustErr := "not the necessary trust relationship"
				validateErr := "validate IAM role permission"
				if opserr.Code() == "ValidationException" && (strings.Contains(opserr.Message(), trustErr) || strings.Contains(opserr.Message(), propErr) || strings.Contains(opserr.Message(), validateErr)) {
					log.Printf("[INFO] Waiting for service IAM role to propagate")
					return resource.RetryableError(cerr)
				}
			}
			return resource.NonRetryableError(cerr)
		}
		return nil
	})
	if err != nil {
		return err
	}

	stackId := *resp.StackId
	d.SetId(stackId)

	if inVpc && *req.UseOpsworksSecurityGroups {
		// For VPC-based stacks, OpsWorks asynchronously creates some default
		// security groups which must exist before layers can be created.
		// Unfortunately it doesn't tell us what the ids of these are, so
		// we can't actually check for them. Instead, we just wait a nominal
		// amount of time for their creation to complete.
		log.Print("[INFO] Waiting for OpsWorks built-in security groups to be created")
		time.Sleep(30 * time.Second)
	}

	return resourceAwsOpsworksStackUpdate(d, meta)
}

func resourceAwsOpsworksStackUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*AWSClient).opsworksconn
	var conErr error
	if v := d.Get("stack_endpoint").(string); v != "" {
		client, conErr = opsworksConnForRegion(v, meta)
		if conErr != nil {
			return conErr
		}
	}

	err := resourceAwsOpsworksStackValidate(d)
	if err != nil {
		return err
	}

	req := &opsworks.UpdateStackInput{
		CustomJson:                aws.String(d.Get("custom_json").(string)),
		DefaultInstanceProfileArn: aws.String(d.Get("default_instance_profile_arn").(string)),
		DefaultRootDeviceType:     aws.String(d.Get("default_root_device_type").(string)),
		DefaultSshKeyName:         aws.String(d.Get("default_ssh_key_name").(string)),
		Name:                      aws.String(d.Get("name").(string)),
		ServiceRoleArn:            aws.String(d.Get("service_role_arn").(string)),
		StackId:                   aws.String(d.Id()),
		UseCustomCookbooks:        aws.Bool(d.Get("use_custom_cookbooks").(bool)),
		UseOpsworksSecurityGroups: aws.Bool(d.Get("use_opsworks_security_groups").(bool)),
		Attributes:                make(map[string]*string),
		CustomCookbooksSource:     resourceAwsOpsworksStackCustomCookbooksSource(d),
	}
	if v, ok := d.GetOk("agent_version"); ok {
		req.AgentVersion = aws.String(v.(string))
	}
	if v, ok := d.GetOk("default_os"); ok {
		req.DefaultOs = aws.String(v.(string))
	}
	if v, ok := d.GetOk("default_subnet_id"); ok {
		req.DefaultSubnetId = aws.String(v.(string))
	}
	if v, ok := d.GetOk("default_availability_zone"); ok {
		req.DefaultAvailabilityZone = aws.String(v.(string))
	}
	if v, ok := d.GetOk("hostname_theme"); ok {
		req.HostnameTheme = aws.String(v.(string))
	}
	if v, ok := d.GetOk("color"); ok {
		req.Attributes["Color"] = aws.String(v.(string))
	}

	arn := arn.ARN{
		Partition: meta.(*AWSClient).partition,
		Region:    meta.(*AWSClient).region,
		Service:   "opsworks",
		AccountID: meta.(*AWSClient).accountid,
		Resource:  fmt.Sprintf("stack/%s/", d.Id()),
	}

	if tagErr := setTagsOpsworks(client, d, arn.String()); tagErr != nil {
		return tagErr
	}

	req.ChefConfiguration = &opsworks.ChefConfiguration{
		BerkshelfVersion: aws.String(d.Get("berkshelf_version").(string)),
		ManageBerkshelf:  aws.Bool(d.Get("manage_berkshelf").(bool)),
	}

	req.ConfigurationManager = &opsworks.StackConfigurationManager{
		Name:    aws.String(d.Get("configuration_manager_name").(string)),
		Version: aws.String(d.Get("configuration_manager_version").(string)),
	}

	log.Printf("[DEBUG] Updating OpsWorks stack: %s", req)

	_, err = client.UpdateStack(req)
	if err != nil {
		return err
	}

	return resourceAwsOpsworksStackRead(d, meta)
}

func resourceAwsOpsworksStackDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*AWSClient).opsworksconn
	var conErr error
	if v := d.Get("stack_endpoint").(string); v != "" {
		client, conErr = opsworksConnForRegion(v, meta)
		if conErr != nil {
			return conErr
		}
	}

	req := &opsworks.DeleteStackInput{
		StackId: aws.String(d.Id()),
	}

	log.Printf("[DEBUG] Deleting OpsWorks stack: %s", d.Id())

	_, err := client.DeleteStack(req)
	if err != nil {
		return err
	}

	// For a stack in a VPC, OpsWorks has created some default security groups
	// in the VPC, which it will now delete.
	// Unfortunately, the security groups are deleted asynchronously and there
	// is no robust way for us to determine when it is done. The VPC itself
	// isn't deletable until the security groups are cleaned up, so this could
	// make 'terraform destroy' fail if the VPC is also managed and we don't
	// wait for the security groups to be deleted.
	// There is no robust way to check for this, so we'll just wait a
	// nominal amount of time.
	_, inVpc := d.GetOk("vpc_id")
	_, useOpsworksDefaultSg := d.GetOk("use_opsworks_security_group")

	if inVpc && useOpsworksDefaultSg {
		log.Print("[INFO] Waiting for Opsworks built-in security groups to be deleted")
		time.Sleep(30 * time.Second)
	}

	return nil
}
