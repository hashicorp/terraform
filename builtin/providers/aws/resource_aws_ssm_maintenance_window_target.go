package aws

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsSsmMaintenanceWindowTarget() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsSsmMaintenanceWindowTargetCreate,
		Read:   resourceAwsSsmMaintenanceWindowTargetRead,
		Delete: resourceAwsSsmMaintenanceWindowTargetDelete,

		Schema: map[string]*schema.Schema{
			"window_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"resource_type": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"targets": {
				Type:     schema.TypeList,
				Required: true,
				ForceNew: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"key": {
							Type:     schema.TypeString,
							Required: true,
						},
						"values": {
							Type:     schema.TypeList,
							Required: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},
					},
				},
			},

			"owner_information": {
				Type:     schema.TypeString,
				ForceNew: true,
				Optional: true,
			},
		},
	}
}

func resourceAwsSsmMaintenanceWindowTargetCreate(d *schema.ResourceData, meta interface{}) error {
	ssmconn := meta.(*AWSClient).ssmconn

	log.Printf("[INFO] Registering SSM Maintenance Window Target")

	params := &ssm.RegisterTargetWithMaintenanceWindowInput{
		WindowId:     aws.String(d.Get("window_id").(string)),
		ResourceType: aws.String(d.Get("resource_type").(string)),
		Targets:      expandAwsSsmTargets(d),
	}

	if v, ok := d.GetOk("owner_information"); ok {
		params.OwnerInformation = aws.String(v.(string))
	}

	resp, err := ssmconn.RegisterTargetWithMaintenanceWindow(params)
	if err != nil {
		return err
	}

	d.SetId(*resp.WindowTargetId)

	return resourceAwsSsmMaintenanceWindowTargetRead(d, meta)
}

func resourceAwsSsmMaintenanceWindowTargetRead(d *schema.ResourceData, meta interface{}) error {
	ssmconn := meta.(*AWSClient).ssmconn

	params := &ssm.DescribeMaintenanceWindowTargetsInput{
		WindowId: aws.String(d.Get("window_id").(string)),
		Filters: []*ssm.MaintenanceWindowFilter{
			{
				Key:    aws.String("WindowTargetId"),
				Values: []*string{aws.String(d.Id())},
			},
		},
	}

	resp, err := ssmconn.DescribeMaintenanceWindowTargets(params)
	if err != nil {
		return err
	}

	found := false
	for _, t := range resp.Targets {
		if *t.WindowTargetId == d.Id() {
			found = true

			d.Set("owner_information", t.OwnerInformation)
			d.Set("window_id", t.WindowId)
			d.Set("resource_type", t.ResourceType)

			if err := d.Set("targets", flattenAwsSsmTargets(t.Targets)); err != nil {
				return fmt.Errorf("[DEBUG] Error setting targets error: %#v", err)
			}
		}
	}

	if !found {
		log.Printf("[INFO] Maintenance Window Target not found. Removing from state")
		d.SetId("")
		return nil
	}

	return nil
}

func resourceAwsSsmMaintenanceWindowTargetDelete(d *schema.ResourceData, meta interface{}) error {
	ssmconn := meta.(*AWSClient).ssmconn

	log.Printf("[INFO] Deregistering SSM Maintenance Window Target: %s", d.Id())

	params := &ssm.DeregisterTargetFromMaintenanceWindowInput{
		WindowId:       aws.String(d.Get("window_id").(string)),
		WindowTargetId: aws.String(d.Id()),
	}

	_, err := ssmconn.DeregisterTargetFromMaintenanceWindow(params)
	if err != nil {
		return err
	}

	return nil
}
