package aws

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"log"
	"math/big"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/hashicorp/terraform/helper/encryption"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
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
				ValidateFunc: validation.IntBetween(5, 128),
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

const (
	charLower   = "abcdefghijklmnopqrstuvwxyz"
	charUpper   = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	charNumbers = "0123456789"
	charSymbols = "!@#$%^&*()_+-=[]{}|'"
)

// generateIAMPassword generates a random password of a given length, matching the
// most restrictive iam password policy.
func generateIAMPassword(length int) string {
	const charset = charLower + charUpper + charNumbers + charSymbols

	result := make([]byte, length)
	charsetSize := big.NewInt(int64(len(charset)))

	// rather than trying to artificially add specific characters from each
	// class to the password to match the policy, we generate passwords
	// randomly and reject those that don't match.
	//
	// Even in the worst case, this tends to take less than 10 tries to find a
	// matching password. Any sufficiently long password is likely to succeed
	// on the first try
	for n := 0; n < 100000; n++ {
		for i := range result {
			r, err := rand.Int(rand.Reader, charsetSize)
			if err != nil {
				panic(err)
			}
			if !r.IsInt64() {
				panic("rand.Int() not representable as an Int64")
			}

			result[i] = charset[r.Int64()]
		}

		if !checkIAMPwdPolicy(result) {
			continue
		}

		return string(result)
	}

	panic("failed to generate acceptable password")
}

// Check the generated password contains all character classes listed in the
// IAM password policy.
func checkIAMPwdPolicy(pass []byte) bool {
	return (bytes.ContainsAny(pass, charLower) &&
		bytes.ContainsAny(pass, charNumbers) &&
		bytes.ContainsAny(pass, charSymbols) &&
		bytes.ContainsAny(pass, charUpper))
}

func resourceAwsIamUserLoginProfileCreate(d *schema.ResourceData, meta interface{}) error {
	iamconn := meta.(*AWSClient).iamconn

	encryptionKey, err := encryption.RetrieveGPGKey(strings.TrimSpace(d.Get("pgp_key").(string)))
	if err != nil {
		return err
	}

	username := d.Get("user").(string)
	passwordResetRequired := d.Get("password_reset_required").(bool)
	passwordLength := d.Get("password_length").(int)

	_, err = iamconn.GetLoginProfile(&iam.GetLoginProfileInput{
		UserName: aws.String(username),
	})
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok && awsErr.Code() != "NoSuchEntity" {
			// If there is already a login profile, bring it under management (to prevent
			// resource creation diffs) - we will never modify it, but obviously cannot
			// set the password.
			d.SetId(username)
			d.Set("key_fingerprint", "")
			d.Set("encrypted_password", "")
			return nil
		}
	}

	initialPassword := generateIAMPassword(passwordLength)

	fingerprint, encrypted, err := encryption.EncryptValue(encryptionKey, initialPassword, "Password")
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
		if awsErr, ok := err.(awserr.Error); ok && awsErr.Code() == "EntityAlreadyExists" {
			// If there is already a login profile, bring it under management (to prevent
			// resource creation diffs) - we will never modify it, but obviously cannot
			// set the password.
			d.SetId(username)
			d.Set("key_fingerprint", "")
			d.Set("encrypted_password", "")
			return nil
		}
		return fmt.Errorf("Error creating IAM User Login Profile for %q: %s", username, err)
	}

	d.SetId(*createResp.LoginProfile.UserName)
	d.Set("key_fingerprint", fingerprint)
	d.Set("encrypted_password", encrypted)
	return nil
}
