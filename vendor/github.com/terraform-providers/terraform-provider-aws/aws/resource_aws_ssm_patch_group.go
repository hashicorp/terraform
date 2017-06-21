package aws

import (
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsSsmPatchGroup() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsSsmPatchGroupCreate,
		Read:   resourceAwsSsmPatchGroupRead,
		Delete: resourceAwsSsmPatchGroupDelete,

		Schema: map[string]*schema.Schema{
			"baseline_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"patch_group": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
		},
	}
}

func resourceAwsSsmPatchGroupCreate(d *schema.ResourceData, meta interface{}) error {
	ssmconn := meta.(*AWSClient).ssmconn

	params := &ssm.RegisterPatchBaselineForPatchGroupInput{
		BaselineId: aws.String(d.Get("baseline_id").(string)),
		PatchGroup: aws.String(d.Get("patch_group").(string)),
	}

	resp, err := ssmconn.RegisterPatchBaselineForPatchGroup(params)
	if err != nil {
		return err
	}

	d.SetId(*resp.PatchGroup)
	return resourceAwsSsmPatchGroupRead(d, meta)
}

func resourceAwsSsmPatchGroupRead(d *schema.ResourceData, meta interface{}) error {
	ssmconn := meta.(*AWSClient).ssmconn

	params := &ssm.DescribePatchGroupsInput{}

	resp, err := ssmconn.DescribePatchGroups(params)
	if err != nil {
		return err
	}

	found := false
	for _, t := range resp.Mappings {
		if *t.PatchGroup == d.Id() {
			found = true

			d.Set("patch_group", t.PatchGroup)
			d.Set("baseline_id", t.BaselineIdentity.BaselineId)
		}
	}

	if !found {
		log.Printf("[INFO] Patch Group not found. Removing from state")
		d.SetId("")
		return nil
	}

	return nil

}

func resourceAwsSsmPatchGroupDelete(d *schema.ResourceData, meta interface{}) error {
	ssmconn := meta.(*AWSClient).ssmconn

	log.Printf("[INFO] Deleting SSM Patch Group: %s", d.Id())

	params := &ssm.DeregisterPatchBaselineForPatchGroupInput{
		BaselineId: aws.String(d.Get("baseline_id").(string)),
		PatchGroup: aws.String(d.Get("patch_group").(string)),
	}

	_, err := ssmconn.DeregisterPatchBaselineForPatchGroup(params)
	if err != nil {
		return err
	}

	return nil
}
