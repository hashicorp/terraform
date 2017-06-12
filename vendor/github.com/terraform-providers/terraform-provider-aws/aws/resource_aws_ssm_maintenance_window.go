package aws

import (
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsSsmMaintenanceWindow() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsSsmMaintenanceWindowCreate,
		Read:   resourceAwsSsmMaintenanceWindowRead,
		Update: resourceAwsSsmMaintenanceWindowUpdate,
		Delete: resourceAwsSsmMaintenanceWindowDelete,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},

			"schedule": {
				Type:     schema.TypeString,
				Required: true,
			},

			"duration": {
				Type:     schema.TypeInt,
				Required: true,
			},

			"cutoff": {
				Type:     schema.TypeInt,
				Required: true,
			},

			"allow_unassociated_targets": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},

			"enabled": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},
		},
	}
}

func resourceAwsSsmMaintenanceWindowCreate(d *schema.ResourceData, meta interface{}) error {
	ssmconn := meta.(*AWSClient).ssmconn

	params := &ssm.CreateMaintenanceWindowInput{
		Name:     aws.String(d.Get("name").(string)),
		Schedule: aws.String(d.Get("schedule").(string)),
		Duration: aws.Int64(int64(d.Get("duration").(int))),
		Cutoff:   aws.Int64(int64(d.Get("cutoff").(int))),
		AllowUnassociatedTargets: aws.Bool(d.Get("allow_unassociated_targets").(bool)),
	}

	resp, err := ssmconn.CreateMaintenanceWindow(params)
	if err != nil {
		return err
	}

	d.SetId(*resp.WindowId)
	return resourceAwsSsmMaintenanceWindowRead(d, meta)
}

func resourceAwsSsmMaintenanceWindowUpdate(d *schema.ResourceData, meta interface{}) error {
	ssmconn := meta.(*AWSClient).ssmconn

	params := &ssm.UpdateMaintenanceWindowInput{
		WindowId: aws.String(d.Id()),
	}

	if d.HasChange("name") {
		params.Name = aws.String(d.Get("name").(string))
	}

	if d.HasChange("schedule") {
		params.Schedule = aws.String(d.Get("schedule").(string))
	}

	if d.HasChange("duration") {
		params.Duration = aws.Int64(int64(d.Get("duration").(int)))
	}

	if d.HasChange("cutoff") {
		params.Cutoff = aws.Int64(int64(d.Get("cutoff").(int)))
	}

	if d.HasChange("allow_unassociated_targets") {
		params.AllowUnassociatedTargets = aws.Bool(d.Get("allow_unassociated_targets").(bool))
	}

	if d.HasChange("enabled") {
		params.Enabled = aws.Bool(d.Get("enabled").(bool))
	}

	_, err := ssmconn.UpdateMaintenanceWindow(params)
	if err != nil {
		return err
	}

	return resourceAwsSsmMaintenanceWindowRead(d, meta)
}

func resourceAwsSsmMaintenanceWindowRead(d *schema.ResourceData, meta interface{}) error {
	ssmconn := meta.(*AWSClient).ssmconn

	params := &ssm.GetMaintenanceWindowInput{
		WindowId: aws.String(d.Id()),
	}

	resp, err := ssmconn.GetMaintenanceWindow(params)
	if err != nil {
		return err
	}

	d.Set("name", resp.Name)
	d.Set("cutoff", resp.Cutoff)
	d.Set("duration", resp.Duration)
	d.Set("enabled", resp.Enabled)
	d.Set("allow_unassociated_targets", resp.AllowUnassociatedTargets)
	d.Set("schedule", resp.Schedule)

	return nil
}

func resourceAwsSsmMaintenanceWindowDelete(d *schema.ResourceData, meta interface{}) error {
	ssmconn := meta.(*AWSClient).ssmconn

	log.Printf("[INFO] Deleting SSM Maintenance Window: %s", d.Id())

	params := &ssm.DeleteMaintenanceWindowInput{
		WindowId: aws.String(d.Id()),
	}

	_, err := ssmconn.DeleteMaintenanceWindow(params)
	if err != nil {
		return err
	}

	return nil
}
