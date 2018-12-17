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

func resourceAwsAppmeshMesh() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsAppmeshMeshCreate,
		Read:   resourceAwsAppmeshMeshRead,
		Delete: resourceAwsAppmeshMeshDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringLenBetween(1, 255),
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

func resourceAwsAppmeshMeshCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).appmeshconn

	meshName := d.Get("name").(string)
	req := &appmesh.CreateMeshInput{
		MeshName: aws.String(meshName),
	}

	log.Printf("[DEBUG] Creating App Mesh service mesh: %#v", req)
	_, err := conn.CreateMesh(req)
	if err != nil {
		return fmt.Errorf("error creating App Mesh service mesh: %s", err)
	}

	d.SetId(meshName)

	return resourceAwsAppmeshMeshRead(d, meta)
}

func resourceAwsAppmeshMeshRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).appmeshconn

	resp, err := conn.DescribeMesh(&appmesh.DescribeMeshInput{
		MeshName: aws.String(d.Id()),
	})
	if err != nil {
		if isAWSErr(err, "NotFoundException", "") {
			log.Printf("[WARN] App Mesh service mesh (%s) not found, removing from state", d.Id())
			d.SetId("")
			return nil
		}
		return fmt.Errorf("error reading App Mesh service mesh: %s", err)
	}
	if aws.StringValue(resp.Mesh.Status.Status) == appmesh.MeshStatusCodeDeleted {
		log.Printf("[WARN] App Mesh service mesh (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	d.Set("name", resp.Mesh.MeshName)
	d.Set("arn", resp.Mesh.Metadata.Arn)
	d.Set("created_date", resp.Mesh.Metadata.CreatedAt.Format(time.RFC3339))
	d.Set("last_updated_date", resp.Mesh.Metadata.LastUpdatedAt.Format(time.RFC3339))

	return nil
}

func resourceAwsAppmeshMeshDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).appmeshconn

	log.Printf("[DEBUG] Deleting App Mesh service mesh: %s", d.Id())
	_, err := conn.DeleteMesh(&appmesh.DeleteMeshInput{
		MeshName: aws.String(d.Id()),
	})
	if err != nil {
		if isAWSErr(err, "NotFoundException", "") {
			return nil
		}
		return fmt.Errorf("error deleting App Mesh service mesh: %s", err)
	}

	return nil
}
