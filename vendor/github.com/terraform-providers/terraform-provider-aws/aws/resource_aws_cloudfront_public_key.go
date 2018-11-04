package aws

import (
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudfront"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsCloudFrontPublicKey() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsCloudFrontPublicKeyCreate,
		Read:   resourceAwsCloudFrontPublicKeyRead,
		Update: resourceAwsCloudFrontPublicKeyUpdate,
		Delete: resourceAwsCloudFrontPublicKeyDelete,

		Schema: map[string]*schema.Schema{
			"caller_reference": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"comment": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"encoded_key": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"etag": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"name": {
				Type:          schema.TypeString,
				Optional:      true,
				Computed:      true,
				ForceNew:      true,
				ConflictsWith: []string{"name_prefix"},
				ValidateFunc:  validateCloudFrontPublicKeyName,
			},
			"name_prefix": {
				Type:          schema.TypeString,
				Optional:      true,
				Computed:      true,
				ForceNew:      true,
				ConflictsWith: []string{"name"},
				ValidateFunc:  validateCloudFrontPublicKeyNamePrefix,
			},
		},
	}
}

func resourceAwsCloudFrontPublicKeyCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).cloudfrontconn

	if v, ok := d.GetOk("name"); ok {
		d.Set("name", v.(string))
	} else if v, ok := d.GetOk("name_prefix"); ok {
		d.Set("name", resource.PrefixedUniqueId(v.(string)))
	} else {
		d.Set("name", resource.PrefixedUniqueId("tf-"))
	}

	request := &cloudfront.CreatePublicKeyInput{
		PublicKeyConfig: expandPublicKeyConfig(d),
	}

	log.Println("[DEBUG] Create CloudFront PublicKey:", request)

	output, err := conn.CreatePublicKey(request)
	if err != nil {
		return fmt.Errorf("error creating CloudFront PublicKey: %s", err)
	}

	d.SetId(aws.StringValue(output.PublicKey.Id))
	return resourceAwsCloudFrontPublicKeyRead(d, meta)
}

func resourceAwsCloudFrontPublicKeyRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).cloudfrontconn
	request := &cloudfront.GetPublicKeyInput{
		Id: aws.String(d.Id()),
	}

	output, err := conn.GetPublicKey(request)
	if err != nil {
		if isAWSErr(err, cloudfront.ErrCodeNoSuchPublicKey, "") {
			log.Printf("[WARN] No PublicKey found: %s, removing from state", d.Id())
			d.SetId("")
			return nil
		}
		return err
	}

	if output == nil || output.PublicKey == nil || output.PublicKey.PublicKeyConfig == nil {
		log.Printf("[WARN] No PublicKey found: %s, removing from state", d.Id())
		d.SetId("")
		return nil
	}
	publicKeyConfig := output.PublicKey.PublicKeyConfig

	d.Set("encoded_key", publicKeyConfig.EncodedKey)
	d.Set("name", publicKeyConfig.Name)
	d.Set("comment", publicKeyConfig.Comment)
	d.Set("caller_reference", publicKeyConfig.CallerReference)
	d.Set("etag", output.ETag)

	return nil
}

func resourceAwsCloudFrontPublicKeyUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).cloudfrontconn

	request := &cloudfront.UpdatePublicKeyInput{
		Id:              aws.String(d.Id()),
		PublicKeyConfig: expandPublicKeyConfig(d),
		IfMatch:         aws.String(d.Get("etag").(string)),
	}

	_, err := conn.UpdatePublicKey(request)
	if err != nil {
		return fmt.Errorf("error updating CloudFront PublicKey (%s): %s", d.Id(), err)
	}

	return resourceAwsCloudFrontPublicKeyRead(d, meta)
}

func resourceAwsCloudFrontPublicKeyDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).cloudfrontconn

	request := &cloudfront.DeletePublicKeyInput{
		Id:      aws.String(d.Id()),
		IfMatch: aws.String(d.Get("etag").(string)),
	}

	_, err := conn.DeletePublicKey(request)
	if err != nil {
		if isAWSErr(err, cloudfront.ErrCodeNoSuchPublicKey, "") {
			return nil
		}
		return err
	}

	return nil
}

func expandPublicKeyConfig(d *schema.ResourceData) *cloudfront.PublicKeyConfig {
	publicKeyConfig := &cloudfront.PublicKeyConfig{
		EncodedKey: aws.String(d.Get("encoded_key").(string)),
		Name:       aws.String(d.Get("name").(string)),
	}

	if v, ok := d.GetOk("comment"); ok {
		publicKeyConfig.Comment = aws.String(v.(string))
	}

	if v, ok := d.GetOk("caller_reference"); ok {
		publicKeyConfig.CallerReference = aws.String(v.(string))
	} else {
		publicKeyConfig.CallerReference = aws.String(time.Now().Format(time.RFC3339Nano))
	}

	return publicKeyConfig
}
