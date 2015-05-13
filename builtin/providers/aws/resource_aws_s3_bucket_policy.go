package main

import (
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsS3BucketPolicy() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsS3BucketPolicyCreate,
		Read:   resourceAwsS3BucketPolicyRead,
		Update: resourceAwsS3BucketPolicyUpdate,
		Delete: resourceAwsS3BucketPolicyDelete,

		Schema: map[string]*schema.Schema{
			"address": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
		},
	}
}

func resourceAwsS3BucketPolicyCreate(d *schema.ResourceData, m interface{}) error {
	return nil
}

func resourceAwsS3BucketPolicyRead(d *schema.ResourceData, m interface{}) error {
	return nil
}

func resourceAwsS3BucketPolicyUpdate(d *schema.ResourceData, m interface{}) error {
	return nil
}

func resourceAwsS3BucketPolicyDelete(d *schema.ResourceData, m interface{}) error {
	return nil
}
