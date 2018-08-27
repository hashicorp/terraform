package aws

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/macie"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsMacieMemberAccountAssociation() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsMacieMemberAccountAssociationCreate,
		Read:   resourceAwsMacieMemberAccountAssociationRead,
		Delete: resourceAwsMacieMemberAccountAssociationDelete,

		Schema: map[string]*schema.Schema{
			"member_account_id": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validateAwsAccountId,
			},
		},
	}
}

func resourceAwsMacieMemberAccountAssociationCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).macieconn

	memberAccountId := d.Get("member_account_id").(string)
	req := &macie.AssociateMemberAccountInput{
		MemberAccountId: aws.String(memberAccountId),
	}

	log.Printf("[DEBUG] Creating Macie member account association: %#v", req)
	_, err := conn.AssociateMemberAccount(req)
	if err != nil {
		return fmt.Errorf("Error creating Macie member account association: %s", err)
	}

	d.SetId(memberAccountId)
	return resourceAwsMacieMemberAccountAssociationRead(d, meta)
}

func resourceAwsMacieMemberAccountAssociationRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).macieconn

	req := &macie.ListMemberAccountsInput{}

	var res *macie.MemberAccount
	err := conn.ListMemberAccountsPages(req, func(page *macie.ListMemberAccountsOutput, lastPage bool) bool {
		for _, v := range page.MemberAccounts {
			if aws.StringValue(v.AccountId) == d.Get("member_account_id").(string) {
				res = v
				return false
			}
		}

		return true
	})
	if err != nil {
		return fmt.Errorf("Error listing Macie member account associations: %s", err)
	}

	if res == nil {
		log.Printf("[WARN] Macie member account association (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	return nil
}

func resourceAwsMacieMemberAccountAssociationDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).macieconn

	log.Printf("[DEBUG] Deleting Macie member account association: %s", d.Id())

	_, err := conn.DisassociateMemberAccount(&macie.DisassociateMemberAccountInput{
		MemberAccountId: aws.String(d.Get("member_account_id").(string)),
	})
	if err != nil {
		if isAWSErr(err, macie.ErrCodeInvalidInputException, "is a master Macie account and cannot be disassociated") {
			log.Printf("[INFO] Macie master account (%s) cannot be disassociated, removing from state", d.Id())
			return nil
		} else if isAWSErr(err, macie.ErrCodeInvalidInputException, "is not yet associated with Macie") {
			return nil
		} else {
			return fmt.Errorf("Error deleting Macie member account association: %s", err)
		}
	}

	return nil
}
