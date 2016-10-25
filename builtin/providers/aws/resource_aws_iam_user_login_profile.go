package aws

import (
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/vault/helper/pgpkeys"
)

func resourceAwsIamUserLoginProfile() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsIamUserLoginProfileCreate,
		Read:   schema.Noop,
		Update: schema.Noop,
		Delete: schema.RemoveFromState,

		Schema: map[string]*schema.Schema{
			"user": {
				Type:     schema.TypeString,
				Required: true,
			},
			"pgp_key": {
				Type:     schema.TypeString,
				Required: true,
			},
			"password_reset_required": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},
			"password_length": {
				Type:         schema.TypeInt,
				Optional:     true,
				Default:      20,
				ValidateFunc: validateAwsIamLoginProfilePasswordLength,
			},

			"key_fingerprint": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"encrypted_password": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func validateAwsIamLoginProfilePasswordLength(v interface{}, _ string) (_ []string, es []error) {
	length := v.(int)
	if length < 1 {
		es = append(es, errors.New("minimum password_length is 1 character"))
	}
	if length > 128 {
		es = append(es, errors.New("maximum password_length is 128 characters"))
	}
	return
}

func generatePassword(length int) string {
	const CharSetIAMPassword = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ012346789!@#$%^&*()_+-=[]{}|'"
	charSetLength := len(CharSetIAMPassword)

	rand.Seed(time.Now().UTC().UnixNano())
	result := make([]byte, length)
	for i := 0; i < length; i++ {
		result[i] = CharSetIAMPassword[rand.Intn(charSetLength)]
	}
	return string(result)
}

func encryptPassword(password string, pgpKey string) (string, string, error) {
	const keybasePrefix = "keybase:"

	encryptionKey := pgpKey
	if strings.HasPrefix(pgpKey, keybasePrefix) {
		publicKeys, err := pgpkeys.FetchKeybasePubkeys([]string{pgpKey})
		if err != nil {
			return "", "", errwrap.Wrapf(
				fmt.Sprintf("Error retrieving Public Key for %s: {{err}}", pgpKey), err)
		}
		encryptionKey = publicKeys[pgpKey]
	}

	fingerprints, encrypted, err := pgpkeys.EncryptShares([][]byte{[]byte(password)}, []string{encryptionKey})
	if err != nil {
		return "", "", errwrap.Wrapf(
			fmt.Sprintf("Error encrypting password for %s: {{err}}", pgpKey), err)
	}

	return fingerprints[0], base64.StdEncoding.EncodeToString(encrypted[0]), nil
}

func resourceAwsIamUserLoginProfileCreate(d *schema.ResourceData, meta interface{}) error {
	iamconn := meta.(*AWSClient).iamconn
	username := d.Get("user").(string)
	passwordResetRequired := d.Get("password_reset_required").(bool)
	passwordLength := d.Get("password_length").(int)

	var pgpKey string
	if pgpKeyInterface, ok := d.GetOk("pgp_key"); ok {
		pgpKey = pgpKeyInterface.(string)
	}

	initialPassword := generatePassword(passwordLength)
	fingerprint, encrypted, err := encryptPassword(initialPassword, pgpKey)
	if err != nil {
		return err
	}

	request := &iam.CreateLoginProfileInput{
		UserName:              aws.String(username),
		Password:              aws.String(initialPassword),
		PasswordResetRequired: aws.Bool(passwordResetRequired),
	}

	log.Println("[DEBUG] Create IAM User Login Profile request:", request)
	createResp, err := iamconn.CreateLoginProfile(request)
	if err != nil {
		return errwrap.Wrapf(fmt.Sprintf("Error creating IAM User Login Profile for %q: {{err}}", username), err)
	}

	d.SetId(*createResp.LoginProfile.UserName)
	d.Set("key_fingerprint", fingerprint)
	d.Set("encrypted_password", encrypted)
	return nil
}
