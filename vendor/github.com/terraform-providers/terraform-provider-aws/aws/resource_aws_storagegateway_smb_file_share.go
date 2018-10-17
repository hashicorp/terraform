package aws

import (
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/storagegateway"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
)

func resourceAwsStorageGatewaySmbFileShare() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsStorageGatewaySmbFileShareCreate,
		Read:   resourceAwsStorageGatewaySmbFileShareRead,
		Update: resourceAwsStorageGatewaySmbFileShareUpdate,
		Delete: resourceAwsStorageGatewaySmbFileShareDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},
		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(10 * time.Minute),
			Update: schema.DefaultTimeout(10 * time.Minute),
			Delete: schema.DefaultTimeout(15 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"authentication": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Default:  "ActiveDirectory",
				ValidateFunc: validation.StringInSlice([]string{
					"ActiveDirectory",
					"GuestAccess",
				}, false),
			},
			"default_storage_class": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "S3_STANDARD",
				ValidateFunc: validation.StringInSlice([]string{
					"S3_ONEZONE_IA",
					"S3_STANDARD_IA",
					"S3_STANDARD",
				}, false),
			},
			"fileshare_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"gateway_arn": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validateArn,
			},
			"guess_mime_type_enabled": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},
			"invalid_user_list": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"kms_encrypted": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"kms_key_arn": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validateArn,
			},
			"location_arn": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validateArn,
			},
			"object_acl": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  storagegateway.ObjectACLPrivate,
				ValidateFunc: validation.StringInSlice([]string{
					storagegateway.ObjectACLAuthenticatedRead,
					storagegateway.ObjectACLAwsExecRead,
					storagegateway.ObjectACLBucketOwnerFullControl,
					storagegateway.ObjectACLBucketOwnerRead,
					storagegateway.ObjectACLPrivate,
					storagegateway.ObjectACLPublicRead,
					storagegateway.ObjectACLPublicReadWrite,
				}, false),
			},
			"read_only": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"requester_pays": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"role_arn": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validateArn,
			},
			"valid_user_list": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
		},
	}
}

func resourceAwsStorageGatewaySmbFileShareCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).storagegatewayconn

	input := &storagegateway.CreateSMBFileShareInput{
		Authentication:       aws.String(d.Get("authentication").(string)),
		ClientToken:          aws.String(resource.UniqueId()),
		DefaultStorageClass:  aws.String(d.Get("default_storage_class").(string)),
		GatewayARN:           aws.String(d.Get("gateway_arn").(string)),
		GuessMIMETypeEnabled: aws.Bool(d.Get("guess_mime_type_enabled").(bool)),
		InvalidUserList:      expandStringSet(d.Get("invalid_user_list").(*schema.Set)),
		KMSEncrypted:         aws.Bool(d.Get("kms_encrypted").(bool)),
		LocationARN:          aws.String(d.Get("location_arn").(string)),
		ObjectACL:            aws.String(d.Get("object_acl").(string)),
		ReadOnly:             aws.Bool(d.Get("read_only").(bool)),
		RequesterPays:        aws.Bool(d.Get("requester_pays").(bool)),
		Role:                 aws.String(d.Get("role_arn").(string)),
		ValidUserList:        expandStringSet(d.Get("valid_user_list").(*schema.Set)),
	}

	if v, ok := d.GetOk("kms_key_arn"); ok && v.(string) != "" {
		input.KMSKey = aws.String(v.(string))
	}

	log.Printf("[DEBUG] Creating Storage Gateway SMB File Share: %s", input)
	output, err := conn.CreateSMBFileShare(input)
	if err != nil {
		return fmt.Errorf("error creating Storage Gateway SMB File Share: %s", err)
	}

	d.SetId(aws.StringValue(output.FileShareARN))

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"CREATING", "MISSING"},
		Target:     []string{"AVAILABLE"},
		Refresh:    storageGatewaySmbFileShareRefreshFunc(d.Id(), conn),
		Timeout:    d.Timeout(schema.TimeoutCreate),
		Delay:      5 * time.Second,
		MinTimeout: 5 * time.Second,
	}
	_, err = stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf("error waiting for Storage Gateway SMB File Share creation: %s", err)
	}

	return resourceAwsStorageGatewaySmbFileShareRead(d, meta)
}

func resourceAwsStorageGatewaySmbFileShareRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).storagegatewayconn

	input := &storagegateway.DescribeSMBFileSharesInput{
		FileShareARNList: []*string{aws.String(d.Id())},
	}

	log.Printf("[DEBUG] Reading Storage Gateway SMB File Share: %s", input)
	output, err := conn.DescribeSMBFileShares(input)
	if err != nil {
		if isAWSErr(err, storagegateway.ErrCodeInvalidGatewayRequestException, "The specified file share was not found.") {
			log.Printf("[WARN] Storage Gateway SMB File Share %q not found, removing from state", d.Id())
			d.SetId("")
			return nil
		}
		return fmt.Errorf("error reading Storage Gateway SMB File Share: %s", err)
	}

	if output == nil || len(output.SMBFileShareInfoList) == 0 || output.SMBFileShareInfoList[0] == nil {
		log.Printf("[WARN] Storage Gateway SMB File Share %q not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	fileshare := output.SMBFileShareInfoList[0]

	d.Set("arn", fileshare.FileShareARN)
	d.Set("authentication", fileshare.Authentication)
	d.Set("default_storage_class", fileshare.DefaultStorageClass)
	d.Set("fileshare_id", fileshare.FileShareId)
	d.Set("gateway_arn", fileshare.GatewayARN)
	d.Set("guess_mime_type_enabled", fileshare.GuessMIMETypeEnabled)

	if err := d.Set("invalid_user_list", schema.NewSet(schema.HashString, flattenStringList(fileshare.InvalidUserList))); err != nil {
		return fmt.Errorf("error setting invalid_user_list: %s", err)
	}

	d.Set("kms_encrypted", fileshare.KMSEncrypted)
	d.Set("kms_key_arn", fileshare.KMSKey)
	d.Set("location_arn", fileshare.LocationARN)
	d.Set("object_acl", fileshare.ObjectACL)
	d.Set("path", fileshare.Path)
	d.Set("read_only", fileshare.ReadOnly)
	d.Set("requester_pays", fileshare.RequesterPays)
	d.Set("role_arn", fileshare.Role)

	if err := d.Set("valid_user_list", schema.NewSet(schema.HashString, flattenStringList(fileshare.ValidUserList))); err != nil {
		return fmt.Errorf("error setting valid_user_list: %s", err)
	}

	return nil
}

func resourceAwsStorageGatewaySmbFileShareUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).storagegatewayconn

	input := &storagegateway.UpdateSMBFileShareInput{
		DefaultStorageClass:  aws.String(d.Get("default_storage_class").(string)),
		FileShareARN:         aws.String(d.Id()),
		GuessMIMETypeEnabled: aws.Bool(d.Get("guess_mime_type_enabled").(bool)),
		InvalidUserList:      expandStringSet(d.Get("invalid_user_list").(*schema.Set)),
		KMSEncrypted:         aws.Bool(d.Get("kms_encrypted").(bool)),
		ObjectACL:            aws.String(d.Get("object_acl").(string)),
		ReadOnly:             aws.Bool(d.Get("read_only").(bool)),
		RequesterPays:        aws.Bool(d.Get("requester_pays").(bool)),
		ValidUserList:        expandStringSet(d.Get("valid_user_list").(*schema.Set)),
	}

	if v, ok := d.GetOk("kms_key_arn"); ok && v.(string) != "" {
		input.KMSKey = aws.String(v.(string))
	}

	log.Printf("[DEBUG] Updating Storage Gateway SMB File Share: %s", input)
	_, err := conn.UpdateSMBFileShare(input)
	if err != nil {
		return fmt.Errorf("error updating Storage Gateway SMB File Share: %s", err)
	}

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"UPDATING"},
		Target:     []string{"AVAILABLE"},
		Refresh:    storageGatewaySmbFileShareRefreshFunc(d.Id(), conn),
		Timeout:    d.Timeout(schema.TimeoutUpdate),
		Delay:      5 * time.Second,
		MinTimeout: 5 * time.Second,
	}
	_, err = stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf("error waiting for Storage Gateway SMB File Share update: %s", err)
	}

	return resourceAwsStorageGatewaySmbFileShareRead(d, meta)
}

func resourceAwsStorageGatewaySmbFileShareDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).storagegatewayconn

	input := &storagegateway.DeleteFileShareInput{
		FileShareARN: aws.String(d.Id()),
	}

	log.Printf("[DEBUG] Deleting Storage Gateway SMB File Share: %s", input)
	_, err := conn.DeleteFileShare(input)
	if err != nil {
		if isAWSErr(err, storagegateway.ErrCodeInvalidGatewayRequestException, "The specified file share was not found.") {
			return nil
		}
		return fmt.Errorf("error deleting Storage Gateway SMB File Share: %s", err)
	}

	stateConf := &resource.StateChangeConf{
		Pending:        []string{"AVAILABLE", "DELETING", "FORCE_DELETING"},
		Target:         []string{"MISSING"},
		Refresh:        storageGatewaySmbFileShareRefreshFunc(d.Id(), conn),
		Timeout:        d.Timeout(schema.TimeoutDelete),
		Delay:          5 * time.Second,
		MinTimeout:     5 * time.Second,
		NotFoundChecks: 1,
	}
	_, err = stateConf.WaitForState()
	if err != nil {
		if isResourceNotFoundError(err) {
			return nil
		}
		return fmt.Errorf("error waiting for Storage Gateway SMB File Share deletion: %s", err)
	}

	return nil
}

func storageGatewaySmbFileShareRefreshFunc(fileShareArn string, conn *storagegateway.StorageGateway) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		input := &storagegateway.DescribeSMBFileSharesInput{
			FileShareARNList: []*string{aws.String(fileShareArn)},
		}

		log.Printf("[DEBUG] Reading Storage Gateway SMB File Share: %s", input)
		output, err := conn.DescribeSMBFileShares(input)
		if err != nil {
			if isAWSErr(err, storagegateway.ErrCodeInvalidGatewayRequestException, "The specified file share was not found.") {
				return nil, "MISSING", nil
			}
			return nil, "ERROR", fmt.Errorf("error reading Storage Gateway SMB File Share: %s", err)
		}

		if output == nil || len(output.SMBFileShareInfoList) == 0 || output.SMBFileShareInfoList[0] == nil {
			return nil, "MISSING", nil
		}

		fileshare := output.SMBFileShareInfoList[0]

		return fileshare, aws.StringValue(fileshare.FileShareStatus), nil
	}
}
