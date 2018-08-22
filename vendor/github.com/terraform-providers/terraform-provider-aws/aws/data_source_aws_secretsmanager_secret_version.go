package aws

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourceAwsSecretsManagerSecretVersion() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceAwsSecretsManagerSecretVersionRead,

		Schema: map[string]*schema.Schema{
			"secret_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"secret_string": {
				Type:      schema.TypeString,
				Computed:  true,
				Sensitive: true,
			},
			"version_id": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"version_stage": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "AWSCURRENT",
			},
			"version_stages": {
				Type:     schema.TypeSet,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
		},
	}
}

func dataSourceAwsSecretsManagerSecretVersionRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).secretsmanagerconn
	secretID := d.Get("secret_id").(string)
	var version string

	input := &secretsmanager.GetSecretValueInput{
		SecretId: aws.String(secretID),
	}

	if v, ok := d.GetOk("version_id"); ok {
		versionID := v.(string)
		input.VersionId = aws.String(versionID)
		version = versionID
	} else {
		versionStage := d.Get("version_stage").(string)
		input.VersionStage = aws.String(versionStage)
		version = versionStage
	}

	log.Printf("[DEBUG] Reading Secrets Manager Secret Version: %s", input)
	output, err := conn.GetSecretValue(input)
	if err != nil {
		if isAWSErr(err, secretsmanager.ErrCodeResourceNotFoundException, "") {
			return fmt.Errorf("Secrets Manager Secret %q Version %q not found", secretID, version)
		}
		if isAWSErr(err, secretsmanager.ErrCodeInvalidRequestException, "You can’t perform this operation on the secret because it was deleted") {
			return fmt.Errorf("Secrets Manager Secret %q Version %q not found", secretID, version)
		}
		return fmt.Errorf("error reading Secrets Manager Secret Version: %s", err)
	}

	d.SetId(fmt.Sprintf("%s|%s", secretID, version))
	d.Set("secret_id", secretID)
	d.Set("secret_string", output.SecretString)
	d.Set("version_id", output.VersionId)

	if err := d.Set("version_stages", flattenStringList(output.VersionStages)); err != nil {
		return fmt.Errorf("error setting version_stages: %s", err)
	}

	return nil
}
