package aws

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/guardduty"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
)

func resourceAwsGuardDutyThreatintelset() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsGuardDutyThreatintelsetCreate,
		Read:   resourceAwsGuardDutyThreatintelsetRead,
		Update: resourceAwsGuardDutyThreatintelsetUpdate,
		Delete: resourceAwsGuardDutyThreatintelsetDelete,

		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"detector_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"format": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
				ValidateFunc: validation.StringInSlice([]string{
					guardduty.ThreatIntelSetFormatTxt,
					guardduty.ThreatIntelSetFormatStix,
					guardduty.ThreatIntelSetFormatOtxCsv,
					guardduty.ThreatIntelSetFormatAlienVault,
					guardduty.ThreatIntelSetFormatProofPoint,
					guardduty.ThreatIntelSetFormatFireEye,
				}, false),
			},
			"location": {
				Type:     schema.TypeString,
				Required: true,
			},
			"activate": {
				Type:     schema.TypeBool,
				Required: true,
			},
		},
	}
}

func resourceAwsGuardDutyThreatintelsetCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).guarddutyconn

	detectorID := d.Get("detector_id").(string)
	input := &guardduty.CreateThreatIntelSetInput{
		DetectorId: aws.String(detectorID),
		Name:       aws.String(d.Get("name").(string)),
		Format:     aws.String(d.Get("format").(string)),
		Location:   aws.String(d.Get("location").(string)),
		Activate:   aws.Bool(d.Get("activate").(bool)),
	}

	resp, err := conn.CreateThreatIntelSet(input)
	if err != nil {
		return err
	}

	stateConf := &resource.StateChangeConf{
		Pending:    []string{guardduty.ThreatIntelSetStatusActivating, guardduty.ThreatIntelSetStatusDeactivating},
		Target:     []string{guardduty.ThreatIntelSetStatusActive, guardduty.ThreatIntelSetStatusInactive},
		Refresh:    guardDutyThreatintelsetRefreshStatusFunc(conn, *resp.ThreatIntelSetId, detectorID),
		Timeout:    5 * time.Minute,
		MinTimeout: 3 * time.Second,
	}

	_, err = stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf("Error waiting for GuardDuty ThreatIntelSet status to be \"%s\" or \"%s\": %s",
			guardduty.ThreatIntelSetStatusActive, guardduty.ThreatIntelSetStatusInactive, err)
	}

	d.SetId(fmt.Sprintf("%s:%s", detectorID, *resp.ThreatIntelSetId))
	return resourceAwsGuardDutyThreatintelsetRead(d, meta)
}

func resourceAwsGuardDutyThreatintelsetRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).guarddutyconn

	threatIntelSetId, detectorId, err := decodeGuardDutyThreatintelsetID(d.Id())
	if err != nil {
		return err
	}
	input := &guardduty.GetThreatIntelSetInput{
		DetectorId:       aws.String(detectorId),
		ThreatIntelSetId: aws.String(threatIntelSetId),
	}

	resp, err := conn.GetThreatIntelSet(input)
	if err != nil {
		if isAWSErr(err, guardduty.ErrCodeBadRequestException, "The request is rejected because the input detectorId is not owned by the current account.") {
			log.Printf("[WARN] GuardDuty ThreatIntelSet %q not found, removing from state", threatIntelSetId)
			d.SetId("")
			return nil
		}
		return err
	}

	d.Set("detector_id", detectorId)
	d.Set("format", resp.Format)
	d.Set("location", resp.Location)
	d.Set("name", resp.Name)
	d.Set("activate", *resp.Status == guardduty.ThreatIntelSetStatusActive)
	return nil
}

func resourceAwsGuardDutyThreatintelsetUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).guarddutyconn

	threatIntelSetID, detectorId, err := decodeGuardDutyThreatintelsetID(d.Id())
	if err != nil {
		return err
	}
	input := &guardduty.UpdateThreatIntelSetInput{
		DetectorId:       aws.String(detectorId),
		ThreatIntelSetId: aws.String(threatIntelSetID),
	}

	if d.HasChange("name") {
		input.Name = aws.String(d.Get("name").(string))
	}
	if d.HasChange("location") {
		input.Location = aws.String(d.Get("location").(string))
	}
	if d.HasChange("activate") {
		input.Activate = aws.Bool(d.Get("activate").(bool))
	}

	_, err = conn.UpdateThreatIntelSet(input)
	if err != nil {
		return err
	}

	return resourceAwsGuardDutyThreatintelsetRead(d, meta)
}

func resourceAwsGuardDutyThreatintelsetDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).guarddutyconn

	threatIntelSetID, detectorId, err := decodeGuardDutyThreatintelsetID(d.Id())
	if err != nil {
		return err
	}
	input := &guardduty.DeleteThreatIntelSetInput{
		DetectorId:       aws.String(detectorId),
		ThreatIntelSetId: aws.String(threatIntelSetID),
	}

	_, err = conn.DeleteThreatIntelSet(input)
	if err != nil {
		return err
	}

	stateConf := &resource.StateChangeConf{
		Pending: []string{
			guardduty.ThreatIntelSetStatusActive,
			guardduty.ThreatIntelSetStatusActivating,
			guardduty.ThreatIntelSetStatusInactive,
			guardduty.ThreatIntelSetStatusDeactivating,
			guardduty.ThreatIntelSetStatusDeletePending,
		},
		Target:     []string{guardduty.ThreatIntelSetStatusDeleted},
		Refresh:    guardDutyThreatintelsetRefreshStatusFunc(conn, threatIntelSetID, detectorId),
		Timeout:    5 * time.Minute,
		MinTimeout: 3 * time.Second,
	}

	_, err = stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf("Error waiting for GuardDuty ThreatIntelSet status to be \"%s\": %s", guardduty.ThreatIntelSetStatusDeleted, err)
	}

	return nil
}

func guardDutyThreatintelsetRefreshStatusFunc(conn *guardduty.GuardDuty, threatIntelSetID, detectorID string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		input := &guardduty.GetThreatIntelSetInput{
			DetectorId:       aws.String(detectorID),
			ThreatIntelSetId: aws.String(threatIntelSetID),
		}
		resp, err := conn.GetThreatIntelSet(input)
		if err != nil {
			return nil, "failed", err
		}
		return resp, *resp.Status, nil
	}
}

func decodeGuardDutyThreatintelsetID(id string) (threatIntelSetID, detectorID string, err error) {
	parts := strings.Split(id, ":")
	if len(parts) != 2 {
		err = fmt.Errorf("GuardDuty ThreatIntelSet ID must be of the form <Detector ID>:<ThreatIntelSet ID>, was provided: %s", id)
		return
	}
	threatIntelSetID = parts[1]
	detectorID = parts[0]
	return
}
