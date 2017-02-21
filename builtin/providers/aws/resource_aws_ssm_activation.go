package aws

import (
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsSsmActivation() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsSsmActivationCreate,
		Read:   resourceAwsSsmActivationRead,
		Delete: resourceAwsSsmActivationDelete,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"description": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"expired": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"expiration_date": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"iam_role": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"registration_limit": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				ForceNew: true,
			},
			"registration_count": &schema.Schema{
				Type:     schema.TypeInt,
				Computed: true,
			},
		},
	}
}

func resourceAwsSsmActivationCreate(d *schema.ResourceData, meta interface{}) error {
	ssmconn := meta.(*AWSClient).ssmconn

	log.Printf("[DEBUG] SSM activation create: %s", d.Id())

	activationInput := &ssm.CreateActivationInput{
		IamRole: aws.String(d.Get("name").(string)),
	}

	if _, ok := d.GetOk("name"); ok {
		activationInput.DefaultInstanceName = aws.String(d.Get("name").(string))
	}

	if _, ok := d.GetOk("description"); ok {
		activationInput.Description = aws.String(d.Get("description").(string))
	}

	if _, ok := d.GetOk("expiration_date"); ok {
		activationInput.ExpirationDate = aws.Time(d.Get("expiration_date").(time.Time))
	}

	if _, ok := d.GetOk("iam_role"); ok {
		activationInput.IamRole = aws.String(d.Get("iam_role").(string))
	}

	if _, ok := d.GetOk("registration_limit"); ok {
		activationInput.RegistrationLimit = aws.Int64(int64(d.Get("registration_limit").(int)))
	}

	// Retry to allow iam_role to be created and policy attachment to take place
	var resp *ssm.CreateActivationOutput
	err := resource.Retry(30*time.Second, func() *resource.RetryError {
		var err error

		resp, err = ssmconn.CreateActivation(activationInput)

		if err != nil {
			return resource.RetryableError(err)
		}

		return resource.NonRetryableError(err)
	})

	if err != nil {
		return errwrap.Wrapf("[ERROR] Error creating SSM activation: {{err}}", err)
	}

	if resp.ActivationId == nil {
		return fmt.Errorf("[ERROR] ActivationId was nil")
	}
	d.SetId(*resp.ActivationId)

	return resourceAwsSsmActivationRead(d, meta)
}

func resourceAwsSsmActivationRead(d *schema.ResourceData, meta interface{}) error {
	ssmconn := meta.(*AWSClient).ssmconn

	log.Printf("[DEBUG] Reading SSM Activation: %s", d.Id())

	params := &ssm.DescribeActivationsInput{
		Filters: []*ssm.DescribeActivationsFilter{
			{
				FilterKey: aws.String("ActivationIds"),
				FilterValues: []*string{
					aws.String(d.Id()),
				},
			},
		},
		MaxResults: aws.Int64(1),
	}

	resp, err := ssmconn.DescribeActivations(params)

	if err != nil {
		return errwrap.Wrapf("[ERROR] Error reading SSM activation: {{err}}", err)
	}
	if resp.ActivationList == nil || len(resp.ActivationList) == 0 {
		return fmt.Errorf("[ERROR] ActivationList was nil or empty")
	}

	activation := resp.ActivationList[0] // Only 1 result as MaxResults is 1 above
	d.Set("name", activation.DefaultInstanceName)
	d.Set("description", activation.Description)
	d.Set("expiration_date", activation.ExpirationDate)
	d.Set("expired", activation.Expired)
	d.Set("iam_role", activation.IamRole)
	d.Set("registration_limit", activation.RegistrationLimit)
	d.Set("registration_count", activation.RegistrationsCount)

	return nil
}

func resourceAwsSsmActivationDelete(d *schema.ResourceData, meta interface{}) error {
	ssmconn := meta.(*AWSClient).ssmconn

	log.Printf("[DEBUG] Deleting SSM Activation: %s", d.Id())

	params := &ssm.DeleteActivationInput{
		ActivationId: aws.String(d.Id()),
	}

	_, err := ssmconn.DeleteActivation(params)

	if err != nil {
		return errwrap.Wrapf("[ERROR] Error deleting SSM activation: {{err}}", err)
	}

	return nil
}
