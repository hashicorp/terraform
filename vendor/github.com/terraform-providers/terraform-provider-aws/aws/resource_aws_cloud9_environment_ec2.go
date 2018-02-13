package aws

import (
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloud9"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsCloud9EnvironmentEc2() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsCloud9EnvironmentEc2Create,
		Read:   resourceAwsCloud9EnvironmentEc2Read,
		Update: resourceAwsCloud9EnvironmentEc2Update,
		Delete: resourceAwsCloud9EnvironmentEc2Delete,

		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"instance_type": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"automatic_stop_time_minutes": {
				Type:     schema.TypeInt,
				Optional: true,
				ForceNew: true,
			},
			"description": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"owner_arn": {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				ForceNew:     true,
				ValidateFunc: validateArn,
			},
			"subnet_id": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"type": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceAwsCloud9EnvironmentEc2Create(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).cloud9conn

	params := &cloud9.CreateEnvironmentEC2Input{
		InstanceType:       aws.String(d.Get("instance_type").(string)),
		Name:               aws.String(d.Get("name").(string)),
		ClientRequestToken: aws.String(resource.UniqueId()),
	}

	if v, ok := d.GetOk("automatic_stop_time_minutes"); ok {
		params.AutomaticStopTimeMinutes = aws.Int64(int64(v.(int)))
	}
	if v, ok := d.GetOk("description"); ok {
		params.Description = aws.String(v.(string))
	}
	if v, ok := d.GetOk("owner_arn"); ok {
		params.OwnerArn = aws.String(v.(string))
	}
	if v, ok := d.GetOk("subnet_id"); ok {
		params.SubnetId = aws.String(v.(string))
	}

	out, err := conn.CreateEnvironmentEC2(params)
	if err != nil {
		return err
	}
	d.SetId(*out.EnvironmentId)

	stateConf := resource.StateChangeConf{
		Pending: []string{
			cloud9.EnvironmentStatusConnecting,
			cloud9.EnvironmentStatusCreating,
		},
		Target: []string{
			cloud9.EnvironmentStatusReady,
		},
		Timeout: 10 * time.Minute,
		Refresh: func() (interface{}, string, error) {
			out, err := conn.DescribeEnvironmentStatus(&cloud9.DescribeEnvironmentStatusInput{
				EnvironmentId: aws.String(d.Id()),
			})
			if err != nil {
				return 42, "", err
			}

			status := *out.Status
			var sErr error
			if status == cloud9.EnvironmentStatusError && out.Message != nil {
				sErr = fmt.Errorf("Reason: %s", *out.Message)
			}

			return out, status, sErr
		},
	}
	_, err = stateConf.WaitForState()
	if err != nil {
		return err
	}

	return resourceAwsCloud9EnvironmentEc2Read(d, meta)
}

func resourceAwsCloud9EnvironmentEc2Read(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).cloud9conn

	log.Printf("[INFO] Reading Cloud9 Environment EC2 %s", d.Id())

	out, err := conn.DescribeEnvironments(&cloud9.DescribeEnvironmentsInput{
		EnvironmentIds: []*string{aws.String(d.Id())},
	})
	if err != nil {
		if isAWSErr(err, cloud9.ErrCodeNotFoundException, "") {
			log.Printf("[WARN] Cloud9 Environment EC2 (%s) not found, removing from state", d.Id())
			d.SetId("")
			return nil
		}
		return err
	}
	if len(out.Environments) == 0 {
		log.Printf("[WARN] Cloud9 Environment EC2 (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}
	env := out.Environments[0]

	d.Set("arn", env.Arn)
	d.Set("description", env.Description)
	d.Set("name", env.Name)
	d.Set("owner_arn", env.OwnerArn)
	d.Set("type", env.Type)

	log.Printf("[DEBUG] Received Cloud9 Environment EC2: %s", env)

	return nil
}

func resourceAwsCloud9EnvironmentEc2Update(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).cloud9conn

	input := cloud9.UpdateEnvironmentInput{
		Description:   aws.String(d.Get("description").(string)),
		EnvironmentId: aws.String(d.Id()),
		Name:          aws.String(d.Get("name").(string)),
	}

	log.Printf("[INFO] Updating Cloud9 Environment EC2: %s", input)

	out, err := conn.UpdateEnvironment(&input)
	if err != nil {
		return err
	}

	log.Printf("[DEBUG] Cloud9 Environment EC2 updated: %s", out)

	return resourceAwsCloud9EnvironmentEc2Read(d, meta)
}

func resourceAwsCloud9EnvironmentEc2Delete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).cloud9conn

	_, err := conn.DeleteEnvironment(&cloud9.DeleteEnvironmentInput{
		EnvironmentId: aws.String(d.Id()),
	})
	if err != nil {
		return err
	}
	err = resource.Retry(1*time.Minute, func() *resource.RetryError {
		out, err := conn.DescribeEnvironments(&cloud9.DescribeEnvironmentsInput{
			EnvironmentIds: []*string{aws.String(d.Id())},
		})
		if err != nil {
			if isAWSErr(err, cloud9.ErrCodeNotFoundException, "") {
				return nil
			}
			// :'-(
			if isAWSErr(err, "AccessDeniedException", "is not authorized to access this resource") {
				return nil
			}
			return resource.NonRetryableError(err)
		}
		if len(out.Environments) == 0 {
			return nil
		}
		return resource.RetryableError(fmt.Errorf("Cloud9 EC2 Environment %q still exists", d.Id()))
	})
	if err != nil {
		return err
	}

	return err
}
