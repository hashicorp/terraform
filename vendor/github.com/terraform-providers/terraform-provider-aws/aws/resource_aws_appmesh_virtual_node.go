package aws

import (
	"bytes"
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/appmesh"
	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
)

func resourceAwsAppmeshVirtualNode() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsAppmeshVirtualNodeCreate,
		Read:   resourceAwsAppmeshVirtualNodeRead,
		Update: resourceAwsAppmeshVirtualNodeUpdate,
		Delete: resourceAwsAppmeshVirtualNodeDelete,

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
						"backends": {
							Type:     schema.TypeSet,
							Optional: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
							Set:      schema.HashString,
						},

						"listener": {
							Type:     schema.TypeSet,
							Optional: true,
							MinItems: 0,
							MaxItems: 1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"port_mapping": {
										Type:     schema.TypeList,
										Required: true,
										MinItems: 1,
										MaxItems: 1,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"port": {
													Type:         schema.TypeInt,
													Required:     true,
													ValidateFunc: validation.IntBetween(1, 65535),
												},

												"protocol": {
													Type:     schema.TypeString,
													Required: true,
													ValidateFunc: validation.StringInSlice([]string{
														appmesh.PortProtocolHttp,
														appmesh.PortProtocolTcp,
													}, false),
												},
											},
										},
									},
								},
							},
							Set: appmeshVirtualNodeListenerHash,
						},

						"service_discovery": {
							Type:     schema.TypeList,
							Optional: true,
							MinItems: 0,
							MaxItems: 1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"dns": {
										Type:     schema.TypeList,
										Required: true,
										MinItems: 1,
										MaxItems: 1,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"service_name": {
													Type:         schema.TypeString,
													Required:     true,
													ValidateFunc: validation.NoZeroValues,
												},
											},
										},
									},
								},
							},
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

func resourceAwsAppmeshVirtualNodeCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).appmeshconn

	req := &appmesh.CreateVirtualNodeInput{
		MeshName:        aws.String(d.Get("mesh_name").(string)),
		VirtualNodeName: aws.String(d.Get("name").(string)),
		Spec:            expandAppmeshVirtualNodeSpec(d.Get("spec").([]interface{})),
	}

	log.Printf("[DEBUG] Creating App Mesh virtual node: %#v", req)
	resp, err := conn.CreateVirtualNode(req)
	if err != nil {
		return fmt.Errorf("error creating App Mesh virtual node: %s", err)
	}

	d.SetId(aws.StringValue(resp.VirtualNode.Metadata.Uid))

	return resourceAwsAppmeshVirtualNodeRead(d, meta)
}

func resourceAwsAppmeshVirtualNodeRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).appmeshconn

	resp, err := conn.DescribeVirtualNode(&appmesh.DescribeVirtualNodeInput{
		MeshName:        aws.String(d.Get("mesh_name").(string)),
		VirtualNodeName: aws.String(d.Get("name").(string)),
	})
	if err != nil {
		if isAWSErr(err, appmesh.ErrCodeNotFoundException, "") {
			log.Printf("[WARN] App Mesh virtual node (%s) not found, removing from state", d.Id())
			d.SetId("")
			return nil
		}
		return fmt.Errorf("error reading App Mesh virtual node: %s", err)
	}
	if aws.StringValue(resp.VirtualNode.Status.Status) == appmesh.VirtualNodeStatusCodeDeleted {
		log.Printf("[WARN] App Mesh virtual node (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	d.Set("name", resp.VirtualNode.VirtualNodeName)
	d.Set("mesh_name", resp.VirtualNode.MeshName)
	d.Set("arn", resp.VirtualNode.Metadata.Arn)
	d.Set("created_date", resp.VirtualNode.Metadata.CreatedAt.Format(time.RFC3339))
	d.Set("last_updated_date", resp.VirtualNode.Metadata.LastUpdatedAt.Format(time.RFC3339))
	err1 := d.Set("spec", flattenAppmeshVirtualNodeSpec(resp.VirtualNode.Spec))
	return err1
}

func resourceAwsAppmeshVirtualNodeUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).appmeshconn

	if d.HasChange("spec") {
		_, v := d.GetChange("spec")
		req := &appmesh.UpdateVirtualNodeInput{
			MeshName:        aws.String(d.Get("mesh_name").(string)),
			VirtualNodeName: aws.String(d.Get("name").(string)),
			Spec:            expandAppmeshVirtualNodeSpec(v.([]interface{})),
		}

		log.Printf("[DEBUG] Updating App Mesh virtual node: %#v", req)
		_, err := conn.UpdateVirtualNode(req)
		if err != nil {
			return fmt.Errorf("error updating App Mesh virtual node: %s", err)
		}
	}

	return resourceAwsAppmeshVirtualNodeRead(d, meta)
}

func resourceAwsAppmeshVirtualNodeDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).appmeshconn

	log.Printf("[DEBUG] Deleting App Mesh virtual node: %s", d.Id())
	_, err := conn.DeleteVirtualNode(&appmesh.DeleteVirtualNodeInput{
		MeshName:        aws.String(d.Get("mesh_name").(string)),
		VirtualNodeName: aws.String(d.Get("name").(string)),
	})
	if err != nil {
		if isAWSErr(err, appmesh.ErrCodeNotFoundException, "") {
			return nil
		}
		return fmt.Errorf("error deleting App Mesh virtual node: %s", err)
	}

	return nil
}

func appmeshVirtualNodeListenerHash(vListener interface{}) int {
	var buf bytes.Buffer
	mListener := vListener.(map[string]interface{})
	if vPortMapping, ok := mListener["port_mapping"].([]interface{}); ok && len(vPortMapping) > 0 && vPortMapping[0] != nil {
		mPortMapping := vPortMapping[0].(map[string]interface{})
		if v, ok := mPortMapping["port"].(int); ok {
			buf.WriteString(fmt.Sprintf("%d-", v))
		}
		if v, ok := mPortMapping["protocol"].(string); ok {
			buf.WriteString(fmt.Sprintf("%s-", v))
		}
	}
	return hashcode.String(buf.String())
}
