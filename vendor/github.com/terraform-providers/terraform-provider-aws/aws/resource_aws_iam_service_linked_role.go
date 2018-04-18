package aws

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsIamServiceLinkedRole() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsIamServiceLinkedRoleCreate,
		Read:   resourceAwsIamServiceLinkedRoleRead,
		Update: resourceAwsIamServiceLinkedRoleUpdate,
		Delete: resourceAwsIamServiceLinkedRoleDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"aws_service_name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
				ValidateFunc: func(v interface{}, k string) (ws []string, es []error) {
					value := v.(string)
					if !strings.HasSuffix(value, ".amazonaws.com") {
						es = append(es, fmt.Errorf(
							"%q must be a service URL e.g. elasticbeanstalk.amazonaws.com", k))
					}
					return
				},
			},

			"name": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"path": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"create_date": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"unique_id": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"custom_suffix": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"description": {
				Type:     schema.TypeString,
				Optional: true,
			},
		},
	}
}

func resourceAwsIamServiceLinkedRoleCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).iamconn

	serviceName := d.Get("aws_service_name").(string)

	params := &iam.CreateServiceLinkedRoleInput{
		AWSServiceName: aws.String(serviceName),
	}

	if v, ok := d.GetOk("custom_suffix"); ok && v.(string) != "" {
		params.CustomSuffix = aws.String(v.(string))
	}

	if v, ok := d.GetOk("description"); ok && v.(string) != "" {
		params.Description = aws.String(v.(string))
	}

	resp, err := conn.CreateServiceLinkedRole(params)

	if err != nil {
		return fmt.Errorf("Error creating service-linked role with name %s: %s", serviceName, err)
	}
	d.SetId(*resp.Role.Arn)

	return resourceAwsIamServiceLinkedRoleRead(d, meta)
}

func resourceAwsIamServiceLinkedRoleRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).iamconn

	serviceName, roleName, customSuffix, err := decodeIamServiceLinkedRoleID(d.Id())
	if err != nil {
		return err
	}

	params := &iam.GetRoleInput{
		RoleName: aws.String(roleName),
	}

	resp, err := conn.GetRole(params)

	if err != nil {
		if isAWSErr(err, iam.ErrCodeNoSuchEntityException, "") {
			log.Printf("[WARN] IAM service linked role %s not found, removing from state", d.Id())
			d.SetId("")
			return nil
		}
		return err
	}

	role := resp.Role

	d.Set("arn", role.Arn)
	d.Set("aws_service_name", serviceName)
	d.Set("create_date", role.CreateDate)
	d.Set("custom_suffix", customSuffix)
	d.Set("description", role.Description)
	d.Set("name", role.RoleName)
	d.Set("path", role.Path)
	d.Set("unique_id", role.RoleId)

	return nil
}

func resourceAwsIamServiceLinkedRoleUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).iamconn

	_, roleName, _, err := decodeIamServiceLinkedRoleID(d.Id())
	if err != nil {
		return err
	}

	params := &iam.UpdateRoleInput{
		Description: aws.String(d.Get("description").(string)),
		RoleName:    aws.String(roleName),
	}

	_, err = conn.UpdateRole(params)

	if err != nil {
		return fmt.Errorf("Error updating service-linked role %s: %s", d.Id(), err)
	}

	return resourceAwsIamServiceLinkedRoleRead(d, meta)
}

func resourceAwsIamServiceLinkedRoleDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).iamconn

	_, roleName, _, err := decodeIamServiceLinkedRoleID(d.Id())
	if err != nil {
		return err
	}

	deletionID, err := deleteIamServiceLinkedRole(conn, roleName)
	if err != nil {
		return fmt.Errorf("Error deleting service-linked role %s: %s", d.Id(), err)
	}
	if deletionID == "" {
		return nil
	}

	err = deleteIamServiceLinkedRoleWaiter(conn, deletionID)
	if err != nil {
		return fmt.Errorf("Error waiting for role (%s) to be deleted: %s", d.Id(), err)
	}

	return nil
}

func decodeIamServiceLinkedRoleID(id string) (serviceName, roleName, customSuffix string, err error) {
	idArn, err := arn.Parse(id)
	if err != nil {
		return "", "", "", err
	}

	resourceParts := strings.Split(idArn.Resource, "/")
	if len(resourceParts) != 4 {
		return "", "", "", fmt.Errorf("expected IAM Service Role ARN (arn:PARTITION:iam::ACCOUNTID:role/aws-service-role/SERVICENAME/ROLENAME), received: %s", id)
	}

	serviceName = resourceParts[2]
	roleName = resourceParts[3]

	roleNameParts := strings.Split(roleName, "_")
	if len(roleNameParts) == 2 {
		customSuffix = roleNameParts[1]
	}

	return
}

func deleteIamServiceLinkedRole(conn *iam.IAM, roleName string) (string, error) {
	params := &iam.DeleteServiceLinkedRoleInput{
		RoleName: aws.String(roleName),
	}

	resp, err := conn.DeleteServiceLinkedRole(params)

	if err != nil {
		if isAWSErr(err, iam.ErrCodeNoSuchEntityException, "") {
			return "", nil
		}
		return "", err
	}

	return aws.StringValue(resp.DeletionTaskId), nil
}

func deleteIamServiceLinkedRoleWaiter(conn *iam.IAM, deletionTaskID string) error {
	stateConf := &resource.StateChangeConf{
		Pending: []string{iam.DeletionTaskStatusTypeInProgress, iam.DeletionTaskStatusTypeNotStarted},
		Target:  []string{iam.DeletionTaskStatusTypeSucceeded},
		Refresh: deleteIamServiceLinkedRoleRefreshFunc(conn, deletionTaskID),
		Timeout: 5 * time.Minute,
		Delay:   10 * time.Second,
	}

	_, err := stateConf.WaitForState()
	if err != nil {
		if isAWSErr(err, iam.ErrCodeNoSuchEntityException, "") {
			return nil
		}
		return err
	}

	return nil
}

func deleteIamServiceLinkedRoleRefreshFunc(conn *iam.IAM, deletionTaskId string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		params := &iam.GetServiceLinkedRoleDeletionStatusInput{
			DeletionTaskId: aws.String(deletionTaskId),
		}

		resp, err := conn.GetServiceLinkedRoleDeletionStatus(params)
		if err != nil {
			return nil, "", err
		}

		return resp, aws.StringValue(resp.Status), nil
	}
}
