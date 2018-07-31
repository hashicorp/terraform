package aws

import (
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dax"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsDaxSubnetGroup() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsDaxSubnetGroupCreate,
		Read:   resourceAwsDaxSubnetGroupRead,
		Update: resourceAwsDaxSubnetGroupUpdate,
		Delete: resourceAwsDaxSubnetGroupDelete,

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
			},
			"subnet_ids": &schema.Schema{
				Type:     schema.TypeSet,
				Required: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},
			"vpc_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceAwsDaxSubnetGroupCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).daxconn

	input := &dax.CreateSubnetGroupInput{
		SubnetGroupName: aws.String(d.Get("name").(string)),
		SubnetIds:       expandStringSet(d.Get("subnet_ids").(*schema.Set)),
	}
	if v, ok := d.GetOk("description"); ok {
		input.Description = aws.String(v.(string))
	}

	_, err := conn.CreateSubnetGroup(input)
	if err != nil {
		return err
	}

	d.SetId(d.Get("name").(string))
	return resourceAwsDaxSubnetGroupRead(d, meta)
}

func resourceAwsDaxSubnetGroupRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).daxconn

	resp, err := conn.DescribeSubnetGroups(&dax.DescribeSubnetGroupsInput{
		SubnetGroupNames: []*string{aws.String(d.Id())},
	})
	if err != nil {
		if isAWSErr(err, dax.ErrCodeSubnetGroupNotFoundFault, "") {
			log.Printf("[WARN] DAX SubnetGroup %q not found, removing from state", d.Id())
			d.SetId("")
			return nil
		}
		return err
	}
	sg := resp.SubnetGroups[0]

	d.Set("name", sg.SubnetGroupName)
	d.Set("description", sg.Description)
	subnetIDs := make([]*string, 0, len(sg.Subnets))
	for _, v := range sg.Subnets {
		subnetIDs = append(subnetIDs, v.SubnetIdentifier)
	}
	d.Set("subnet_ids", flattenStringList(subnetIDs))
	d.Set("vpc_id", sg.VpcId)
	return nil
}

func resourceAwsDaxSubnetGroupUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).daxconn

	input := &dax.UpdateSubnetGroupInput{
		SubnetGroupName: aws.String(d.Id()),
	}

	if d.HasChange("description") {
		input.Description = aws.String(d.Get("description").(string))
	}

	if d.HasChange("subnet_ids") {
		input.SubnetIds = expandStringSet(d.Get("subnet_ids").(*schema.Set))
	}

	_, err := conn.UpdateSubnetGroup(input)
	if err != nil {
		return err
	}

	return resourceAwsDaxSubnetGroupRead(d, meta)
}

func resourceAwsDaxSubnetGroupDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).daxconn

	input := &dax.DeleteSubnetGroupInput{
		SubnetGroupName: aws.String(d.Id()),
	}

	_, err := conn.DeleteSubnetGroup(input)
	if err != nil {
		if isAWSErr(err, dax.ErrCodeSubnetGroupNotFoundFault, "") {
			return nil
		}
		return err
	}

	return nil
}
