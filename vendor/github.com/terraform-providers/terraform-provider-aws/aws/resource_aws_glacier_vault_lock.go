package aws

import (
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/glacier"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
)

func resourceAwsGlacierVaultLock() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsGlacierVaultLockCreate,
		Read:   resourceAwsGlacierVaultLockRead,
		// Allow ignore_deletion_error update
		Update: schema.Noop,
		Delete: resourceAwsGlacierVaultLockDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"complete_lock": {
				Type:     schema.TypeBool,
				Required: true,
				ForceNew: true,
			},
			"ignore_deletion_error": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"policy": {
				Type:             schema.TypeString,
				Required:         true,
				ForceNew:         true,
				DiffSuppressFunc: suppressEquivalentAwsPolicyDiffs,
				ValidateFunc:     validateIAMPolicyJson,
			},
			"vault_name": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.NoZeroValues,
			},
		},
	}
}

func resourceAwsGlacierVaultLockCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).glacierconn
	vaultName := d.Get("vault_name").(string)

	input := &glacier.InitiateVaultLockInput{
		AccountId: aws.String("-"),
		Policy: &glacier.VaultLockPolicy{
			Policy: aws.String(d.Get("policy").(string)),
		},
		VaultName: aws.String(vaultName),
	}

	log.Printf("[DEBUG] Initiating Glacier Vault Lock: %s", input)
	output, err := conn.InitiateVaultLock(input)
	if err != nil {
		return fmt.Errorf("error initiating Glacier Vault Lock: %s", err)
	}

	d.SetId(vaultName)

	if !d.Get("complete_lock").(bool) {
		return resourceAwsGlacierVaultLockRead(d, meta)
	}

	completeLockInput := &glacier.CompleteVaultLockInput{
		LockId:    output.LockId,
		VaultName: aws.String(vaultName),
	}

	log.Printf("[DEBUG] Completing Glacier Vault (%s) Lock: %s", vaultName, completeLockInput)
	if _, err := conn.CompleteVaultLock(completeLockInput); err != nil {
		return fmt.Errorf("error completing Glacier Vault (%s) Lock: %s", vaultName, err)
	}

	if err := waitForGlacierVaultLockCompletion(conn, vaultName); err != nil {
		return fmt.Errorf("error waiting for Glacier Vault Lock (%s) completion: %s", d.Id(), err)
	}

	return resourceAwsGlacierVaultLockRead(d, meta)
}

func resourceAwsGlacierVaultLockRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).glacierconn

	input := &glacier.GetVaultLockInput{
		AccountId: aws.String("-"),
		VaultName: aws.String(d.Id()),
	}

	log.Printf("[DEBUG] Reading Glacier Vault Lock (%s): %s", d.Id(), input)
	output, err := conn.GetVaultLock(input)

	if isAWSErr(err, glacier.ErrCodeResourceNotFoundException, "") {
		log.Printf("[WARN] Glacier Vault Lock (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	if err != nil {
		return fmt.Errorf("error reading Glacier Vault Lock (%s): %s", d.Id(), err)
	}

	if output == nil {
		log.Printf("[WARN] Glacier Vault Lock (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	d.Set("complete_lock", aws.StringValue(output.State) == "Locked")
	d.Set("policy", output.Policy)
	d.Set("vault_name", d.Id())

	return nil
}

func resourceAwsGlacierVaultLockDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).glacierconn

	input := &glacier.AbortVaultLockInput{
		VaultName: aws.String(d.Id()),
	}

	log.Printf("[DEBUG] Aborting Glacier Vault Lock (%s): %s", d.Id(), input)
	_, err := conn.AbortVaultLock(input)

	if isAWSErr(err, glacier.ErrCodeResourceNotFoundException, "") {
		return nil
	}

	if err != nil && !d.Get("ignore_deletion_error").(bool) {
		return fmt.Errorf("error aborting Glacier Vault Lock (%s): %s", d.Id(), err)
	}

	return nil
}

func glacierVaultLockRefreshFunc(conn *glacier.Glacier, vaultName string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		input := &glacier.GetVaultLockInput{
			AccountId: aws.String("-"),
			VaultName: aws.String(vaultName),
		}

		log.Printf("[DEBUG] Reading Glacier Vault Lock (%s): %s", vaultName, input)
		output, err := conn.GetVaultLock(input)

		if isAWSErr(err, glacier.ErrCodeResourceNotFoundException, "") {
			return nil, "", nil
		}

		if err != nil {
			return nil, "", fmt.Errorf("error reading Glacier Vault Lock (%s): %s", vaultName, err)
		}

		if output == nil {
			return nil, "", nil
		}

		return output, aws.StringValue(output.State), nil
	}
}

func waitForGlacierVaultLockCompletion(conn *glacier.Glacier, vaultName string) error {
	stateConf := &resource.StateChangeConf{
		Pending: []string{"InProgress"},
		Target:  []string{"Locked"},
		Refresh: glacierVaultLockRefreshFunc(conn, vaultName),
		Timeout: 5 * time.Minute,
	}

	log.Printf("[DEBUG] Waiting for Glacier Vault Lock (%s) completion", vaultName)
	_, err := stateConf.WaitForState()

	return err
}
