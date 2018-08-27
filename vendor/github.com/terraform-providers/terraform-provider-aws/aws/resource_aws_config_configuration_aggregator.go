package aws

import (
	"fmt"
	"log"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/configservice"

	"github.com/hashicorp/terraform/helper/customdiff"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
)

func resourceAwsConfigConfigurationAggregator() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsConfigConfigurationAggregatorPut,
		Read:   resourceAwsConfigConfigurationAggregatorRead,
		Update: resourceAwsConfigConfigurationAggregatorPut,
		Delete: resourceAwsConfigConfigurationAggregatorDelete,

		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		CustomizeDiff: customdiff.Sequence(
			// This is to prevent this error:
			// All fields are ForceNew or Computed w/out Optional, Update is superfluous
			customdiff.ForceNewIfChange("account_aggregation_source", func(old, new, meta interface{}) bool {
				return len(old.([]interface{})) == 0 && len(new.([]interface{})) > 0
			}),
			customdiff.ForceNewIfChange("organization_aggregation_source", func(old, new, meta interface{}) bool {
				return len(old.([]interface{})) == 0 && len(new.([]interface{})) > 0
			}),
		),

		Schema: map[string]*schema.Schema{
			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"name": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringLenBetween(0, 256),
			},
			"account_aggregation_source": {
				Type:          schema.TypeList,
				Optional:      true,
				MaxItems:      1,
				ConflictsWith: []string{"organization_aggregation_source"},
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"account_ids": {
							Type:     schema.TypeList,
							Required: true,
							MinItems: 1,
							Elem: &schema.Schema{
								Type:         schema.TypeString,
								ValidateFunc: validateAwsAccountId,
							},
						},
						"all_regions": {
							Type:     schema.TypeBool,
							Default:  false,
							Optional: true,
						},
						"regions": {
							Type:     schema.TypeList,
							Optional: true,
							MinItems: 1,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},
					},
				},
			},
			"organization_aggregation_source": {
				Type:          schema.TypeList,
				Optional:      true,
				MaxItems:      1,
				ConflictsWith: []string{"account_aggregation_source"},
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"all_regions": {
							Type:     schema.TypeBool,
							Default:  false,
							Optional: true,
						},
						"regions": {
							Type:     schema.TypeList,
							Optional: true,
							MinItems: 1,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},
						"role_arn": {
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: validateArn,
						},
					},
				},
			},
		},
	}
}

func resourceAwsConfigConfigurationAggregatorPut(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).configconn

	name := d.Get("name").(string)

	req := &configservice.PutConfigurationAggregatorInput{
		ConfigurationAggregatorName: aws.String(name),
	}

	account_aggregation_sources := d.Get("account_aggregation_source").([]interface{})
	if len(account_aggregation_sources) > 0 {
		req.AccountAggregationSources = expandConfigAccountAggregationSources(account_aggregation_sources)
	}

	organization_aggregation_sources := d.Get("organization_aggregation_source").([]interface{})
	if len(organization_aggregation_sources) > 0 {
		req.OrganizationAggregationSource = expandConfigOrganizationAggregationSource(organization_aggregation_sources[0].(map[string]interface{}))
	}

	_, err := conn.PutConfigurationAggregator(req)
	if err != nil {
		return fmt.Errorf("Error creating aggregator: %s", err)
	}

	d.SetId(strings.ToLower(name))

	return resourceAwsConfigConfigurationAggregatorRead(d, meta)
}

func resourceAwsConfigConfigurationAggregatorRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).configconn
	req := &configservice.DescribeConfigurationAggregatorsInput{
		ConfigurationAggregatorNames: []*string{aws.String(d.Id())},
	}

	res, err := conn.DescribeConfigurationAggregators(req)
	if err != nil {
		if isAWSErr(err, configservice.ErrCodeNoSuchConfigurationAggregatorException, "") {
			log.Printf("[WARN] No such configuration aggregator (%s), removing from state", d.Id())
			d.SetId("")
			return nil
		}
		return err
	}

	if res == nil || len(res.ConfigurationAggregators) == 0 {
		log.Printf("[WARN] No aggregators returned (%s), removing from state", d.Id())
		d.SetId("")
		return nil
	}

	aggregator := res.ConfigurationAggregators[0]
	d.Set("arn", aggregator.ConfigurationAggregatorArn)
	d.Set("name", aggregator.ConfigurationAggregatorName)

	if err := d.Set("account_aggregation_source", flattenConfigAccountAggregationSources(aggregator.AccountAggregationSources)); err != nil {
		return fmt.Errorf("error setting account_aggregation_source: %s", err)
	}

	if err := d.Set("organization_aggregation_source", flattenConfigOrganizationAggregationSource(aggregator.OrganizationAggregationSource)); err != nil {
		return fmt.Errorf("error setting organization_aggregation_source: %s", err)
	}

	return nil
}

func resourceAwsConfigConfigurationAggregatorDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).configconn

	req := &configservice.DeleteConfigurationAggregatorInput{
		ConfigurationAggregatorName: aws.String(d.Id()),
	}
	_, err := conn.DeleteConfigurationAggregator(req)
	if err != nil {
		return err
	}

	d.SetId("")
	return nil
}
