package aws

import (
	"fmt"
	"log"
	"net/url"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/iam"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsIamRolePolicy() *schema.Resource {
	return &schema.Resource{
		// PutRolePolicy API is idempotent, so these can be the same.
		Create: resourceAwsIamRolePolicyPut,
		Update: resourceAwsIamRolePolicyPut,

		Read:   resourceAwsIamRolePolicyRead,
		Delete: resourceAwsIamRolePolicyDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"policy": {
				Type:             schema.TypeString,
				Required:         true,
				ValidateFunc:     validateIAMPolicyJson,
				DiffSuppressFunc: suppressEquivalentAwsPolicyDiffs,
			},
			"name": {
				Type:          schema.TypeString,
				Optional:      true,
				Computed:      true,
				ForceNew:      true,
				ConflictsWith: []string{"name_prefix"},
				ValidateFunc:  validateIamRolePolicyName,
			},
			"name_prefix": {
				Type:          schema.TypeString,
				Optional:      true,
				ForceNew:      true,
				ConflictsWith: []string{"name"},
				ValidateFunc:  validateIamRolePolicyNamePrefix,
			},
			"role": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
		},
	}
}

func resourceAwsIamRolePolicyPut(d *schema.ResourceData, meta interface{}) error {
	iamconn := meta.(*AWSClient).iamconn

	request := &iam.PutRolePolicyInput{
		RoleName:       aws.String(d.Get("role").(string)),
		PolicyDocument: aws.String(d.Get("policy").(string)),
	}

	var policyName string
	if v, ok := d.GetOk("name"); ok {
		policyName = v.(string)
	} else if v, ok := d.GetOk("name_prefix"); ok {
		policyName = resource.PrefixedUniqueId(v.(string))
	} else {
		policyName = resource.UniqueId()
	}
	request.PolicyName = aws.String(policyName)

	if _, err := iamconn.PutRolePolicy(request); err != nil {
		return fmt.Errorf("Error putting IAM role policy %s: %s", *request.PolicyName, err)
	}

	d.SetId(fmt.Sprintf("%s:%s", *request.RoleName, *request.PolicyName))
	return resourceAwsIamRolePolicyRead(d, meta)
}

func resourceAwsIamRolePolicyRead(d *schema.ResourceData, meta interface{}) error {
	iamconn := meta.(*AWSClient).iamconn

	role, name, err := resourceAwsIamRolePolicyParseId(d.Id())
	if err != nil {
		return err
	}

	request := &iam.GetRolePolicyInput{
		PolicyName: aws.String(name),
		RoleName:   aws.String(role),
	}

	getResp, err := iamconn.GetRolePolicy(request)
	if err != nil {
		if isAWSErr(err, iam.ErrCodeNoSuchEntityException, "") {
			log.Printf("[WARN] IAM Role Policy (%s) for %s not found, removing from state", name, role)
			d.SetId("")
			return nil
		}
		return fmt.Errorf("Error reading IAM policy %s from role %s: %s", name, role, err)
	}

	if getResp.PolicyDocument == nil {
		return fmt.Errorf("GetRolePolicy returned a nil policy document")
	}

	policy, err := url.QueryUnescape(*getResp.PolicyDocument)
	if err != nil {
		return err
	}
	if err := d.Set("policy", policy); err != nil {
		return err
	}
	if err := d.Set("name", name); err != nil {
		return err
	}
	return d.Set("role", role)
}

func resourceAwsIamRolePolicyDelete(d *schema.ResourceData, meta interface{}) error {
	iamconn := meta.(*AWSClient).iamconn

	role, name, err := resourceAwsIamRolePolicyParseId(d.Id())
	if err != nil {
		return err
	}

	request := &iam.DeleteRolePolicyInput{
		PolicyName: aws.String(name),
		RoleName:   aws.String(role),
	}

	if _, err := iamconn.DeleteRolePolicy(request); err != nil {
		if isAWSErr(err, iam.ErrCodeNoSuchEntityException, "") {
			return nil
		}
		return fmt.Errorf("Error deleting IAM role policy %s: %s", d.Id(), err)
	}
	return nil
}

func resourceAwsIamRolePolicyParseId(id string) (roleName, policyName string, err error) {
	parts := strings.SplitN(id, ":", 2)
	if len(parts) != 2 {
		err = fmt.Errorf("role_policy id must be of the form <role name>:<policy name>")
		return
	}

	roleName = parts[0]
	policyName = parts[1]
	return
}
