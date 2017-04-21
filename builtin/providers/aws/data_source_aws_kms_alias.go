package aws

import (
	"fmt"
	"time"

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
	d.Set("target_key_id", alias.TargetKeyId)

	return nil
}
