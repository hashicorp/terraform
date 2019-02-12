package aws

import (
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/appmesh"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
)

func resourceAwsAppmeshVirtualRouter() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsAppmeshVirtualRouterCreate,
		Read:   resourceAwsAppmeshVirtualRouterRead,
		Update: resourceAwsAppmeshVirtualRouterUpdate,
		Delete: resourceAwsAppmeshVirtualRouterDelete,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringLenBetween(1, 255),
			},

			"mesh_name": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringLenBetween(1, 255),
			},

			"spec": {
				Type:     schema.TypeList,
				Required: true,
				MinItems: 1,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"service_names": {
							Type:     schema.TypeSet,
							Required: true,
							MinItems: 1,
							MaxItems: 10,
							Elem:     &schema.Schema{Type: schema.TypeString},
							Set:      schema.HashString,
						},
					},
				},
			},

			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"created_date": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"last_updated_date": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceAwsAppmeshVirtualRouterCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).appmeshconn

	req := &appmesh.CreateVirtualRouterInput{
		MeshName:          aws.String(d.Get("mesh_name").(string)),
		VirtualRouterName: aws.String(d.Get("name").(string)),
		Spec:              expandAppmeshVirtualRouterSpec(d.Get("spec").([]interface{})),
	}

	log.Printf("[DEBUG] Creating App Mesh virtual router: %#v", req)
	resp, err := conn.CreateVirtualRouter(req)
	if err != nil {
		return fmt.Errorf("error creating App Mesh virtual router: %s", err)
	}

	d.SetId(aws.StringValue(resp.VirtualRouter.Metadata.Uid))

	return resourceAwsAppmeshVirtualRouterRead(d, meta)
}

func resourceAwsAppmeshVirtualRouterUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).appmeshconn

	if d.HasChange("spec") {
		_, v := d.GetChange("spec")
		req := &appmesh.UpdateVirtualRouterInput{
			MeshName:          aws.String(d.Get("mesh_name").(string)),
			VirtualRouterName: aws.String(d.Get("name").(string)),
			Spec:              expandAppmeshVirtualRouterSpec(v.([]interface{})),
		}

		log.Printf("[DEBUG] Updating App Mesh virtual router: %#v", req)
		_, err := conn.UpdateVirtualRouter(req)
		if err != nil {
			return fmt.Errorf("error updating App Mesh virtual router: %s", err)
		}
	}

	return resourceAwsAppmeshVirtualRouterRead(d, meta)
}

func resourceAwsAppmeshVirtualRouterRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).appmeshconn

	resp, err := conn.DescribeVirtualRouter(&appmesh.DescribeVirtualRouterInput{
		MeshName:          aws.String(d.Get("mesh_name").(string)),
		VirtualRouterName: aws.String(d.Get("name").(string)),
	})
	if err != nil {
		if isAWSErr(err, appmesh.ErrCodeNotFoundException, "") {
			log.Printf("[WARN] App Mesh virtual router (%s) not found, removing from state", d.Id())
			d.SetId("")
			return nil
		}
		return fmt.Errorf("error reading App Mesh virtual router: %s", err)
	}
	if aws.StringValue(resp.VirtualRouter.Status.Status) == appmesh.VirtualRouterStatusCodeDeleted {
		log.Printf("[WARN] App Mesh virtual router (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	d.Set("name", resp.VirtualRouter.VirtualRouterName)
	d.Set("mesh_name", resp.VirtualRouter.MeshName)
	d.Set("arn", resp.VirtualRouter.Metadata.Arn)
	d.Set("created_date", resp.VirtualRouter.Metadata.CreatedAt.Format(time.RFC3339))
	d.Set("last_updated_date", resp.VirtualRouter.Metadata.LastUpdatedAt.Format(time.RFC3339))
	err1 := d.Set("spec", flattenAppmeshVirtualRouterSpec(resp.VirtualRouter.Spec))
	return err1
}

func resourceAwsAppmeshVirtualRouterDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).appmeshconn

	log.Printf("[DEBUG] Deleting App Mesh virtual router: %s", d.Id())
	_, err := conn.DeleteVirtualRouter(&appmesh.DeleteVirtualRouterInput{
		MeshName:          aws.String(d.Get("mesh_name").(string)),
		VirtualRouterName: aws.String(d.Get("name").(string)),
	})
	if err != nil {
		if isAWSErr(err, appmesh.ErrCodeNotFoundException, "") {
			return nil
		}
		return fmt.Errorf("error deleting App Mesh virtual router: %s", err)
	}

	return nil
}
