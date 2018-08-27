package aws

import (
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dax"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsDaxParameterGroup() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsDaxParameterGroupCreate,
		Read:   resourceAwsDaxParameterGroupRead,
		Update: resourceAwsDaxParameterGroupUpdate,
		Delete: resourceAwsDaxParameterGroupDelete,

		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"description": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"parameters": {
				Type:     schema.TypeSet,
				Optional: true,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:     schema.TypeString,
							Required: true,
						},
						"value": {
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},
		},
	}
}

func resourceAwsDaxParameterGroupCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).daxconn

	input := &dax.CreateParameterGroupInput{
		ParameterGroupName: aws.String(d.Get("name").(string)),
	}
	if v, ok := d.GetOk("description"); ok {
		input.Description = aws.String(v.(string))
	}

	_, err := conn.CreateParameterGroup(input)
	if err != nil {
		return err
	}

	d.SetId(d.Get("name").(string))

	if len(d.Get("parameters").(*schema.Set).List()) > 0 {
		return resourceAwsDaxParameterGroupUpdate(d, meta)
	}
	return resourceAwsDaxParameterGroupRead(d, meta)
}

func resourceAwsDaxParameterGroupRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).daxconn

	resp, err := conn.DescribeParameterGroups(&dax.DescribeParameterGroupsInput{
		ParameterGroupNames: []*string{aws.String(d.Id())},
	})
	if err != nil {
		if isAWSErr(err, dax.ErrCodeParameterGroupNotFoundFault, "") {
			log.Printf("[WARN] DAX ParameterGroup %q not found, removing from state", d.Id())
			d.SetId("")
			return nil
		}
		return err
	}

	if len(resp.ParameterGroups) == 0 {
		log.Printf("[WARN] DAX ParameterGroup %q not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	pg := resp.ParameterGroups[0]

	paramresp, err := conn.DescribeParameters(&dax.DescribeParametersInput{
		ParameterGroupName: aws.String(d.Id()),
	})
	if err != nil {
		if isAWSErr(err, dax.ErrCodeParameterGroupNotFoundFault, "") {
			log.Printf("[WARN] DAX ParameterGroup %q not found, removing from state", d.Id())
			d.SetId("")
			return nil
		}
		return err
	}

	d.Set("name", pg.ParameterGroupName)
	desc := pg.Description
	// default description is " "
	if desc != nil && *desc == " " {
		*desc = ""
	}
	d.Set("description", desc)
	d.Set("parameters", flattenDaxParameterGroupParameters(paramresp.Parameters))
	return nil
}

func resourceAwsDaxParameterGroupUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).daxconn

	input := &dax.UpdateParameterGroupInput{
		ParameterGroupName: aws.String(d.Id()),
	}

	if d.HasChange("parameters") {
		input.ParameterNameValues = expandDaxParameterGroupParameterNameValue(
			d.Get("parameters").(*schema.Set).List(),
		)
	}

	_, err := conn.UpdateParameterGroup(input)
	if err != nil {
		return err
	}

	return resourceAwsDaxParameterGroupRead(d, meta)
}

func resourceAwsDaxParameterGroupDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).daxconn

	input := &dax.DeleteParameterGroupInput{
		ParameterGroupName: aws.String(d.Id()),
	}

	_, err := conn.DeleteParameterGroup(input)
	if err != nil {
		if isAWSErr(err, dax.ErrCodeParameterGroupNotFoundFault, "") {
			return nil
		}
		return err
	}

	return nil
}
