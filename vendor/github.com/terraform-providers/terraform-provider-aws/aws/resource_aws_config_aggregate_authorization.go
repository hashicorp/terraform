package aws

import (
	"fmt"
	"log"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/configservice"

	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsConfigAggregateAuthorization() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsConfigAggregateAuthorizationPut,
		Read:   resourceAwsConfigAggregateAuthorizationRead,
		Delete: resourceAwsConfigAggregateAuthorizationDelete,

		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"account_id": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validateAwsAccountId,
			},
			"region": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
		},
	}
}

func resourceAwsConfigAggregateAuthorizationPut(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).configconn

	accountId := d.Get("account_id").(string)
	region := d.Get("region").(string)

	req := &configservice.PutAggregationAuthorizationInput{
		AuthorizedAccountId: aws.String(accountId),
		AuthorizedAwsRegion: aws.String(region),
	}

	_, err := conn.PutAggregationAuthorization(req)
	if err != nil {
		return fmt.Errorf("Error creating aggregate authorization: %s", err)
	}

	d.SetId(fmt.Sprintf("%s:%s", accountId, region))
	return resourceAwsConfigAggregateAuthorizationRead(d, meta)
}

func resourceAwsConfigAggregateAuthorizationRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).configconn

	accountId, region, err := resourceAwsConfigAggregateAuthorizationParseID(d.Id())
	if err != nil {
		return err
	}

	d.Set("account_id", accountId)
	d.Set("region", region)

	aggregateAuthorizations, err := describeConfigAggregateAuthorizations(conn)
	if err != nil {
		return fmt.Errorf("Error retrieving list of aggregate authorizations: %s", err)
	}

	// Check for existing authorization
	for _, auth := range aggregateAuthorizations {
		if accountId == aws.StringValue(auth.AuthorizedAccountId) && region == aws.StringValue(auth.AuthorizedAwsRegion) {
			d.Set("arn", auth.AggregationAuthorizationArn)
			return nil
		}
	}

	log.Printf("[WARN] Aggregate Authorization not found, removing from state: %s", d.Id())
	d.SetId("")
	return nil
}

func resourceAwsConfigAggregateAuthorizationDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).configconn

	accountId, region, err := resourceAwsConfigAggregateAuthorizationParseID(d.Id())
	if err != nil {
		return err
	}

	req := &configservice.DeleteAggregationAuthorizationInput{
		AuthorizedAccountId: aws.String(accountId),
		AuthorizedAwsRegion: aws.String(region),
	}

	_, err = conn.DeleteAggregationAuthorization(req)
	if err != nil {
		return fmt.Errorf("Error deleting aggregate authorization: %s", err)
	}

	return nil
}

func describeConfigAggregateAuthorizations(conn *configservice.ConfigService) ([]*configservice.AggregationAuthorization, error) {
	aggregationAuthorizations := []*configservice.AggregationAuthorization{}
	input := &configservice.DescribeAggregationAuthorizationsInput{}

	for {
		output, err := conn.DescribeAggregationAuthorizations(input)
		if err != nil {
			return aggregationAuthorizations, err
		}
		aggregationAuthorizations = append(aggregationAuthorizations, output.AggregationAuthorizations...)
		if output.NextToken == nil {
			break
		}
		input.NextToken = output.NextToken
	}

	return aggregationAuthorizations, nil
}

func resourceAwsConfigAggregateAuthorizationParseID(id string) (string, string, error) {
	idParts := strings.Split(id, ":")
	if len(idParts) != 2 {
		return "", "", fmt.Errorf("Please make sure the ID is in the form account_id:region (i.e. 123456789012:us-east-1")
	}
	accountId := idParts[0]
	region := idParts[1]
	return accountId, region, nil
}
