package aws

import (
	"fmt"
	"log"
	"strings"

	"encoding/base64"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsSecretsManagerSecretVersion() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsSecretsManagerSecretVersionCreate,
		Read:   resourceAwsSecretsManagerSecretVersionRead,
		Update: resourceAwsSecretsManagerSecretVersionUpdate,
		Delete: resourceAwsSecretsManagerSecretVersionDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"secret_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"secret_string": {
				Type:          schema.TypeString,
				Optional:      true,
				ForceNew:      true,
				Sensitive:     true,
				ConflictsWith: []string{"secret_binary"},
			},
			"secret_binary": {
				Type:          schema.TypeString,
				Optional:      true,
				ForceNew:      true,
				Sensitive:     true,
				ConflictsWith: []string{"secret_string"},
			},
			"version_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"version_stages": {
				Type:     schema.TypeSet,
				Optional: true,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
		},
	}
}

func resourceAwsSecretsManagerSecretVersionCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).secretsmanagerconn
	secretID := d.Get("secret_id").(string)

	input := &secretsmanager.PutSecretValueInput{
		SecretId: aws.String(secretID),
	}

	if v, ok := d.GetOk("secret_string"); ok {
		input.SecretString = aws.String(v.(string))
	}

	if v, ok := d.GetOk("secret_binary"); ok {
		vs := []byte(v.(string))

		if !isBase64Encoded(vs) {
			return fmt.Errorf("expected base64 in secret_binary")
		}

		var err error
		input.SecretBinary, err = base64.StdEncoding.DecodeString(v.(string))

		if err != nil {
			return fmt.Errorf("error decoding secret binary value: %s", err)
		}
	}

	if v, ok := d.GetOk("version_stages"); ok {
		input.VersionStages = expandStringList(v.(*schema.Set).List())
	}

	log.Printf("[DEBUG] Putting Secrets Manager Secret %q value", secretID)
	output, err := conn.PutSecretValue(input)
	if err != nil {
		return fmt.Errorf("error putting Secrets Manager Secret value: %s", err)
	}

	d.SetId(fmt.Sprintf("%s|%s", secretID, aws.StringValue(output.VersionId)))

	return resourceAwsSecretsManagerSecretVersionRead(d, meta)
}

func resourceAwsSecretsManagerSecretVersionRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).secretsmanagerconn

	secretID, versionID, err := decodeSecretsManagerSecretVersionID(d.Id())
	if err != nil {
		return err
	}

	input := &secretsmanager.GetSecretValueInput{
		SecretId:  aws.String(secretID),
		VersionId: aws.String(versionID),
	}

	log.Printf("[DEBUG] Reading Secrets Manager Secret Version: %s", input)
	output, err := conn.GetSecretValue(input)
	if err != nil {
		if isAWSErr(err, secretsmanager.ErrCodeResourceNotFoundException, "") {
			log.Printf("[WARN] Secrets Manager Secret Version %q not found - removing from state", d.Id())
			d.SetId("")
			return nil
		}
		if isAWSErr(err, secretsmanager.ErrCodeInvalidRequestException, "You can’t perform this operation on the secret because it was deleted") {
			log.Printf("[WARN] Secrets Manager Secret Version %q not found - removing from state", d.Id())
			d.SetId("")
			return nil
		}
		return fmt.Errorf("error reading Secrets Manager Secret Version: %s", err)
	}

	d.Set("secret_id", secretID)
	d.Set("secret_string", output.SecretString)
	d.Set("secret_binary", base64Encode(output.SecretBinary))
	d.Set("version_id", output.VersionId)
	d.Set("arn", output.ARN)

	if err := d.Set("version_stages", flattenStringList(output.VersionStages)); err != nil {
		return fmt.Errorf("error setting version_stages: %s", err)
	}

	return nil
}

func resourceAwsSecretsManagerSecretVersionUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).secretsmanagerconn

	secretID, versionID, err := decodeSecretsManagerSecretVersionID(d.Id())
	if err != nil {
		return err
	}

	o, n := d.GetChange("version_stages")
	os := o.(*schema.Set)
	ns := n.(*schema.Set)
	stagesToAdd := ns.Difference(os).List()
	stagesToRemove := os.Difference(ns).List()

	for _, stage := range stagesToAdd {
		input := &secretsmanager.UpdateSecretVersionStageInput{
			MoveToVersionId: aws.String(versionID),
			SecretId:        aws.String(secretID),
			VersionStage:    aws.String(stage.(string)),
		}

		log.Printf("[DEBUG] Updating Secrets Manager Secret Version Stage: %s", input)
		_, err := conn.UpdateSecretVersionStage(input)
		if err != nil {
			return fmt.Errorf("error updating Secrets Manager Secret %q Version Stage %q: %s", secretID, stage.(string), err)
		}
	}

	for _, stage := range stagesToRemove {
		// InvalidParameterException: You can only move staging label AWSCURRENT to a different secret version. It can’t be completely removed.
		if stage.(string) == "AWSCURRENT" {
			log.Printf("[INFO] Skipping removal of AWSCURRENT staging label for secret %q version %q", secretID, versionID)
			continue
		}
		input := &secretsmanager.UpdateSecretVersionStageInput{
			RemoveFromVersionId: aws.String(versionID),
			SecretId:            aws.String(secretID),
			VersionStage:        aws.String(stage.(string)),
		}
		log.Printf("[DEBUG] Updating Secrets Manager Secret Version Stage: %s", input)
		_, err := conn.UpdateSecretVersionStage(input)
		if err != nil {
			return fmt.Errorf("error updating Secrets Manager Secret %q Version Stage %q: %s", secretID, stage.(string), err)
		}
	}

	return resourceAwsSecretsManagerSecretVersionRead(d, meta)
}

func resourceAwsSecretsManagerSecretVersionDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).secretsmanagerconn

	secretID, versionID, err := decodeSecretsManagerSecretVersionID(d.Id())
	if err != nil {
		return err
	}

	if v, ok := d.GetOk("version_stages"); ok {
		for _, stage := range v.(*schema.Set).List() {
			// InvalidParameterException: You can only move staging label AWSCURRENT to a different secret version. It can’t be completely removed.
			if stage.(string) == "AWSCURRENT" {
				log.Printf("[WARN] Cannot remove AWSCURRENT staging label, which may leave the secret %q version %q active", secretID, versionID)
				continue
			}
			input := &secretsmanager.UpdateSecretVersionStageInput{
				RemoveFromVersionId: aws.String(versionID),
				SecretId:            aws.String(secretID),
				VersionStage:        aws.String(stage.(string)),
			}
			log.Printf("[DEBUG] Updating Secrets Manager Secret Version Stage: %s", input)
			_, err := conn.UpdateSecretVersionStage(input)
			if err != nil {
				if isAWSErr(err, secretsmanager.ErrCodeResourceNotFoundException, "") {
					return nil
				}
				if isAWSErr(err, secretsmanager.ErrCodeInvalidRequestException, "You can’t perform this operation on the secret because it was deleted") {
					return nil
				}
				return fmt.Errorf("error updating Secrets Manager Secret %q Version Stage %q: %s", secretID, stage.(string), err)
			}
		}
	}

	return nil
}

func decodeSecretsManagerSecretVersionID(id string) (string, string, error) {
	idParts := strings.Split(id, "|")
	if len(idParts) != 2 {
		return "", "", fmt.Errorf("expected ID in format SecretID|VersionID, received: %s", id)
	}
	return idParts[0], idParts[1], nil
}
