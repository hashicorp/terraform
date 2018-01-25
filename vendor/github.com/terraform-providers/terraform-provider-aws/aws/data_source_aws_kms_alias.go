package aws

import (
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/service/kms"
	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourceAwsKmsAlias() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceAwsKmsAliasRead,
		Schema: map[string]*schema.Schema{
			"name": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validateAwsKmsName,
			},
			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"target_key_arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"target_key_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func dataSourceAwsKmsAliasRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).kmsconn
	params := &kms.ListAliasesInput{}

	target := d.Get("name")
	var alias *kms.AliasListEntry
	log.Printf("[DEBUG] Reading KMS Alias: %s", params)
	err := conn.ListAliasesPages(params, func(page *kms.ListAliasesOutput, lastPage bool) bool {
		for _, entity := range page.Aliases {
			if *entity.AliasName == target {
				alias = entity
				return false
			}
		}

		return true
	})
	if err != nil {
		return errwrap.Wrapf("Error fetch KMS alias list: {{err}}", err)
	}

	if alias == nil {
		return fmt.Errorf("No alias with name %q found in this region.", target)
	}

	d.SetId(time.Now().UTC().String())
	d.Set("arn", alias.AliasArn)

	aliasARN, err := arn.Parse(*alias.AliasArn)
	if err != nil {
		return err
	}
	targetKeyARN := arn.ARN{
		Partition: aliasARN.Partition,
		Service:   aliasARN.Service,
		Region:    aliasARN.Region,
		AccountID: aliasARN.AccountID,
		Resource:  fmt.Sprintf("key/%s", *alias.TargetKeyId),
	}
	d.Set("target_key_arn", targetKeyARN.String())

	d.Set("target_key_id", alias.TargetKeyId)

	return nil
}
