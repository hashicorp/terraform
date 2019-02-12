package aws

import (
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ram"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsRamResourceShare() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsRamResourceShareCreate,
		Read:   resourceAwsRamResourceShareRead,
		Update: resourceAwsRamResourceShareUpdate,
		Delete: resourceAwsRamResourceShareDelete,

		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(5 * time.Minute),
			Delete: schema.DefaultTimeout(5 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},

			"allow_external_principals": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},

			"tags": tagsSchema(),
		},
	}
}

func resourceAwsRamResourceShareCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ramconn

	request := &ram.CreateResourceShareInput{
		Name:                    aws.String(d.Get("name").(string)),
		AllowExternalPrincipals: aws.Bool(d.Get("allow_external_principals").(bool)),
	}

	if v, ok := d.GetOk("tags"); ok {
		tags := tagsFromMapRAM(v.(map[string]interface{}))
		request.Tags = tags
	}

	log.Println("[DEBUG] Create RAM resource share request:", request)
	createResp, err := conn.CreateResourceShare(request)
	if err != nil {
		return fmt.Errorf("Error creating RAM resource share: %s", err)
	}

	d.SetId(aws.StringValue(createResp.ResourceShare.ResourceShareArn))

	stateConf := &resource.StateChangeConf{
		Pending: []string{ram.ResourceShareStatusPending},
		Target:  []string{ram.ResourceShareStatusActive},
		Refresh: resourceAwsRamResourceShareStateRefreshFunc(conn, d.Id()),
		Timeout: d.Timeout(schema.TimeoutCreate),
	}

	_, err = stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf("Error waiting for RAM resource share (%s) to become ready: %s", d.Id(), err)
	}

	return resourceAwsRamResourceShareRead(d, meta)
}

func resourceAwsRamResourceShareRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ramconn

	request := &ram.GetResourceSharesInput{
		ResourceShareArns: []*string{aws.String(d.Id())},
		ResourceOwner:     aws.String(ram.ResourceOwnerSelf),
	}

	output, err := conn.GetResourceShares(request)
	if err != nil {
		if isAWSErr(err, ram.ErrCodeUnknownResourceException, "") {
			log.Printf("[WARN] No RAM resource share by ARN (%s) found, removing from state", d.Id())
			d.SetId("")
			return nil
		}
		return fmt.Errorf("Error reading RAM resource share %s: %s", d.Id(), err)
	}

	if len(output.ResourceShares) == 0 {
		log.Printf("[WARN] No RAM resource share by ARN (%s) found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	resourceShare := output.ResourceShares[0]

	if aws.StringValue(resourceShare.Status) != ram.ResourceShareStatusActive {
		log.Printf("[WARN] RAM resource share (%s) delet(ing|ed), removing from state", d.Id())
		d.SetId("")
		return nil
	}

	d.Set("name", resourceShare.Name)
	d.Set("allow_external_principals", resourceShare.AllowExternalPrincipals)

	if err := d.Set("tags", tagsToMapRAM(resourceShare.Tags)); err != nil {
		return fmt.Errorf("Error setting tags: %s", err)
	}

	return nil
}

func resourceAwsRamResourceShareUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ramconn

	d.Partial(true)

	if d.HasChange("name") || d.HasChange("allow_external_principals") {
		request := &ram.UpdateResourceShareInput{
			ResourceShareArn:        aws.String(d.Id()),
			Name:                    aws.String(d.Get("name").(string)),
			AllowExternalPrincipals: aws.Bool(d.Get("allow_external_principals").(bool)),
		}

		log.Println("[DEBUG] Update RAM resource share request:", request)
		_, err := conn.UpdateResourceShare(request)
		if err != nil {
			if isAWSErr(err, ram.ErrCodeUnknownResourceException, "") {
				log.Printf("[WARN] No RAM resource share by ARN (%s) found", d.Id())
				d.SetId("")
				return nil
			}
			return fmt.Errorf("Error updating RAM resource share %s: %s", d.Id(), err)
		}

		d.SetPartial("name")
		d.SetPartial("allow_external_principals")
	}

	if d.HasChange("tags") {
		// Reset all tags to empty set
		oraw, nraw := d.GetChange("tags")
		o := oraw.(map[string]interface{})
		n := nraw.(map[string]interface{})
		c, r := diffTagsRAM(tagsFromMapRAM(o), tagsFromMapRAM(n))

		if len(r) > 0 {
			_, err := conn.UntagResource(&ram.UntagResourceInput{
				ResourceShareArn: aws.String(d.Id()),
				TagKeys:          tagKeysRam(r),
			})
			if err != nil {
				return fmt.Errorf("Error deleting RAM resource share tags: %s", err)
			}
		}

		if len(c) > 0 {
			input := &ram.TagResourceInput{
				ResourceShareArn: aws.String(d.Id()),
				Tags:             c,
			}
			_, err := conn.TagResource(input)
			if err != nil {
				return fmt.Errorf("Error updating RAM resource share tags: %s", err)
			}
		}

		d.SetPartial("tags")
	}

	d.Partial(false)

	return resourceAwsRamResourceShareRead(d, meta)
}

func resourceAwsRamResourceShareDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ramconn

	deleteResourceShareInput := &ram.DeleteResourceShareInput{
		ResourceShareArn: aws.String(d.Id()),
	}

	log.Println("[DEBUG] Delete RAM resource share request:", deleteResourceShareInput)
	_, err := conn.DeleteResourceShare(deleteResourceShareInput)
	if err != nil {
		if isAWSErr(err, ram.ErrCodeUnknownResourceException, "") {
			return nil
		}
		return fmt.Errorf("Error deleting RAM resource share %s: %s", d.Id(), err)
	}

	stateConf := &resource.StateChangeConf{
		Pending: []string{ram.ResourceShareStatusDeleting},
		Target:  []string{ram.ResourceShareStatusDeleted},
		Refresh: resourceAwsRamResourceShareStateRefreshFunc(conn, d.Id()),
		Timeout: d.Timeout(schema.TimeoutDelete),
	}

	_, err = stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf("Error waiting for RAM resource share (%s) to become ready: %s", d.Id(), err)
	}

	return nil
}

func resourceAwsRamResourceShareStateRefreshFunc(conn *ram.RAM, resourceShareArn string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		request := &ram.GetResourceSharesInput{
			ResourceShareArns: []*string{aws.String(resourceShareArn)},
			ResourceOwner:     aws.String(ram.ResourceOwnerSelf),
		}

		output, err := conn.GetResourceShares(request)

		if err != nil {
			return nil, ram.ResourceShareStatusFailed, err
		}

		if len(output.ResourceShares) == 0 {
			return nil, ram.ResourceShareStatusDeleted, nil
		}

		resourceShare := output.ResourceShares[0]

		return resourceShare, aws.StringValue(resourceShare.Status), nil
	}
}
