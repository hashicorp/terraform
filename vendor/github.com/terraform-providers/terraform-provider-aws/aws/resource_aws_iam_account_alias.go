package aws

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsIamAccountAlias() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsIamAccountAliasCreate,
		Read:   resourceAwsIamAccountAliasRead,
		Delete: resourceAwsIamAccountAliasDelete,

		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"account_alias": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validateAccountAlias,
			},
		},
	}
}

func resourceAwsIamAccountAliasCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).iamconn

	account_alias := d.Get("account_alias").(string)

	params := &iam.CreateAccountAliasInput{
		AccountAlias: aws.String(account_alias),
	}

	_, err := conn.CreateAccountAlias(params)

	if err != nil {
		return fmt.Errorf("Error creating account alias with name '%s': %s", account_alias, err)
	}

	d.SetId(account_alias)

	return nil
}

func resourceAwsIamAccountAliasRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).iamconn

	params := &iam.ListAccountAliasesInput{}

	resp, err := conn.ListAccountAliases(params)

	if err != nil {
		return fmt.Errorf("Error listing account aliases: %s", err)
	}

	if resp == nil || len(resp.AccountAliases) == 0 {
		d.SetId("")
		return nil
	}

	account_alias := aws.StringValue(resp.AccountAliases[0])

	d.SetId(account_alias)
	d.Set("account_alias", account_alias)

	return nil
}

func resourceAwsIamAccountAliasDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).iamconn

	account_alias := d.Get("account_alias").(string)

	params := &iam.DeleteAccountAliasInput{
		AccountAlias: aws.String(account_alias),
	}

	_, err := conn.DeleteAccountAlias(params)

	if err != nil {
		return fmt.Errorf("Error deleting account alias with name '%s': %s", account_alias, err)
	}

	d.SetId("")

	return nil
}
