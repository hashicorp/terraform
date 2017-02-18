package aws

import (
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourceAwsCallerIdentity() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceAwsCallerIdentityRead,

		Schema: map[string]*schema.Schema{
			"account_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"user_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"user_name": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func dataSourceAwsCallerIdentityRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*AWSClient)

	log.Printf("[DEBUG] Reading Caller Identity.")
	d.SetId(time.Now().UTC().String())

	if client.accountid == "" {
		log.Println("[DEBUG] No Account ID available, failing")
		return fmt.Errorf("No AWS Account ID is available to the provider. Please ensure that\n" +
			"skip_requesting_account_id is not set on the AWS provider.")
	}

	user, err := client.iamconn.GetUser(&iam.GetUserInput{})
	if err != nil {
		return errwrap.Wrapf("Error retrieving current IAM user: {{err}}", err)
	}

	log.Printf("[DEBUG] Setting AWS Account ID to %s, user_id %s, user_name %s", client.accountid, *user.User.UserId, *user.User.UserName)
	d.Set("account_id", meta.(*AWSClient).accountid)
	d.Set("user_id", *user.User.UserId)
	d.Set("user_name", *user.User.UserName)
	return nil
}
