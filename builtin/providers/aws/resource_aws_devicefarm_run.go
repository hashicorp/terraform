package aws

import "github.com/hashicorp/terraform/helper/schema"

func resourceAwsDeviceFarmRun() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsDeviceFarmRunCreate,
		Read:   resourceAwsDeviceFarmRunRead,
		Update: resourceAwsDeviceFarmRunUpdate,
		Delete: resourceAwsDeviceFarmRunDelete,

		Schema: map[string]*schema.Schema{
			"app_arn": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"project_arn": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"device_pool_arn": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"app_name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"run_name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"app_content_type": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"app_upload_type": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"app_metadata": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceAwsDeviceFarmRunCreate(d *schema.ResourceData, meta interface{}) error {
	//	conn := meta.(*AWSClient).devicefarmconn
	//	region := meta.(*AWSClient).region
	//
	//	//	We need to ensure that DeviceFarm is only being run against us-west-2
	//	//	As this is the only place that AWS currently supports it
	//	if region != "us-west-2" {
	//		return fmt.Errorf("DeviceFarm can only be used with us-west-2. You are trying to use it on %s", region)
	//	}
	//
	//	input := &devicefarm.CreateUploadInput{
	//		Name:       aws.String(d.Get("app_name").(string)),
	//		ProjectArn: aws.String(d.Get("project_arn").(string)),
	//		Type:       aws.String(d.Get("app_upload_type").(string)),
	//	}
	//	if v, ok := d.GetOk("app_content_type"); ok {
	//		input.ContentType = aws.String(v.(string))
	//	}
	//
	//	log.Printf("[DEBUG] Creating DeviceFarm Upload: %s", d.Get("app_name").(string))
	//	out, err := conn.CreateUpload(input)
	//	if err != nil {
	//		return fmt.Errorf("Error creating DeviceFarm Upload: %s", err)
	//	}
	//
	//	log.Printf("[DEBUG] Successsfully Created DeviceFarm Upload: %s", *out.Upload.Arn)
	//	d.SetId(*out.Upload.Arn)
	//
	//	runInput := &devicefarm.ScheduleRunInput{
	//		Name:          aws.String(d.Get("run_name").(string)),
	//		AppArn:        aws.String(d.Id()),
	//		DevicePoolArn: aws.String(d.Get("device_pool_arn").(string)),
	//	}
	//
	//	log.Printf("[DEBUG] Scheduling DeviceFarm Run: %s", d.Get("run_name").(string))
	//	_, err := conn.ScheduleRun(input)
	//	if err != nil {
	//		return fmt.Errorf("Error Scheduling DeviceFarm Run: %s", err)
	//	}

	return resourceAwsDeviceFarmRunRead(d, meta)
}

func resourceAwsDeviceFarmRunRead(d *schema.ResourceData, meta interface{}) error {
	//	conn := meta.(*AWSClient).devicefarmconn
	//
	//	input := &devicefarm.GetUploadInput{
	//		Arn: aws.String(d.Id()),
	//	}
	//
	//	log.Printf("[DEBUG] Reading DeviceFarm Project: %s", d.Id())
	//	out, err := conn.GetUpload(input)
	//	if err != nil {
	//		return fmt.Errorf("Error reading DeviceFarm Project: %s", err)
	//	}
	//
	//	d.Set("app_name", out.Upload.Name)
	//	d.Set("app_arn", out.Upload.Arn)
	//	d.Set("app_content_type", out.Upload.ContentType)
	//	d.Set("app_metadata", out.Upload.Metadata)

	return nil
}

func resourceAwsDeviceFarmRunUpdate(d *schema.ResourceData, meta interface{}) error {
	//	conn := meta.(*AWSClient).devicefarmconn
	//
	//	if d.HasChange("name") {
	//		input := &devicefarm.UpdateProjectInput{
	//			Arn:  aws.String(d.Id()),
	//			Name: aws.String(d.Get("name").(string)),
	//		}
	//
	//		log.Printf("[DEBUG] Updating DeviceFarm Project: %s", d.Id())
	//		_, err := conn.UpdateProject(input)
	//		if err != nil {
	//			return fmt.Errorf("Error Updating DeviceFarm Project: %s", err)
	//		}
	//
	//	}

	return resourceAwsDeviceFarmRunRead(d, meta)
}

func resourceAwsDeviceFarmRunDelete(d *schema.ResourceData, meta interface{}) error {
	/*conn := meta.(*AWSClient).devicefarmconn

	input := &devicefarm.DeleteRunInput{
		Arn: aws.String(d.Id()),
	}

	log.Printf("[DEBUG] Deleting DeviceFarm Run: %s", d.Id())
	_, err := conn.DeleteProject(input)
	if err != nil {
		return fmt.Errorf("Error deleting DeviceFarm Run: %s", err)
	}*/

	return nil
}
