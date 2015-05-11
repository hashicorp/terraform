package aws

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"

	"github.com/awslabs/aws-sdk-go/aws"
	"github.com/awslabs/aws-sdk-go/aws/awserr"
	"github.com/awslabs/aws-sdk-go/service/opsworks"
)

func resourceAwsOpsworksStack() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsOpsworksStackCreate,
		Read:   resourceAwsOpsworksStackRead,
		Update: resourceAwsOpsworksStackUpdate,
		Delete: resourceAwsOpsworksStackDelete,

		Schema: map[string]*schema.Schema{
			"id": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"region": &schema.Schema{
				Type:     schema.TypeString,
				ForceNew: true,
				Required: true,
			},

			"service_role_arn": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"default_instance_profile_arn": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"color": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},

			"configuration_manager_name": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "Chef",
			},

			"configuration_manager_version": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "11.4",
			},

			"manage_berkshelf": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},

			"berkshelf_version": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "3.2.0",
			},

			"custom_cookbooks_source": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"type": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},

						"url": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},

						"username": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},

						"password": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},

						"revision": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},

						"ssh_key": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
					},
				},
			},

			"custom_json": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},

			"default_availability_zone": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"default_os": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "Ubuntu 12.04 LTS",
			},

			"default_root_device_type": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "instance-store",
			},

			"default_ssh_key_name": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},

			"default_subnet_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},

			"hostname_theme": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "Layer_Dependent",
			},

			"use_custom_cookbooks": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},

			"use_opsworks_security_groups": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},

			"vpc_id": &schema.Schema{
				Type:     schema.TypeString,
				ForceNew: true,
				Optional: true,
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
		URL:      aws.String(d.Get("custom_cookbooks_source.0.url").(string)),
		Username: aws.String(d.Get("custom_cookbooks_source.0.username").(string)),
		Password: aws.String(d.Get("custom_cookbooks_source.0.password").(string)),
		Revision: aws.String(d.Get("custom_cookbooks_source.0.revision").(string)),
		SSHKey:   aws.String(d.Get("custom_cookbooks_source.0.ssh_key").(string)),
	}
}

func resourceAwsOpsworksSetStackCustomCookbooksSource(d *schema.ResourceData, v *opsworks.Source) {
	nv := make([]interface{}, 0, 1)
	if v != nil {
		m := make(map[string]interface{})
		if v.Type != nil {
			m["type"] = *v.Type
		}
		if v.URL != nil {
			m["url"] = *v.URL
		}
		if v.Username != nil {
			m["username"] = *v.Username
		}
		if v.Password != nil {
			m["password"] = *v.Password
		}
		if v.Revision != nil {
			m["revision"] = *v.Revision
		}
		if v.SSHKey != nil {
			m["ssh_key"] = *v.SSHKey
		}
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

	req := &opsworks.DescribeStacksInput{
		StackIDs: []*string{
			aws.String(d.Id()),
		},
	}

	log.Printf("[DEBUG] Reading OpsWorks stack: %s", d.Id())

	resp, err := client.DescribeStacks(req)
	if err != nil {
		if awserr, ok := err.(awserr.Error); ok {
			if awserr.Code() == "ResourceNotFound" {
				d.SetId("")
				return nil
			}
		}
		return err
	}

	stack := resp.Stacks[0]
	d.Set("name", stack.Name)
	d.Set("region", stack.Region)
	d.Set("default_instance_profile_arn", stack.DefaultInstanceProfileARN)
	d.Set("service_role_arn", stack.ServiceRoleARN)
	d.Set("default_availability_zone", stack.DefaultAvailabilityZone)
	d.Set("default_os", stack.DefaultOs)
	d.Set("default_root_device_type", stack.DefaultRootDeviceType)
	d.Set("default_ssh_key_name", stack.DefaultSSHKeyName)
	d.Set("default_subnet_id", stack.DefaultSubnetID)
	d.Set("hostname_theme", stack.HostnameTheme)
	d.Set("use_custom_cookbooks", stack.UseCustomCookbooks)
	d.Set("use_opsworks_security_groups", stack.UseOpsWorksSecurityGroups)
	d.Set("vpc_id", stack.VPCID)
	if stack.Attributes != nil {
		if color, ok := (*stack.Attributes)["Color"]; ok {
			d.Set("color", color)
		}
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

func resourceAwsOpsworksStackCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*AWSClient).opsworksconn

	err := resourceAwsOpsworksStackValidate(d)
	if err != nil {
		return err
	}

	req := &opsworks.CreateStackInput{
		DefaultInstanceProfileARN: aws.String(d.Get("default_instance_profile_arn").(string)),
		Name:                    aws.String(d.Get("name").(string)),
		Region:                  aws.String(d.Get("region").(string)),
		ServiceRoleARN:          aws.String(d.Get("service_role_arn").(string)),
		VPCID:                   aws.String(d.Get("vpc_id").(string)),
		DefaultSubnetID:         aws.String(d.Get("default_subnet_id").(string)),
		DefaultAvailabilityZone: aws.String(d.Get("default_availability_zone").(string)),
	}
	inVpc := false
	if vpcId, ok := d.GetOk("vpc_id"); ok {
		req.VPCID = aws.String(vpcId.(string))
		inVpc = true
	}
	if defaultSubnetId, ok := d.GetOk("default_subnet_id"); ok {
		req.DefaultSubnetID = aws.String(defaultSubnetId.(string))
	}
	if defaultAvailabilityZone, ok := d.GetOk("default_availability_zone"); ok {
		req.DefaultAvailabilityZone = aws.String(defaultAvailabilityZone.(string))
	}

	log.Printf("[DEBUG] Creating OpsWorks stack: %s", *req.Name)

	var resp *opsworks.CreateStackOutput
	err = resource.Retry(20*time.Minute, func() error {
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
				if opserr.Code() == "ValidationException" && strings.Contains(opserr.Message(), "not yet propagated") {
					log.Printf("[INFO] Waiting for service IAM role to propagate")
					return cerr
				}
			}
			return resource.RetryError{Err: cerr}
		}
		return nil
	})
	if err != nil {
		return err
	}

	stackId := *resp.StackID
	d.SetId(stackId)
	d.Set("id", stackId)

	if inVpc {
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

	err := resourceAwsOpsworksStackValidate(d)
	if err != nil {
		return err
	}

	attributes := make(map[string]*string)

	req := &opsworks.UpdateStackInput{
		CustomJSON:                aws.String(d.Get("custom_json").(string)),
		DefaultInstanceProfileARN: aws.String(d.Get("default_instance_profile_arn").(string)),
		DefaultRootDeviceType:     aws.String(d.Get("default_root_device_type").(string)),
		DefaultSSHKeyName:         aws.String(d.Get("default_ssh_key_name").(string)),
		Name:                      aws.String(d.Get("name").(string)),
		ServiceRoleARN:            aws.String(d.Get("service_role_arn").(string)),
		StackID:                   aws.String(d.Id()),
		UseCustomCookbooks:        aws.Boolean(d.Get("use_custom_cookbooks").(bool)),
		UseOpsWorksSecurityGroups: aws.Boolean(d.Get("use_opsworks_security_groups").(bool)),
		Attributes:                &attributes,
		CustomCookbooksSource:     resourceAwsOpsworksStackCustomCookbooksSource(d),
	}
	if v, ok := d.GetOk("default_os"); ok {
		req.DefaultOs = aws.String(v.(string))
	}
	if v, ok := d.GetOk("default_subnet_id"); ok {
		req.DefaultSubnetID = aws.String(v.(string))
	}
	if v, ok := d.GetOk("default_availability_zone"); ok {
		req.DefaultAvailabilityZone = aws.String(v.(string))
	}
	if v, ok := d.GetOk("hostname_theme"); ok {
		req.HostnameTheme = aws.String(v.(string))
	}
	if v, ok := d.GetOk("color"); ok {
		attributes["Color"] = aws.String(v.(string))
	}
	req.ChefConfiguration = &opsworks.ChefConfiguration{
		BerkshelfVersion: aws.String(d.Get("berkshelf_version").(string)),
		ManageBerkshelf:  aws.Boolean(d.Get("manage_berkshelf").(bool)),
	}
	req.ConfigurationManager = &opsworks.StackConfigurationManager{
		Name:    aws.String(d.Get("configuration_manager_name").(string)),
		Version: aws.String(d.Get("configuration_manager_version").(string)),
	}

	log.Printf("[DEBUG] Updating OpsWorks stack: %s", d.Id())

	_, err = client.UpdateStack(req)
	if err != nil {
		return err
	}

	return resourceAwsOpsworksStackRead(d, meta)
}

func resourceAwsOpsworksStackDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*AWSClient).opsworksconn

	req := &opsworks.DeleteStackInput{
		StackID: aws.String(d.Id()),
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
	if _, ok := d.GetOk("vpc_id"); ok {
		log.Print("[INFO] Waiting for Opsworks built-in security groups to be deleted")
		time.Sleep(30 * time.Second)
	}

	return nil
}
