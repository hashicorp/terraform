package aws

import (
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	elasticsearch "github.com/aws/aws-sdk-go/service/elasticsearchservice"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/structure"
	"github.com/hashicorp/terraform/helper/validation"
)

func resourceAwsElasticSearchDomain() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsElasticSearchDomainCreate,
		Read:   resourceAwsElasticSearchDomainRead,
		Update: resourceAwsElasticSearchDomainUpdate,
		Delete: resourceAwsElasticSearchDomainDelete,
		Importer: &schema.ResourceImporter{
			State: resourceAwsElasticSearchDomainImport,
		},

		Schema: map[string]*schema.Schema{
			"access_policies": {
				Type:             schema.TypeString,
				Optional:         true,
				Computed:         true,
				ValidateFunc:     validateJsonString,
				DiffSuppressFunc: suppressEquivalentAwsPolicyDiffs,
			},
			"advanced_options": {
				Type:     schema.TypeMap,
				Optional: true,
				Computed: true,
			},
			"domain_name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
				ValidateFunc: func(v interface{}, k string) (ws []string, errors []error) {
					value := v.(string)
					if !regexp.MustCompile(`^[a-z][0-9a-z\-]{2,27}$`).MatchString(value) {
						errors = append(errors, fmt.Errorf(
							"%q must start with a lowercase alphabet and be at least 3 and no more than 28 characters long. Valid characters are a-z (lowercase letters), 0-9, and - (hyphen).", k))
					}
					return
				},
			},
			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"domain_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"endpoint": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"kibana_endpoint": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"ebs_options": {
				Type:     schema.TypeList,
				Optional: true,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"ebs_enabled": {
							Type:     schema.TypeBool,
							Required: true,
						},
						"iops": {
							Type:     schema.TypeInt,
							Optional: true,
						},
						"volume_size": {
							Type:     schema.TypeInt,
							Optional: true,
						},
						"volume_type": {
							Type:     schema.TypeString,
							Optional: true,
							Computed: true,
						},
					},
				},
			},
			"encrypt_at_rest": {
				Type:     schema.TypeList,
				Optional: true,
				Computed: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"enabled": {
							Type:     schema.TypeBool,
							Required: true,
							ForceNew: true,
						},
						"kms_key_id": {
							Type:             schema.TypeString,
							Optional:         true,
							Computed:         true,
							ForceNew:         true,
							DiffSuppressFunc: suppressEquivalentKmsKeyIds,
						},
					},
				},
			},
			"cluster_config": {
				Type:     schema.TypeList,
				Optional: true,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"dedicated_master_count": {
							Type:     schema.TypeInt,
							Optional: true,
						},
						"dedicated_master_enabled": {
							Type:     schema.TypeBool,
							Optional: true,
							Default:  false,
						},
						"dedicated_master_type": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"instance_count": {
							Type:     schema.TypeInt,
							Optional: true,
							Default:  1,
						},
						"instance_type": {
							Type:     schema.TypeString,
							Optional: true,
							Default:  "m3.medium.elasticsearch",
						},
						"zone_awareness_enabled": {
							Type:     schema.TypeBool,
							Optional: true,
						},
					},
				},
			},
			"snapshot_options": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"automated_snapshot_start_hour": {
							Type:     schema.TypeInt,
							Required: true,
						},
					},
				},
			},
			"vpc_options": {
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"availability_zones": {
							Type:     schema.TypeSet,
							Computed: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
							Set:      schema.HashString,
						},
						"security_group_ids": {
							Type:     schema.TypeSet,
							Optional: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
							Set:      schema.HashString,
						},
						"subnet_ids": {
							Type:     schema.TypeSet,
							Optional: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
							Set:      schema.HashString,
						},
						"vpc_id": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},
			"log_publishing_options": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"log_type": {
							Type:     schema.TypeString,
							Required: true,
							ValidateFunc: validation.StringInSlice([]string{
								elasticsearch.LogTypeIndexSlowLogs,
								elasticsearch.LogTypeSearchSlowLogs,
							}, false),
						},
						"cloudwatch_log_group_arn": {
							Type:     schema.TypeString,
							Required: true,
						},
						"enabled": {
							Type:     schema.TypeBool,
							Optional: true,
							Default:  true,
						},
					},
				},
			},
			"elasticsearch_version": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "1.5",
				ForceNew: true,
			},

			"tags": tagsSchema(),
		},
	}
}

func resourceAwsElasticSearchDomainImport(
	d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	d.Set("domain_name", d.Id())
	return []*schema.ResourceData{d}, nil
}

// This would be created automatically if the domain is created via Console
// see http://docs.aws.amazon.com/elasticsearch-service/latest/developerguide/es-vpc.html#es-enabling-slr
func createAwsElasticsearchIAMServiceRoleIfMissing(meta interface{}) error {
	serviceRoleName := "AWSServiceRoleForAmazonElasticsearchService"
	serviceName := "es.amazonaws.com"

	conn := meta.(*AWSClient).iamconn

	getRequest := &iam.GetRoleInput{
		RoleName: aws.String(serviceRoleName),
	}
	_, err := conn.GetRole(getRequest)
	if err != nil {
		if isAWSErr(err, iam.ErrCodeNoSuchEntityException, "Role not found") {
			createRequest := &iam.CreateServiceLinkedRoleInput{
				AWSServiceName: aws.String(serviceName),
			}
			_, err := conn.CreateServiceLinkedRole(createRequest)
			if err != nil {
				if isAWSErr(err, iam.ErrCodeInvalidInputException, "has been taken in this account") {
					return nil
				}
				return fmt.Errorf("Error creating IAM Service-Linked Role %s: %s", serviceRoleName, err)
			}
			return nil
		}
		return fmt.Errorf("Error reading IAM Role %s: %s", serviceRoleName, err)
	}
	return nil
}

func resourceAwsElasticSearchDomainCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).esconn

	// The API doesn't check for duplicate names
	// so w/out this check Create would act as upsert
	// and might cause duplicate domain to appear in state
	resp, err := conn.DescribeElasticsearchDomain(&elasticsearch.DescribeElasticsearchDomainInput{
		DomainName: aws.String(d.Get("domain_name").(string)),
	})
	if err == nil {
		return fmt.Errorf("ElasticSearch domain %q already exists", *resp.DomainStatus.DomainName)
	}

	input := elasticsearch.CreateElasticsearchDomainInput{
		DomainName:           aws.String(d.Get("domain_name").(string)),
		ElasticsearchVersion: aws.String(d.Get("elasticsearch_version").(string)),
	}

	if v, ok := d.GetOk("access_policies"); ok {
		input.AccessPolicies = aws.String(v.(string))
	}

	if v, ok := d.GetOk("advanced_options"); ok {
		input.AdvancedOptions = stringMapToPointers(v.(map[string]interface{}))
	}

	if v, ok := d.GetOk("ebs_options"); ok {
		options := v.([]interface{})

		if len(options) > 1 {
			return fmt.Errorf("Only a single ebs_options block is expected")
		} else if len(options) == 1 {
			if options[0] == nil {
				return fmt.Errorf("At least one field is expected inside ebs_options")
			}

			s := options[0].(map[string]interface{})
			input.EBSOptions = expandESEBSOptions(s)
		}
	}

	if v, ok := d.GetOk("encrypt_at_rest"); ok {
		options := v.([]interface{})
		if options[0] == nil {
			return fmt.Errorf("At least one field is expected inside encrypt_at_rest")
		}

		s := options[0].(map[string]interface{})
		input.EncryptionAtRestOptions = expandESEncryptAtRestOptions(s)
	}

	if v, ok := d.GetOk("cluster_config"); ok {
		config := v.([]interface{})

		if len(config) > 1 {
			return fmt.Errorf("Only a single cluster_config block is expected")
		} else if len(config) == 1 {
			if config[0] == nil {
				return fmt.Errorf("At least one field is expected inside cluster_config")
			}
			m := config[0].(map[string]interface{})
			input.ElasticsearchClusterConfig = expandESClusterConfig(m)
		}
	}

	if v, ok := d.GetOk("snapshot_options"); ok {
		options := v.([]interface{})

		if len(options) > 1 {
			return fmt.Errorf("Only a single snapshot_options block is expected")
		} else if len(options) == 1 {
			if options[0] == nil {
				return fmt.Errorf("At least one field is expected inside snapshot_options")
			}

			o := options[0].(map[string]interface{})

			snapshotOptions := elasticsearch.SnapshotOptions{
				AutomatedSnapshotStartHour: aws.Int64(int64(o["automated_snapshot_start_hour"].(int))),
			}

			input.SnapshotOptions = &snapshotOptions
		}
	}

	if v, ok := d.GetOk("vpc_options"); ok {
		err = createAwsElasticsearchIAMServiceRoleIfMissing(meta)
		if err != nil {
			return err
		}

		options := v.([]interface{})
		if options[0] == nil {
			return fmt.Errorf("At least one field is expected inside vpc_options")
		}

		s := options[0].(map[string]interface{})
		input.VPCOptions = expandESVPCOptions(s)
	}

	if v, ok := d.GetOk("log_publishing_options"); ok {
		input.LogPublishingOptions = make(map[string]*elasticsearch.LogPublishingOption)
		options := v.(*schema.Set).List()
		for _, vv := range options {
			lo := vv.(map[string]interface{})
			input.LogPublishingOptions[lo["log_type"].(string)] = &elasticsearch.LogPublishingOption{
				CloudWatchLogsLogGroupArn: aws.String(lo["cloudwatch_log_group_arn"].(string)),
				Enabled:                   aws.Bool(lo["enabled"].(bool)),
			}
		}
	}

	log.Printf("[DEBUG] Creating ElasticSearch domain: %s", input)

	// IAM Roles can take some time to propagate if set in AccessPolicies and created in the same terraform
	var out *elasticsearch.CreateElasticsearchDomainOutput
	err = resource.Retry(30*time.Second, func() *resource.RetryError {
		var err error
		out, err = conn.CreateElasticsearchDomain(&input)
		if err != nil {
			if isAWSErr(err, "InvalidTypeException", "Error setting policy") {
				log.Printf("[DEBUG] Retrying creation of ElasticSearch domain %s", *input.DomainName)
				return resource.RetryableError(err)
			}
			if isAWSErr(err, "ValidationException", "enable a service-linked role to give Amazon ES permissions") {
				return resource.RetryableError(err)
			}
			if isAWSErr(err, "ValidationException", "Domain is still being deleted") {
				return resource.RetryableError(err)
			}

			return resource.NonRetryableError(err)
		}
		return nil
	})

	if err != nil {
		return err
	}

	d.SetId(*out.DomainStatus.ARN)

	// Whilst the domain is being created, we can initialise the tags.
	// This should mean that if the creation fails (eg because your token expired
	// whilst the operation is being performed), we still get the required tags on
	// the resources.
	tags := tagsFromMapElasticsearchService(d.Get("tags").(map[string]interface{}))

	if err := setTagsElasticsearchService(conn, d, *out.DomainStatus.ARN); err != nil {
		return err
	}

	d.Set("tags", tagsToMapElasticsearchService(tags))
	d.SetPartial("tags")

	log.Printf("[DEBUG] Waiting for ElasticSearch domain %q to be created", d.Id())
	err = waitForElasticSearchDomainCreation(conn, d.Get("domain_name").(string), d.Id())
	if err != nil {
		return err
	}
	d.Partial(false)

	log.Printf("[DEBUG] ElasticSearch domain %q created", d.Id())

	return resourceAwsElasticSearchDomainRead(d, meta)
}

func waitForElasticSearchDomainCreation(conn *elasticsearch.ElasticsearchService, domainName, arn string) error {
	return resource.Retry(60*time.Minute, func() *resource.RetryError {
		out, err := conn.DescribeElasticsearchDomain(&elasticsearch.DescribeElasticsearchDomainInput{
			DomainName: aws.String(domainName),
		})
		if err != nil {
			return resource.NonRetryableError(err)
		}

		if !*out.DomainStatus.Processing && (out.DomainStatus.Endpoint != nil || out.DomainStatus.Endpoints != nil) {
			return nil
		}

		return resource.RetryableError(
			fmt.Errorf("%q: Timeout while waiting for the domain to be created", arn))
	})
}

func resourceAwsElasticSearchDomainRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).esconn

	out, err := conn.DescribeElasticsearchDomain(&elasticsearch.DescribeElasticsearchDomainInput{
		DomainName: aws.String(d.Get("domain_name").(string)),
	})
	if err != nil {
		if ec2err, ok := err.(awserr.Error); ok && ec2err.Code() == "ResourceNotFoundException" {
			log.Printf("[INFO] ElasticSearch Domain %q not found", d.Get("domain_name").(string))
			d.SetId("")
			return nil
		}
		return err
	}

	log.Printf("[DEBUG] Received ElasticSearch domain: %s", out)

	ds := out.DomainStatus

	if ds.AccessPolicies != nil && *ds.AccessPolicies != "" {
		policies, err := structure.NormalizeJsonString(*ds.AccessPolicies)
		if err != nil {
			return errwrap.Wrapf("access policies contain an invalid JSON: {{err}}", err)
		}
		d.Set("access_policies", policies)
	}
	err = d.Set("advanced_options", pointersMapToStringList(ds.AdvancedOptions))
	if err != nil {
		return err
	}
	d.SetId(*ds.ARN)
	d.Set("domain_id", ds.DomainId)
	d.Set("domain_name", ds.DomainName)
	d.Set("elasticsearch_version", ds.ElasticsearchVersion)

	err = d.Set("ebs_options", flattenESEBSOptions(ds.EBSOptions))
	if err != nil {
		return err
	}
	err = d.Set("encrypt_at_rest", flattenESEncryptAtRestOptions(ds.EncryptionAtRestOptions))
	if err != nil {
		return err
	}
	err = d.Set("cluster_config", flattenESClusterConfig(ds.ElasticsearchClusterConfig))
	if err != nil {
		return err
	}
	if ds.SnapshotOptions != nil {
		d.Set("snapshot_options", map[string]interface{}{
			"automated_snapshot_start_hour": *ds.SnapshotOptions.AutomatedSnapshotStartHour,
		})
	}
	if ds.VPCOptions != nil {
		err = d.Set("vpc_options", flattenESVPCDerivedInfo(ds.VPCOptions))
		if err != nil {
			return err
		}
		endpoints := pointersMapToStringList(ds.Endpoints)
		err = d.Set("endpoint", endpoints["vpc"])
		if err != nil {
			return err
		}
		d.Set("kibana_endpoint", getKibanaEndpoint(d))
		if ds.Endpoint != nil {
			return fmt.Errorf("%q: Elasticsearch domain in VPC expected to have null Endpoint value", d.Id())
		}
	} else {
		if ds.Endpoint != nil {
			d.Set("endpoint", *ds.Endpoint)
			d.Set("kibana_endpoint", getKibanaEndpoint(d))
		}
		if ds.Endpoints != nil {
			return fmt.Errorf("%q: Elasticsearch domain not in VPC expected to have null Endpoints value", d.Id())
		}
	}

	if ds.LogPublishingOptions != nil {
		m := make([]map[string]interface{}, 0)
		for k, val := range ds.LogPublishingOptions {
			mm := map[string]interface{}{}
			mm["log_type"] = k
			if val.CloudWatchLogsLogGroupArn != nil {
				mm["cloudwatch_log_group_arn"] = *val.CloudWatchLogsLogGroupArn
			}
			mm["enabled"] = *val.Enabled
			m = append(m, mm)
		}
		d.Set("log_publishing_options", m)
	}

	d.Set("arn", ds.ARN)

	listOut, err := conn.ListTags(&elasticsearch.ListTagsInput{
		ARN: ds.ARN,
	})

	if err != nil {
		return err
	}
	var est []*elasticsearch.Tag
	if len(listOut.TagList) > 0 {
		est = listOut.TagList
	}

	d.Set("tags", tagsToMapElasticsearchService(est))

	return nil
}

func resourceAwsElasticSearchDomainUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).esconn

	d.Partial(true)

	if err := setTagsElasticsearchService(conn, d, d.Id()); err != nil {
		return err
	}

	d.SetPartial("tags")

	input := elasticsearch.UpdateElasticsearchDomainConfigInput{
		DomainName: aws.String(d.Get("domain_name").(string)),
	}

	if d.HasChange("access_policies") {
		input.AccessPolicies = aws.String(d.Get("access_policies").(string))
	}

	if d.HasChange("advanced_options") {
		input.AdvancedOptions = stringMapToPointers(d.Get("advanced_options").(map[string]interface{}))
	}

	if d.HasChange("ebs_options") || d.HasChange("cluster_config") {
		options := d.Get("ebs_options").([]interface{})

		if len(options) > 1 {
			return fmt.Errorf("Only a single ebs_options block is expected")
		} else if len(options) == 1 {
			s := options[0].(map[string]interface{})
			input.EBSOptions = expandESEBSOptions(s)
		}

		if d.HasChange("cluster_config") {
			config := d.Get("cluster_config").([]interface{})

			if len(config) > 1 {
				return fmt.Errorf("Only a single cluster_config block is expected")
			} else if len(config) == 1 {
				m := config[0].(map[string]interface{})
				input.ElasticsearchClusterConfig = expandESClusterConfig(m)
			}
		}

	}

	if d.HasChange("snapshot_options") {
		options := d.Get("snapshot_options").([]interface{})

		if len(options) > 1 {
			return fmt.Errorf("Only a single snapshot_options block is expected")
		} else if len(options) == 1 {
			o := options[0].(map[string]interface{})

			snapshotOptions := elasticsearch.SnapshotOptions{
				AutomatedSnapshotStartHour: aws.Int64(int64(o["automated_snapshot_start_hour"].(int))),
			}

			input.SnapshotOptions = &snapshotOptions
		}
	}

	if d.HasChange("vpc_options") {
		options := d.Get("vpc_options").([]interface{})
		s := options[0].(map[string]interface{})
		input.VPCOptions = expandESVPCOptions(s)
	}

	if d.HasChange("log_publishing_options") {
		input.LogPublishingOptions = make(map[string]*elasticsearch.LogPublishingOption)
		options := d.Get("log_publishing_options").(*schema.Set).List()
		for _, vv := range options {
			lo := vv.(map[string]interface{})
			input.LogPublishingOptions[lo["log_type"].(string)] = &elasticsearch.LogPublishingOption{
				CloudWatchLogsLogGroupArn: aws.String(lo["cloudwatch_log_group_arn"].(string)),
				Enabled:                   aws.Bool(lo["enabled"].(bool)),
			}
		}
	}

	_, err := conn.UpdateElasticsearchDomainConfig(&input)
	if err != nil {
		return err
	}

	err = resource.Retry(60*time.Minute, func() *resource.RetryError {
		out, err := conn.DescribeElasticsearchDomain(&elasticsearch.DescribeElasticsearchDomainInput{
			DomainName: aws.String(d.Get("domain_name").(string)),
		})
		if err != nil {
			return resource.NonRetryableError(err)
		}

		if *out.DomainStatus.Processing == false {
			return nil
		}

		return resource.RetryableError(
			fmt.Errorf("%q: Timeout while waiting for changes to be processed", d.Id()))
	})
	if err != nil {
		return err
	}

	d.Partial(false)

	return resourceAwsElasticSearchDomainRead(d, meta)
}

func resourceAwsElasticSearchDomainDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).esconn
	domainName := d.Get("domain_name").(string)

	log.Printf("[DEBUG] Deleting ElasticSearch domain: %q", domainName)
	_, err := conn.DeleteElasticsearchDomain(&elasticsearch.DeleteElasticsearchDomainInput{
		DomainName: aws.String(domainName),
	})
	if err != nil {
		if isAWSErr(err, elasticsearch.ErrCodeResourceNotFoundException, "") {
			return nil
		}
		return err
	}

	log.Printf("[DEBUG] Waiting for ElasticSearch domain %q to be deleted", domainName)
	err = resourceAwsElasticSearchDomainDeleteWaiter(domainName, conn)

	return err
}

func resourceAwsElasticSearchDomainDeleteWaiter(domainName string, conn *elasticsearch.ElasticsearchService) error {
	input := &elasticsearch.DescribeElasticsearchDomainInput{
		DomainName: aws.String(domainName),
	}
	err := resource.Retry(90*time.Minute, func() *resource.RetryError {
		out, err := conn.DescribeElasticsearchDomain(input)

		if err != nil {
			if isAWSErr(err, elasticsearch.ErrCodeResourceNotFoundException, "") {
				return nil
			}
			return resource.NonRetryableError(err)
		}

		if out.DomainStatus != nil && !aws.BoolValue(out.DomainStatus.Processing) {
			return nil
		}

		return resource.RetryableError(fmt.Errorf("timeout while waiting for the domain %q to be deleted", domainName))
	})

	return err
}

func suppressEquivalentKmsKeyIds(k, old, new string, d *schema.ResourceData) bool {
	// The Elasticsearch API accepts a short KMS key id but always returns the ARN of the key.
	// The ARN is of the format 'arn:aws:kms:REGION:ACCOUNT_ID:key/KMS_KEY_ID'.
	// These should be treated as equivalent.
	return strings.Contains(old, new)
}

func getKibanaEndpoint(d *schema.ResourceData) string {
	return d.Get("endpoint").(string) + "/_plugin/kibana/"
}
