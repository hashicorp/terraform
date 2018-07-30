package aws

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/opsworks"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
)

func resourceAwsOpsworksApplication() *schema.Resource {
	return &schema.Resource{

		Create: resourceAwsOpsworksApplicationCreate,
		Read:   resourceAwsOpsworksApplicationRead,
		Update: resourceAwsOpsworksApplicationUpdate,
		Delete: resourceAwsOpsworksApplicationDelete,
		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"short_name": {
				Type:     schema.TypeString,
				Computed: true,
				Optional: true,
			},
			"type": {
				Type:     schema.TypeString,
				Required: true,
				ValidateFunc: validation.StringInSlice([]string{
					opsworks.AppTypeAwsFlowRuby,
					opsworks.AppTypeJava,
					opsworks.AppTypeRails,
					opsworks.AppTypePhp,
					opsworks.AppTypeNodejs,
					opsworks.AppTypeStatic,
					opsworks.AppTypeOther,
				}, false),
			},
			"stack_id": {
				Type:     schema.TypeString,
				Required: true,
			},
			// TODO: the following 4 vals are really part of the Attributes array. We should validate that only ones relevant to the chosen type are set, perhaps. (what is the default type? how do they map?)
			"document_root": {
				Type:     schema.TypeString,
				Optional: true,
				//Default:  "public",
			},
			"rails_env": {
				Type:     schema.TypeString,
				Optional: true,
				//Default:  "production",
			},
			"auto_bundle_on_deploy": {
				Type:     schema.TypeString,
				Optional: true,
				//Default:  true,
			},
			"aws_flow_ruby_settings": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"app_source": {
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
							Optional: true,
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
			// AutoSelectOpsworksMysqlInstance, OpsworksMysqlInstance, or RdsDbInstance.
			// anything beside auto select will lead into failure in case the instance doesn't exist
			// XXX: validation?
			"data_source_type": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"data_source_database_name": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"data_source_arn": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"description": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"domains": {
				Type:     schema.TypeList,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"environment": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"key": {
							Type:     schema.TypeString,
							Required: true,
						},
						"value": {
							Type:     schema.TypeString,
							Required: true,
						},
						"secure": {
							Type:     schema.TypeBool,
							Optional: true,
							Default:  true,
						},
					},
				},
			},
			"enable_ssl": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"ssl_configuration": {
				Type:     schema.TypeList,
				Optional: true,
				//Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"certificate": {
							Type:     schema.TypeString,
							Required: true,
							StateFunc: func(v interface{}) string {
								switch v.(type) {
								case string:
									return strings.TrimSpace(v.(string))
								default:
									return ""
								}
							},
						},
						"private_key": {
							Type:      schema.TypeString,
							Required:  true,
							Sensitive: true,
							StateFunc: func(v interface{}) string {
								switch v.(type) {
								case string:
									return strings.TrimSpace(v.(string))
								default:
									return ""
								}
							},
						},
						"chain": {
							Type:     schema.TypeString,
							Optional: true,
							StateFunc: func(v interface{}) string {
								switch v.(type) {
								case string:
									return strings.TrimSpace(v.(string))
								default:
									return ""
								}
							},
						},
					},
				},
			},
		},
	}
}

func resourceAwsOpsworksApplicationValidate(d *schema.ResourceData) error {
	appSourceCount := d.Get("app_source.#").(int)
	if appSourceCount > 1 {
		return fmt.Errorf("Only one app_source is permitted.")
	}

	sslCount := d.Get("ssl_configuration.#").(int)
	if sslCount > 1 {
		return fmt.Errorf("Only one ssl_configuration is permitted.")
	}

	if d.Get("type") == opsworks.AppTypeNodejs || d.Get("type") == opsworks.AppTypeJava {
		// allowed attributes: none
		if d.Get("document_root").(string) != "" || d.Get("rails_env").(string) != "" || d.Get("auto_bundle_on_deploy").(string) != "" || d.Get("aws_flow_ruby_settings").(string) != "" {
			return fmt.Errorf("No additional attributes are allowed for app type '%s'.", d.Get("type").(string))
		}
	} else if d.Get("type") == opsworks.AppTypeRails {
		// allowed attributes: document_root, rails_env, auto_bundle_on_deploy
		if d.Get("aws_flow_ruby_settings").(string) != "" {
			return fmt.Errorf("Only 'document_root, rails_env, auto_bundle_on_deploy' are allowed for app type '%s'.", opsworks.AppTypeRails)
		}
		// rails_env is required
		if _, ok := d.GetOk("rails_env"); !ok {
			return fmt.Errorf("Set rails_env must be set if type is set to rails.")
		}
	} else if d.Get("type") == opsworks.AppTypePhp || d.Get("type") == opsworks.AppTypeStatic || d.Get("type") == opsworks.AppTypeOther {
		log.Printf("[DEBUG] the app type is : %s", d.Get("type").(string))
		log.Printf("[DEBUG] the attributes are: document_root '%s', rails_env '%s', auto_bundle_on_deploy '%s', aws_flow_ruby_settings '%s'", d.Get("document_root").(string), d.Get("rails_env").(string), d.Get("auto_bundle_on_deploy").(string), d.Get("aws_flow_ruby_settings").(string))
		// allowed attributes: document_root
		if d.Get("rails_env").(string) != "" || d.Get("auto_bundle_on_deploy").(string) != "" || d.Get("aws_flow_ruby_settings").(string) != "" {
			return fmt.Errorf("Only 'document_root' is allowed for app type '%s'.", d.Get("type").(string))
		}
	} else if d.Get("type") == opsworks.AppTypeAwsFlowRuby {
		// allowed attributes: aws_flow_ruby_settings
		if d.Get("document_root").(string) != "" || d.Get("rails_env").(string) != "" || d.Get("auto_bundle_on_deploy").(string) != "" {
			return fmt.Errorf("Only 'aws_flow_ruby_settings' is allowed for app type '%s'.", d.Get("type").(string))
		}
	}

	return nil
}

func resourceAwsOpsworksApplicationRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*AWSClient).opsworksconn

	req := &opsworks.DescribeAppsInput{
		AppIds: []*string{
			aws.String(d.Id()),
		},
	}

	log.Printf("[DEBUG] Reading OpsWorks app: %s", d.Id())

	resp, err := client.DescribeApps(req)
	if err != nil {
		if awserr, ok := err.(awserr.Error); ok {
			if awserr.Code() == "ResourceNotFoundException" {
				log.Printf("[INFO] App not found: %s", d.Id())
				d.SetId("")
				return nil
			}
		}
		return err
	}

	app := resp.Apps[0]

	d.Set("name", app.Name)
	d.Set("stack_id", app.StackId)
	d.Set("type", app.Type)
	d.Set("description", app.Description)
	d.Set("domains", flattenStringList(app.Domains))
	d.Set("enable_ssl", app.EnableSsl)
	resourceAwsOpsworksSetApplicationSsl(d, app.SslConfiguration)
	resourceAwsOpsworksSetApplicationSource(d, app.AppSource)
	resourceAwsOpsworksSetApplicationDataSources(d, app.DataSources)
	resourceAwsOpsworksSetApplicationEnvironmentVariable(d, app.Environment)
	resourceAwsOpsworksSetApplicationAttributes(d, app.Attributes)
	return nil
}

func resourceAwsOpsworksApplicationCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*AWSClient).opsworksconn

	err := resourceAwsOpsworksApplicationValidate(d)
	if err != nil {
		return err
	}

	req := &opsworks.CreateAppInput{
		Name:             aws.String(d.Get("name").(string)),
		Shortname:        aws.String(d.Get("short_name").(string)),
		StackId:          aws.String(d.Get("stack_id").(string)),
		Type:             aws.String(d.Get("type").(string)),
		Description:      aws.String(d.Get("description").(string)),
		Domains:          expandStringList(d.Get("domains").([]interface{})),
		EnableSsl:        aws.Bool(d.Get("enable_ssl").(bool)),
		SslConfiguration: resourceAwsOpsworksApplicationSsl(d),
		AppSource:        resourceAwsOpsworksApplicationSource(d),
		DataSources:      resourceAwsOpsworksApplicationDataSources(d),
		Environment:      resourceAwsOpsworksApplicationEnvironmentVariable(d),
		Attributes:       resourceAwsOpsworksApplicationAttributes(d),
	}

	var resp *opsworks.CreateAppOutput
	err = resource.Retry(2*time.Minute, func() *resource.RetryError {
		var cerr error
		resp, cerr = client.CreateApp(req)
		if cerr != nil {
			log.Printf("[INFO] client error")
			if opserr, ok := cerr.(awserr.Error); ok {
				// XXX: handle errors
				log.Printf("[ERROR] OpsWorks error: %s message: %s", opserr.Code(), opserr.Message())
				return resource.RetryableError(cerr)
			}
			return resource.NonRetryableError(cerr)
		}
		return nil
	})

	if err != nil {
		return err
	}

	appID := *resp.AppId
	d.SetId(appID)

	return resourceAwsOpsworksApplicationRead(d, meta)
}

func resourceAwsOpsworksApplicationUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*AWSClient).opsworksconn

	err := resourceAwsOpsworksApplicationValidate(d)
	if err != nil {
		return err
	}

	req := &opsworks.UpdateAppInput{
		AppId:            aws.String(d.Id()),
		Name:             aws.String(d.Get("name").(string)),
		Type:             aws.String(d.Get("type").(string)),
		Description:      aws.String(d.Get("description").(string)),
		Domains:          expandStringList(d.Get("domains").([]interface{})),
		EnableSsl:        aws.Bool(d.Get("enable_ssl").(bool)),
		SslConfiguration: resourceAwsOpsworksApplicationSsl(d),
		AppSource:        resourceAwsOpsworksApplicationSource(d),
		DataSources:      resourceAwsOpsworksApplicationDataSources(d),
		Environment:      resourceAwsOpsworksApplicationEnvironmentVariable(d),
		Attributes:       resourceAwsOpsworksApplicationAttributes(d),
	}

	log.Printf("[DEBUG] Updating OpsWorks layer: %s", d.Id())

	err = resource.Retry(2*time.Minute, func() *resource.RetryError {
		_, cerr := client.UpdateApp(req)
		if cerr != nil {
			log.Printf("[INFO] client error")
			if opserr, ok := cerr.(awserr.Error); ok {
				// XXX: handle errors
				log.Printf("[ERROR] OpsWorks error: %s message: %s", opserr.Code(), opserr.Message())
				return resource.NonRetryableError(cerr)
			}
			return resource.RetryableError(cerr)
		}
		return nil
	})

	if err != nil {
		return err
	}
	return resourceAwsOpsworksApplicationRead(d, meta)
}

func resourceAwsOpsworksApplicationDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*AWSClient).opsworksconn

	req := &opsworks.DeleteAppInput{
		AppId: aws.String(d.Id()),
	}

	log.Printf("[DEBUG] Deleting OpsWorks application: %s", d.Id())

	_, err := client.DeleteApp(req)
	return err
}

func resourceAwsOpsworksSetApplicationEnvironmentVariable(d *schema.ResourceData, v []*opsworks.EnvironmentVariable) {
	log.Printf("[DEBUG] envs: %s %d", v, len(v))
	if len(v) == 0 {
		d.Set("environment", nil)
		return
	}
	newValue := make([]*map[string]interface{}, len(v))

	for i := 0; i < len(v); i++ {
		config := v[i]
		data := make(map[string]interface{})
		newValue[i] = &data

		if config.Key != nil {
			data["key"] = *config.Key
		}
		if config.Value != nil {
			data["value"] = *config.Value
		}
		if config.Secure != nil {

			if bool(*config.Secure) {
				data["secure"] = &opsworksTrueString
			} else {
				data["secure"] = &opsworksFalseString
			}
		}
		log.Printf("[DEBUG] v: %s", data)
	}

	d.Set("environment", newValue)
}

func resourceAwsOpsworksApplicationEnvironmentVariable(d *schema.ResourceData) []*opsworks.EnvironmentVariable {
	environmentVariables := d.Get("environment").(*schema.Set).List()
	result := make([]*opsworks.EnvironmentVariable, len(environmentVariables))

	for i := 0; i < len(environmentVariables); i++ {
		env := environmentVariables[i].(map[string]interface{})

		result[i] = &opsworks.EnvironmentVariable{
			Key:    aws.String(env["key"].(string)),
			Value:  aws.String(env["value"].(string)),
			Secure: aws.Bool(env["secure"].(bool)),
		}
	}
	return result
}

func resourceAwsOpsworksApplicationSource(d *schema.ResourceData) *opsworks.Source {
	count := d.Get("app_source.#").(int)
	if count == 0 {
		return nil
	}

	return &opsworks.Source{
		Type:     aws.String(d.Get("app_source.0.type").(string)),
		Url:      aws.String(d.Get("app_source.0.url").(string)),
		Username: aws.String(d.Get("app_source.0.username").(string)),
		Password: aws.String(d.Get("app_source.0.password").(string)),
		Revision: aws.String(d.Get("app_source.0.revision").(string)),
		SshKey:   aws.String(d.Get("app_source.0.ssh_key").(string)),
	}
}

func resourceAwsOpsworksSetApplicationSource(d *schema.ResourceData, v *opsworks.Source) {
	nv := make([]interface{}, 0, 1)
	if v != nil {
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
		if v.Password != nil {
			m["password"] = *v.Password
		}
		if v.Revision != nil {
			m["revision"] = *v.Revision
		}
		nv = append(nv, m)
	}

	err := d.Set("app_source", nv)
	if err != nil {
		// should never happen
		panic(err)
	}
}

func resourceAwsOpsworksApplicationDataSources(d *schema.ResourceData) []*opsworks.DataSource {
	arn := d.Get("data_source_arn").(string)
	databaseName := d.Get("data_source_database_name").(string)
	databaseType := d.Get("data_source_type").(string)

	result := make([]*opsworks.DataSource, 1)

	if len(arn) > 0 || len(databaseName) > 0 || len(databaseType) > 0 {
		result[0] = &opsworks.DataSource{
			Arn:          aws.String(arn),
			DatabaseName: aws.String(databaseName),
			Type:         aws.String(databaseType),
		}
	}
	return result
}

func resourceAwsOpsworksSetApplicationDataSources(d *schema.ResourceData, v []*opsworks.DataSource) {
	d.Set("data_source_arn", nil)
	d.Set("data_source_database_name", nil)
	d.Set("data_source_type", nil)

	if len(v) == 0 {
		return
	}

	d.Set("data_source_arn", v[0].Arn)
	d.Set("data_source_database_name", v[0].DatabaseName)
	d.Set("data_source_type", v[0].Type)
}

func resourceAwsOpsworksApplicationSsl(d *schema.ResourceData) *opsworks.SslConfiguration {
	count := d.Get("ssl_configuration.#").(int)
	if count == 0 {
		return nil
	}

	return &opsworks.SslConfiguration{
		PrivateKey:  aws.String(d.Get("ssl_configuration.0.private_key").(string)),
		Certificate: aws.String(d.Get("ssl_configuration.0.certificate").(string)),
		Chain:       aws.String(d.Get("ssl_configuration.0.chain").(string)),
	}
}

func resourceAwsOpsworksSetApplicationSsl(d *schema.ResourceData, v *opsworks.SslConfiguration) {
	nv := make([]interface{}, 0, 1)
	set := false
	if v != nil {
		m := make(map[string]interface{})
		if v.PrivateKey != nil {
			m["private_key"] = *v.PrivateKey
			set = true
		}
		if v.Certificate != nil {
			m["certificate"] = *v.Certificate
			set = true
		}
		if v.Chain != nil {
			m["chain"] = *v.Chain
			set = true
		}
		if set {
			nv = append(nv, m)
		}
	}

	err := d.Set("ssl_configuration", nv)
	if err != nil {
		// should never happen
		panic(err)
	}
}

func resourceAwsOpsworksApplicationAttributes(d *schema.ResourceData) map[string]*string {
	attributes := make(map[string]*string)

	if val := d.Get("document_root").(string); len(val) > 0 {
		attributes[opsworks.AppAttributesKeysDocumentRoot] = aws.String(val)
	}
	if val := d.Get("aws_flow_ruby_settings").(string); len(val) > 0 {
		attributes[opsworks.AppAttributesKeysAwsFlowRubySettings] = aws.String(val)
	}
	if val := d.Get("rails_env").(string); len(val) > 0 {
		attributes[opsworks.AppAttributesKeysRailsEnv] = aws.String(val)
	}
	if val := d.Get("auto_bundle_on_deploy").(string); len(val) > 0 {
		if val == "1" {
			val = "true"
		} else if val == "0" {
			val = "false"
		}
		attributes[opsworks.AppAttributesKeysAutoBundleOnDeploy] = aws.String(val)
	}

	return attributes
}

func resourceAwsOpsworksSetApplicationAttributes(d *schema.ResourceData, v map[string]*string) {
	d.Set("document_root", nil)
	d.Set("rails_env", nil)
	d.Set("aws_flow_ruby_settings", nil)
	d.Set("auto_bundle_on_deploy", nil)

	if d.Get("type") == opsworks.AppTypeNodejs || d.Get("type") == opsworks.AppTypeJava {
		return
	} else if d.Get("type") == opsworks.AppTypeRails {
		if val, ok := v[opsworks.AppAttributesKeysDocumentRoot]; ok {
			d.Set("document_root", val)
		}
		if val, ok := v[opsworks.AppAttributesKeysRailsEnv]; ok {
			d.Set("rails_env", val)
		}
		if val, ok := v[opsworks.AppAttributesKeysAutoBundleOnDeploy]; ok {
			d.Set("auto_bundle_on_deploy", val)
		}
		return
	} else if d.Get("type") == opsworks.AppTypePhp || d.Get("type") == opsworks.AppTypeStatic || d.Get("type") == opsworks.AppTypeOther {
		if val, ok := v[opsworks.AppAttributesKeysDocumentRoot]; ok {
			d.Set("document_root", val)
		}
		return
	} else if d.Get("type") == opsworks.AppTypeAwsFlowRuby {
		if val, ok := v[opsworks.AppAttributesKeysAwsFlowRubySettings]; ok {
			d.Set("aws_flow_ruby_settings", val)
		}
		return
	}

	return
}
