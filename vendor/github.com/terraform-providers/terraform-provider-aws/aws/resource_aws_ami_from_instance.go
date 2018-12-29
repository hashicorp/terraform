package aws

import (
	"bytes"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"

	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsAmiFromInstance() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsAmiFromInstanceCreate,

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(AWSAMIRetryTimeout),
			Update: schema.DefaultTimeout(AWSAMIRetryTimeout),
			Delete: schema.DefaultTimeout(AWSAMIDeleteRetryTimeout),
		},

		Schema: map[string]*schema.Schema{
			"architecture": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"description": {
				Type:     schema.TypeString,
				Optional: true,
			},
			// The following block device attributes intentionally mimick the
			// corresponding attributes on aws_instance, since they have the
			// same meaning.
			// However, we don't use root_block_device here because the constraint
			// on which root device attributes can be overridden for an instance to
			// not apply when registering an AMI.
			"ebs_block_device": {
				Type:     schema.TypeSet,
				Optional: true,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"delete_on_termination": {
							Type:     schema.TypeBool,
							Computed: true,
						},

						"device_name": {
							Type:     schema.TypeString,
							Computed: true,
						},

						"encrypted": {
							Type:     schema.TypeBool,
							Computed: true,
						},

						"iops": {
							Type:     schema.TypeInt,
							Computed: true,
						},

						"snapshot_id": {
							Type:     schema.TypeString,
							Computed: true,
						},

						"volume_size": {
							Type:     schema.TypeInt,
							Computed: true,
						},

						"volume_type": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
				Set: func(v interface{}) int {
					var buf bytes.Buffer
					m := v.(map[string]interface{})
					buf.WriteString(fmt.Sprintf("%s-", m["device_name"].(string)))
					buf.WriteString(fmt.Sprintf("%s-", m["snapshot_id"].(string)))
					return hashcode.String(buf.String())
				},
			},
			"ena_support": {
				Type:     schema.TypeBool,
				Computed: true,
			},
			"ephemeral_block_device": {
				Type:     schema.TypeSet,
				Optional: true,
				Computed: true,
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"device_name": {
							Type:     schema.TypeString,
							Computed: true,
						},

						"virtual_name": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
				Set: func(v interface{}) int {
					var buf bytes.Buffer
					m := v.(map[string]interface{})
					buf.WriteString(fmt.Sprintf("%s-", m["device_name"].(string)))
					buf.WriteString(fmt.Sprintf("%s-", m["virtual_name"].(string)))
					return hashcode.String(buf.String())
				},
			},
			"image_location": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"kernel_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			// Not a public attribute; used to let the aws_ami_copy and aws_ami_from_instance
			// resources record that they implicitly created new EBS snapshots that we should
			// now manage. Not set by aws_ami, since the snapshots used there are presumed to
			// be independently managed.
			"manage_ebs_snapshots": {
				Type:     schema.TypeBool,
				Computed: true,
				ForceNew: true,
			},
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"ramdisk_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"root_device_name": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"root_snapshot_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"source_instance_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"snapshot_without_reboot": {
				Type:     schema.TypeBool,
				Optional: true,
				ForceNew: true,
			},
			"sriov_net_support": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"tags": tagsSchema(),
			"virtualization_type": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},

		// The remaining operations are shared with the generic aws_ami resource,
		// since the aws_ami_copy resource only differs in how it's created.
		Read:   resourceAwsAmiRead,
		Update: resourceAwsAmiUpdate,
		Delete: resourceAwsAmiDelete,
	}
}

func resourceAwsAmiFromInstanceCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*AWSClient).ec2conn

	req := &ec2.CreateImageInput{
		Name:        aws.String(d.Get("name").(string)),
		Description: aws.String(d.Get("description").(string)),
		InstanceId:  aws.String(d.Get("source_instance_id").(string)),
		NoReboot:    aws.Bool(d.Get("snapshot_without_reboot").(bool)),
	}

	res, err := client.CreateImage(req)
	if err != nil {
		return err
	}

	id := *res.ImageId
	d.SetId(id)
	d.Partial(true) // make sure we record the id even if the rest of this gets interrupted
	d.Set("manage_ebs_snapshots", true)
	d.SetPartial("manage_ebs_snapshots")
	d.Partial(false)

	_, err = resourceAwsAmiWaitForAvailable(d.Timeout(schema.TimeoutCreate), id, client)
	if err != nil {
		return err
	}

	return resourceAwsAmiUpdate(d, meta)
}
