package aws

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"

	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsAmiFromInstance() *schema.Resource {
	// Inherit all of the common AMI attributes from aws_ami, since we're
	// implicitly creating an aws_ami resource.
	resourceSchema := resourceAwsAmiCommonSchema(true)

	// Additional attributes unique to the copy operation.
	resourceSchema["source_instance_id"] = &schema.Schema{
		Type:     schema.TypeString,
		Required: true,
		ForceNew: true,
	}
	resourceSchema["snapshot_without_reboot"] = &schema.Schema{
		Type:     schema.TypeBool,
		Optional: true,
		ForceNew: true,
	}

	return &schema.Resource{
		Create: resourceAwsAmiFromInstanceCreate,

		Schema: resourceSchema,

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
	d.Set("id", id)
	d.Set("manage_ebs_snapshots", true)
	d.SetPartial("id")
	d.SetPartial("manage_ebs_snapshots")
	d.Partial(false)

	_, err = resourceAwsAmiWaitForAvailable(id, client)
	if err != nil {
		return err
	}

	return resourceAwsAmiUpdate(d, meta)
}
