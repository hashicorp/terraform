package aws

import (
	"fmt"
	"log"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/guardduty"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsGuardDutyMember() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsGuardDutyMemberCreate,
		Read:   resourceAwsGuardDutyMemberRead,
		Delete: resourceAwsGuardDutyMemberDelete,

		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"account_id": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validateAwsAccountId,
			},
			"detector_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"email": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
		},
	}
}

func resourceAwsGuardDutyMemberCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).guarddutyconn
	accountID := d.Get("account_id").(string)
	detectorID := d.Get("detector_id").(string)

	input := guardduty.CreateMembersInput{
		AccountDetails: []*guardduty.AccountDetail{{
			AccountId: aws.String(accountID),
			Email:     aws.String(d.Get("email").(string)),
		}},
		DetectorId: aws.String(detectorID),
	}

	log.Printf("[DEBUG] Creating GuardDuty Member: %s", input)
	_, err := conn.CreateMembers(&input)
	if err != nil {
		return fmt.Errorf("Creating GuardDuty Member failed: %s", err.Error())
	}
	d.SetId(fmt.Sprintf("%s:%s", detectorID, accountID))

	return resourceAwsGuardDutyMemberRead(d, meta)
}

func resourceAwsGuardDutyMemberRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).guarddutyconn

	accountID, detectorID, err := decodeGuardDutyMemberID(d.Id())
	if err != nil {
		return err
	}

	input := guardduty.GetMembersInput{
		AccountIds: []*string{aws.String(accountID)},
		DetectorId: aws.String(detectorID),
	}

	log.Printf("[DEBUG] Reading GuardDuty Member: %s", input)
	gmo, err := conn.GetMembers(&input)
	if err != nil {
		if isAWSErr(err, guardduty.ErrCodeBadRequestException, "The request is rejected because the input detectorId is not owned by the current account.") {
			log.Printf("[WARN] GuardDuty detector %q not found, removing from state", d.Id())
			d.SetId("")
			return nil
		}
		return fmt.Errorf("Reading GuardDuty Member '%s' failed: %s", d.Id(), err.Error())
	}

	if gmo.Members == nil || (len(gmo.Members) < 1) {
		log.Printf("[WARN] GuardDuty Member %q not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}
	member := gmo.Members[0]
	d.Set("account_id", member.AccountId)
	d.Set("detector_id", detectorID)
	d.Set("email", member.Email)

	return nil
}

func resourceAwsGuardDutyMemberDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).guarddutyconn

	accountID, detectorID, err := decodeGuardDutyMemberID(d.Id())
	if err != nil {
		return err
	}

	input := guardduty.DeleteMembersInput{
		AccountIds: []*string{aws.String(accountID)},
		DetectorId: aws.String(detectorID),
	}

	log.Printf("[DEBUG] Delete GuardDuty Member: %s", input)
	_, err = conn.DeleteMembers(&input)
	if err != nil {
		return fmt.Errorf("Deleting GuardDuty Member '%s' failed: %s", d.Id(), err.Error())
	}
	return nil
}

func decodeGuardDutyMemberID(id string) (accountID, detectorID string, err error) {
	parts := strings.Split(id, ":")
	if len(parts) != 2 {
		err = fmt.Errorf("GuardDuty Member ID must be of the form <Detector ID>:<Member AWS Account ID>, was provided: %s", id)
		return
	}
	accountID = parts[1]
	detectorID = parts[0]
	return
}
