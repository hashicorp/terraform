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

func resourceAwsStorageGatewayNfsFileShare() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsStorageGatewayNfsFileShareCreate,
		Read:   resourceAwsStorageGatewayNfsFileShareRead,
		Update: resourceAwsStorageGatewayNfsFileShareUpdate,
		Delete: resourceAwsStorageGatewayNfsFileShareDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},
		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(10 * time.Minute),
			Update: schema.DefaultTimeout(10 * time.Minute),
			Delete: schema.DefaultTimeout(10 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"client_list": {
				Type:     schema.TypeSet,
				Required: true,
				MinItems: 1,
				MaxItems: 100,
				Elem:     &schema.Schema{Type: schema.TypeString},
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
			"nfs_file_share_defaults": {
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"directory_mode": {
							Type:     schema.TypeString,
							Optional: true,
							Default:  "0777",
						},
						"file_mode": {
							Type:     schema.TypeString,
							Optional: true,
							Default:  "0666",
						},
						"group_id": {
							Type:         schema.TypeInt,
							Optional:     true,
							Default:      65534,
							ValidateFunc: validation.IntAtLeast(0),
						},
						"owner_id": {
							Type:         schema.TypeInt,
							Optional:     true,
							Default:      65534,
							ValidateFunc: validation.IntAtLeast(0),
						},
					},
				},
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
			"squash": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "RootSquash",
				ValidateFunc: validation.StringInSlice([]string{
					"AllSquash",
					"NoSquash",
					"RootSquash",
				}, false),
			},
		},
	}
}

func resourceAwsStorageGatewayNfsFileShareCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).storagegatewayconn

	input := &storagegateway.CreateNFSFileShareInput{
		ClientList:           expandStringSet(d.Get("client_list").(*schema.Set)),
		ClientToken:          aws.String(resource.UniqueId()),
		DefaultStorageClass:  aws.String(d.Get("default_storage_class").(string)),
		GatewayARN:           aws.String(d.Get("gateway_arn").(string)),
		GuessMIMETypeEnabled: aws.Bool(d.Get("guess_mime_type_enabled").(bool)),
		KMSEncrypted:         aws.Bool(d.Get("kms_encrypted").(bool)),
		LocationARN:          aws.String(d.Get("location_arn").(string)),
		NFSFileShareDefaults: expandStorageGatewayNfsFileShareDefaults(d.Get("nfs_file_share_defaults").([]interface{})),
		ObjectACL:            aws.String(d.Get("object_acl").(string)),
		ReadOnly:             aws.Bool(d.Get("read_only").(bool)),
		RequesterPays:        aws.Bool(d.Get("requester_pays").(bool)),
		Role:                 aws.String(d.Get("role_arn").(string)),
		Squash:               aws.String(d.Get("squash").(string)),
	}

	if v, ok := d.GetOk("kms_key_arn"); ok && v.(string) != "" {
		input.KMSKey = aws.String(v.(string))
	}

	log.Printf("[DEBUG] Creating Storage Gateway NFS File Share: %s", input)
	output, err := conn.CreateNFSFileShare(input)
	if err != nil {
		return fmt.Errorf("error creating Storage Gateway NFS File Share: %s", err)
	}

	d.SetId(aws.StringValue(output.FileShareARN))

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"CREATING", "MISSING"},
		Target:     []string{"AVAILABLE"},
		Refresh:    storageGatewayNfsFileShareRefreshFunc(d.Id(), conn),
		Timeout:    d.Timeout(schema.TimeoutCreate),
		Delay:      5 * time.Second,
		MinTimeout: 5 * time.Second,
	}
	_, err = stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf("error waiting for Storage Gateway NFS File Share creation: %s", err)
	}

	return resourceAwsStorageGatewayNfsFileShareRead(d, meta)
}

func resourceAwsStorageGatewayNfsFileShareRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).storagegatewayconn

	input := &storagegateway.DescribeNFSFileSharesInput{
		FileShareARNList: []*string{aws.String(d.Id())},
	}

	log.Printf("[DEBUG] Reading Storage Gateway NFS File Share: %s", input)
	output, err := conn.DescribeNFSFileShares(input)
	if err != nil {
		if isAWSErr(err, storagegateway.ErrCodeInvalidGatewayRequestException, "The specified file share was not found.") {
			log.Printf("[WARN] Storage Gateway NFS File Share %q not found, removing from state", d.Id())
			d.SetId("")
			return nil
		}
		return fmt.Errorf("error reading Storage Gateway NFS File Share: %s", err)
	}

	if output == nil || len(output.NFSFileShareInfoList) == 0 || output.NFSFileShareInfoList[0] == nil {
		log.Printf("[WARN] Storage Gateway NFS File Share %q not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	fileshare := output.NFSFileShareInfoList[0]

	d.Set("arn", fileshare.FileShareARN)

	if err := d.Set("client_list", schema.NewSet(schema.HashString, flattenStringList(fileshare.ClientList))); err != nil {
		return fmt.Errorf("error setting client_list: %s", err)
	}

	d.Set("default_storage_class", fileshare.DefaultStorageClass)
	d.Set("fileshare_id", fileshare.FileShareId)
	d.Set("gateway_arn", fileshare.GatewayARN)
	d.Set("guess_mime_type_enabled", fileshare.GuessMIMETypeEnabled)
	d.Set("kms_encrypted", fileshare.KMSEncrypted)
	d.Set("kms_key_arn", fileshare.KMSKey)
	d.Set("location_arn", fileshare.LocationARN)

	if err := d.Set("nfs_file_share_defaults", flattenStorageGatewayNfsFileShareDefaults(fileshare.NFSFileShareDefaults)); err != nil {
		return fmt.Errorf("error setting nfs_file_share_defaults: %s", err)
	}

	d.Set("object_acl", fileshare.ObjectACL)
	d.Set("path", fileshare.Path)
	d.Set("read_only", fileshare.ReadOnly)
	d.Set("requester_pays", fileshare.RequesterPays)
	d.Set("role_arn", fileshare.Role)
	d.Set("squash", fileshare.Squash)

	return nil
}

func resourceAwsStorageGatewayNfsFileShareUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).storagegatewayconn

	input := &storagegateway.UpdateNFSFileShareInput{
		ClientList:           expandStringSet(d.Get("client_list").(*schema.Set)),
		DefaultStorageClass:  aws.String(d.Get("default_storage_class").(string)),
		FileShareARN:         aws.String(d.Id()),
		GuessMIMETypeEnabled: aws.Bool(d.Get("guess_mime_type_enabled").(bool)),
		KMSEncrypted:         aws.Bool(d.Get("kms_encrypted").(bool)),
		NFSFileShareDefaults: expandStorageGatewayNfsFileShareDefaults(d.Get("nfs_file_share_defaults").([]interface{})),
		ObjectACL:            aws.String(d.Get("object_acl").(string)),
		ReadOnly:             aws.Bool(d.Get("read_only").(bool)),
		RequesterPays:        aws.Bool(d.Get("requester_pays").(bool)),
		Squash:               aws.String(d.Get("squash").(string)),
	}

	if v, ok := d.GetOk("kms_key_arn"); ok && v.(string) != "" {
		input.KMSKey = aws.String(v.(string))
	}

	log.Printf("[DEBUG] Updating Storage Gateway NFS File Share: %s", input)
	_, err := conn.UpdateNFSFileShare(input)
	if err != nil {
		return fmt.Errorf("error updating Storage Gateway NFS File Share: %s", err)
	}

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"UPDATING"},
		Target:     []string{"AVAILABLE"},
		Refresh:    storageGatewayNfsFileShareRefreshFunc(d.Id(), conn),
		Timeout:    d.Timeout(schema.TimeoutUpdate),
		Delay:      5 * time.Second,
		MinTimeout: 5 * time.Second,
	}
	_, err = stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf("error waiting for Storage Gateway NFS File Share update: %s", err)
	}

	return resourceAwsStorageGatewayNfsFileShareRead(d, meta)
}

func resourceAwsStorageGatewayNfsFileShareDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).storagegatewayconn

	input := &storagegateway.DeleteFileShareInput{
		FileShareARN: aws.String(d.Id()),
	}

	log.Printf("[DEBUG] Deleting Storage Gateway NFS File Share: %s", input)
	_, err := conn.DeleteFileShare(input)
	if err != nil {
		if isAWSErr(err, storagegateway.ErrCodeInvalidGatewayRequestException, "The specified file share was not found.") {
			return nil
		}
		return fmt.Errorf("error deleting Storage Gateway NFS File Share: %s", err)
	}

	stateConf := &resource.StateChangeConf{
		Pending:        []string{"AVAILABLE", "DELETING", "FORCE_DELETING"},
		Target:         []string{"MISSING"},
		Refresh:        storageGatewayNfsFileShareRefreshFunc(d.Id(), conn),
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
		return fmt.Errorf("error waiting for Storage Gateway NFS File Share deletion: %s", err)
	}

	return nil
}

func storageGatewayNfsFileShareRefreshFunc(fileShareArn string, conn *storagegateway.StorageGateway) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		input := &storagegateway.DescribeNFSFileSharesInput{
			FileShareARNList: []*string{aws.String(fileShareArn)},
		}

		log.Printf("[DEBUG] Reading Storage Gateway NFS File Share: %s", input)
		output, err := conn.DescribeNFSFileShares(input)
		if err != nil {
			if isAWSErr(err, storagegateway.ErrCodeInvalidGatewayRequestException, "The specified file share was not found.") {
				return nil, "MISSING", nil
			}
			return nil, "ERROR", fmt.Errorf("error reading Storage Gateway NFS File Share: %s", err)
		}

		if output == nil || len(output.NFSFileShareInfoList) == 0 || output.NFSFileShareInfoList[0] == nil {
			return nil, "MISSING", nil
		}

		fileshare := output.NFSFileShareInfoList[0]

		return fileshare, aws.StringValue(fileshare.FileShareStatus), nil
	}
}

func expandStorageGatewayNfsFileShareDefaults(l []interface{}) *storagegateway.NFSFileShareDefaults {
	if len(l) == 0 || l[0] == nil {
		return nil
	}

	m := l[0].(map[string]interface{})

	nfsFileShareDefaults := &storagegateway.NFSFileShareDefaults{
		DirectoryMode: aws.String(m["directory_mode"].(string)),
		FileMode:      aws.String(m["file_mode"].(string)),
		GroupId:       aws.Int64(int64(m["group_id"].(int))),
		OwnerId:       aws.Int64(int64(m["owner_id"].(int))),
	}

	return nfsFileShareDefaults
}

func flattenStorageGatewayNfsFileShareDefaults(nfsFileShareDefaults *storagegateway.NFSFileShareDefaults) []interface{} {
	if nfsFileShareDefaults == nil {
		return []interface{}{}
	}

	m := map[string]interface{}{
		"directory_mode": aws.StringValue(nfsFileShareDefaults.DirectoryMode),
		"file_mode":      aws.StringValue(nfsFileShareDefaults.FileMode),
		"group_id":       int(aws.Int64Value(nfsFileShareDefaults.GroupId)),
		"owner_id":       int(aws.Int64Value(nfsFileShareDefaults.OwnerId)),
	}

	return []interface{}{m}
}
