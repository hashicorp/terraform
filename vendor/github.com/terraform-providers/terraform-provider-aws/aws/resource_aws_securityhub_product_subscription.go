package aws

import (
	"fmt"
	"log"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/securityhub"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsSecurityHubProductSubscription() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsSecurityHubProductSubscriptionCreate,
		Read:   resourceAwsSecurityHubProductSubscriptionRead,
		Delete: resourceAwsSecurityHubProductSubscriptionDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"product_arn": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validateArn,
			},
			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceAwsSecurityHubProductSubscriptionCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).securityhubconn
	productArn := d.Get("product_arn").(string)

	log.Printf("[DEBUG] Enabling Security Hub product subscription for product %s", productArn)

	resp, err := conn.EnableImportFindingsForProduct(&securityhub.EnableImportFindingsForProductInput{
		ProductArn: aws.String(productArn),
	})

	if err != nil {
		return fmt.Errorf("Error enabling Security Hub product subscription for product %s: %s", productArn, err)
	}

	d.SetId(fmt.Sprintf("%s,%s", productArn, *resp.ProductSubscriptionArn))

	return resourceAwsSecurityHubProductSubscriptionRead(d, meta)
}

func resourceAwsSecurityHubProductSubscriptionRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).securityhubconn

	productArn, productSubscriptionArn, err := resourceAwsSecurityHubProductSubscriptionParseId(d.Id())

	if err != nil {
		return err
	}

	log.Printf("[DEBUG] Reading Security Hub product subscriptions to find %s", d.Id())

	exists, err := resourceAwsSecurityHubProductSubscriptionCheckExists(conn, productSubscriptionArn)

	if err != nil {
		return fmt.Errorf("Error reading Security Hub product subscriptions to find %s: %s", d.Id(), err)
	}

	if !exists {
		log.Printf("[WARN] Security Hub product subscriptions (%s) not found, removing from state", d.Id())
		d.SetId("")
	}

	d.Set("product_arn", productArn)
	d.Set("arn", productSubscriptionArn)

	return nil
}

func resourceAwsSecurityHubProductSubscriptionCheckExists(conn *securityhub.SecurityHub, productSubscriptionArn string) (bool, error) {
	input := &securityhub.ListEnabledProductsForImportInput{}
	exists := false

	err := conn.ListEnabledProductsForImportPages(input, func(page *securityhub.ListEnabledProductsForImportOutput, lastPage bool) bool {
		for _, readProductSubscriptionArn := range page.ProductSubscriptions {
			if aws.StringValue(readProductSubscriptionArn) == productSubscriptionArn {
				exists = true
				return false
			}
		}
		return !lastPage
	})

	if err != nil {
		return false, err
	}

	return exists, nil
}

func resourceAwsSecurityHubProductSubscriptionParseId(id string) (string, string, error) {
	parts := strings.SplitN(id, ",", 2)

	if len(parts) != 2 {
		return "", "", fmt.Errorf("Expected Security Hub product subscription ID in format <product_arn>,<arn> - received: %s", id)
	}

	return parts[0], parts[1], nil
}

func resourceAwsSecurityHubProductSubscriptionDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).securityhubconn
	log.Printf("[DEBUG] Disabling Security Hub product subscription %s", d.Id())

	_, productSubscriptionArn, err := resourceAwsSecurityHubProductSubscriptionParseId(d.Id())

	if err != nil {
		return err
	}

	_, err = conn.DisableImportFindingsForProduct(&securityhub.DisableImportFindingsForProductInput{
		ProductSubscriptionArn: aws.String(productSubscriptionArn),
	})

	if err != nil {
		return fmt.Errorf("Error disabling Security Hub product subscription %s: %s", d.Id(), err)
	}

	return nil
}
