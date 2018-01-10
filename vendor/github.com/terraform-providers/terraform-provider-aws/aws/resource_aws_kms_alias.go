package aws

import (
	"fmt"
	"log"
	"regexp"
	"time"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/kms"
)

func resourceAwsKmsAlias() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsKmsAliasCreate,
		Read:   resourceAwsKmsAliasRead,
		Update: resourceAwsKmsAliasUpdate,
		Delete: resourceAwsKmsAliasDelete,

		Importer: &schema.ResourceImporter{
			State: resourceAwsKmsAliasImport,
		},

		Schema: map[string]*schema.Schema{
			"arn": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"name": &schema.Schema{
				Type:          schema.TypeString,
				Optional:      true,
				ForceNew:      true,
				ConflictsWith: []string{"name_prefix"},
				ValidateFunc:  validateAwsKmsName,
			},
			"name_prefix": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				ValidateFunc: func(v interface{}, k string) (ws []string, es []error) {
					value := v.(string)
					if !regexp.MustCompile(`^(alias\/)[a-zA-Z0-9:/_-]+$`).MatchString(value) {
						es = append(es, fmt.Errorf(
							"%q must begin with 'alias/' and be comprised of only [a-zA-Z0-9:/_-]", k))
					}
					return
				},
			},
			"target_key_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
		},
	}
}

func resourceAwsKmsAliasCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).kmsconn

	var name string
	if v, ok := d.GetOk("name"); ok {
		name = v.(string)
	} else if v, ok := d.GetOk("name_prefix"); ok {
		name = resource.PrefixedUniqueId(v.(string))
	} else {
		name = resource.PrefixedUniqueId("alias/")
	}

	targetKeyId := d.Get("target_key_id").(string)

	log.Printf("[DEBUG] KMS alias create name: %s, target_key: %s", name, targetKeyId)

	req := &kms.CreateAliasInput{
		AliasName:   aws.String(name),
		TargetKeyId: aws.String(targetKeyId),
	}

	// KMS is eventually consistent
	_, err := retryOnAwsCode("NotFoundException", func() (interface{}, error) {
		return conn.CreateAlias(req)
	})
	if err != nil {
		return err
	}
	d.SetId(name)
	return resourceAwsKmsAliasRead(d, meta)
}

func resourceAwsKmsAliasRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).kmsconn

	var alias *kms.AliasListEntry
	var err error
	if d.IsNewResource() {
		alias, err = retryFindKmsAliasByName(conn, d.Id())
	} else {
		alias, err = findKmsAliasByName(conn, d.Id(), nil)
	}
	if err != nil {
		return err
	}
	if alias == nil {
		log.Printf("[DEBUG] Removing KMS Alias (%s) as it's already gone", d.Id())
		d.SetId("")
		return nil
	}

	log.Printf("[DEBUG] Found KMS Alias: %s", alias)

	d.Set("arn", alias.AliasArn)
	d.Set("target_key_id", alias.TargetKeyId)

	return nil
}

func resourceAwsKmsAliasUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).kmsconn

	if d.HasChange("target_key_id") {
		err := resourceAwsKmsAliasTargetUpdate(conn, d)
		if err != nil {
			return err
		}
	}
	return nil
}

func resourceAwsKmsAliasTargetUpdate(conn *kms.KMS, d *schema.ResourceData) error {
	name := d.Get("name").(string)
	targetKeyId := d.Get("target_key_id").(string)

	log.Printf("[DEBUG] KMS alias: %s, update target: %s", name, targetKeyId)

	req := &kms.UpdateAliasInput{
		AliasName:   aws.String(name),
		TargetKeyId: aws.String(targetKeyId),
	}
	_, err := conn.UpdateAlias(req)

	return err
}

func resourceAwsKmsAliasDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).kmsconn

	req := &kms.DeleteAliasInput{
		AliasName: aws.String(d.Id()),
	}
	_, err := conn.DeleteAlias(req)
	if err != nil {
		return err
	}

	log.Printf("[DEBUG] KMS Alias: (%s) deleted.", d.Id())
	d.SetId("")
	return nil
}

func retryFindKmsAliasByName(conn *kms.KMS, name string) (*kms.AliasListEntry, error) {
	var resp *kms.AliasListEntry
	err := resource.Retry(1*time.Minute, func() *resource.RetryError {
		var err error
		resp, err = findKmsAliasByName(conn, name, nil)
		if err != nil {
			return resource.NonRetryableError(err)
		}
		if resp == nil {
			return resource.RetryableError(err)
		}
		return nil
	})
	return resp, err
}

// API by default limits results to 50 aliases
// This is how we make sure we won't miss any alias
// See http://docs.aws.amazon.com/kms/latest/APIReference/API_ListAliases.html
func findKmsAliasByName(conn *kms.KMS, name string, marker *string) (*kms.AliasListEntry, error) {
	req := kms.ListAliasesInput{
		Limit: aws.Int64(int64(100)),
	}
	if marker != nil {
		req.Marker = marker
	}

	log.Printf("[DEBUG] Listing KMS aliases: %s", req)
	resp, err := conn.ListAliases(&req)
	if err != nil {
		return nil, err
	}

	for _, entry := range resp.Aliases {
		if *entry.AliasName == name {
			return entry, nil
		}
	}
	if *resp.Truncated {
		log.Printf("[DEBUG] KMS alias list is truncated, listing more via %s", *resp.NextMarker)
		return findKmsAliasByName(conn, name, resp.NextMarker)
	}

	return nil, nil
}

func resourceAwsKmsAliasImport(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	d.Set("name", d.Id())
	return []*schema.ResourceData{d}, nil
}
