package aws

import (
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/kms"
	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourceAwsKmsKey() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceAwsKmsKeyRead,
		Schema: map[string]*schema.Schema{
			"key_id": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validateKmsKey,
			},
			"grant_tokens": {
				Type:     schema.TypeList,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"aws_account_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"creation_date": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"deletion_date": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"description": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"enabled": {
				Type:     schema.TypeBool,
				Computed: true,
			},
			"expiration_model": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"key_manager": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"key_state": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"key_usage": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"origin": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"valid_to": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func dataSourceAwsKmsKeyRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).kmsconn
	keyId := d.Get("key_id")
	var grantTokens []*string
	if v, ok := d.GetOk("grant_tokens"); ok {
		grantTokens = aws.StringSlice(v.([]string))
	}
	input := &kms.DescribeKeyInput{
		KeyId:       aws.String(keyId.(string)),
		GrantTokens: grantTokens,
	}
	output, err := conn.DescribeKey(input)
	if err != nil {
		return fmt.Errorf("error while describing key [%s]: %s", keyId, err)
	}
	d.SetId(aws.StringValue(output.KeyMetadata.KeyId))
	d.Set("arn", output.KeyMetadata.Arn)
	d.Set("aws_account_id", output.KeyMetadata.AWSAccountId)
	d.Set("creation_date", aws.TimeValue(output.KeyMetadata.CreationDate).Format(time.RFC3339))
	if output.KeyMetadata.DeletionDate != nil {
		d.Set("deletion_date", aws.TimeValue(output.KeyMetadata.DeletionDate).Format(time.RFC3339))
	}
	d.Set("description", output.KeyMetadata.Description)
	d.Set("enabled", output.KeyMetadata.Enabled)
	d.Set("expiration_model", output.KeyMetadata.ExpirationModel)
	d.Set("key_manager", output.KeyMetadata.KeyManager)
	d.Set("key_state", output.KeyMetadata.KeyState)
	d.Set("key_usage", output.KeyMetadata.KeyUsage)
	d.Set("origin", output.KeyMetadata.Origin)
	if output.KeyMetadata.ValidTo != nil {
		d.Set("valid_to", aws.TimeValue(output.KeyMetadata.ValidTo).Format(time.RFC3339))
	}
	return nil
}
