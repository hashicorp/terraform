package aws

import (
	"bytes"
	"fmt"
	"log"
	"regexp"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/appmesh"
	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
)

func resourceAwsAppmeshRoute() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsAppmeshRouteCreate,
		Read:   resourceAwsAppmeshRouteRead,
		Update: resourceAwsAppmeshRouteUpdate,
		Delete: resourceAwsAppmeshRouteDelete,

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

			"virtual_router_name": {
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
						"http_route": {
							Type:     schema.TypeList,
							Optional: true,
							MinItems: 0,
							MaxItems: 1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"action": {
										Type:     schema.TypeList,
										Required: true,
										MinItems: 1,
										MaxItems: 1,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"weighted_target": {
													Type:     schema.TypeSet,
													Required: true,
													MinItems: 1,
													Elem: &schema.Resource{
														Schema: map[string]*schema.Schema{
															"virtual_node": {
																Type:         schema.TypeString,
																Required:     true,
																ValidateFunc: validation.StringLenBetween(1, 255),
															},
															"weight": {
																Type:         schema.TypeInt,
																Required:     true,
																ValidateFunc: validation.IntBetween(0, 100),
															},
														},
													},
													Set: appmeshRouteWeightedTargetHash,
												},
											},
										},
									},

									"match": {
										Type:     schema.TypeList,
										Required: true,
										MinItems: 1,
										MaxItems: 1,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"prefix": {
													Type:         schema.TypeString,
													Required:     true,
													ValidateFunc: validation.StringMatch(regexp.MustCompile(`^/`), "must start with /"),
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

func resourceAwsAppmeshRouteCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).appmeshconn

	req := &appmesh.CreateRouteInput{
		MeshName:          aws.String(d.Get("mesh_name").(string)),
		RouteName:         aws.String(d.Get("name").(string)),
		VirtualRouterName: aws.String(d.Get("virtual_router_name").(string)),
		Spec:              expandAppmeshRouteSpec(d.Get("spec").([]interface{})),
	}

	log.Printf("[DEBUG] Creating App Mesh route: %#v", req)
	resp, err := conn.CreateRoute(req)
	if err != nil {
		return fmt.Errorf("error creating App Mesh route: %s", err)
	}

	d.SetId(aws.StringValue(resp.Route.Metadata.Uid))

	return resourceAwsAppmeshRouteRead(d, meta)
}

func resourceAwsAppmeshRouteRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).appmeshconn

	resp, err := conn.DescribeRoute(&appmesh.DescribeRouteInput{
		MeshName:          aws.String(d.Get("mesh_name").(string)),
		RouteName:         aws.String(d.Get("name").(string)),
		VirtualRouterName: aws.String(d.Get("virtual_router_name").(string)),
	})
	if err != nil {
		if isAWSErr(err, appmesh.ErrCodeNotFoundException, "") {
			log.Printf("[WARN] App Mesh route (%s) not found, removing from state", d.Id())
			d.SetId("")
			return nil
		}
		return fmt.Errorf("error reading App Mesh route: %s", err)
	}
	if aws.StringValue(resp.Route.Status.Status) == appmesh.RouteStatusCodeDeleted {
		log.Printf("[WARN] App Mesh route (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	d.Set("name", resp.Route.RouteName)
	d.Set("mesh_name", resp.Route.MeshName)
	d.Set("virtual_router_name", resp.Route.VirtualRouterName)
	d.Set("arn", resp.Route.Metadata.Arn)
	d.Set("created_date", resp.Route.Metadata.CreatedAt.Format(time.RFC3339))
	d.Set("last_updated_date", resp.Route.Metadata.LastUpdatedAt.Format(time.RFC3339))
	if err := d.Set("spec", flattenAppmeshRouteSpec(resp.Route.Spec)); err != nil {
		return err
	}

	return nil
}

func resourceAwsAppmeshRouteUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).appmeshconn

	if d.HasChange("spec") {
		_, v := d.GetChange("spec")
		req := &appmesh.UpdateRouteInput{
			MeshName:          aws.String(d.Get("mesh_name").(string)),
			RouteName:         aws.String(d.Get("name").(string)),
			VirtualRouterName: aws.String(d.Get("virtual_router_name").(string)),
			Spec:              expandAppmeshRouteSpec(v.([]interface{})),
		}

		log.Printf("[DEBUG] Updating App Mesh route: %#v", req)
		_, err := conn.UpdateRoute(req)
		if err != nil {
			return fmt.Errorf("error updating App Mesh route: %s", err)
		}
	}

	return resourceAwsAppmeshRouteRead(d, meta)
}

func resourceAwsAppmeshRouteDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).appmeshconn

	log.Printf("[DEBUG] Deleting App Mesh route: %s", d.Id())
	_, err := conn.DeleteRoute(&appmesh.DeleteRouteInput{
		MeshName:          aws.String(d.Get("mesh_name").(string)),
		RouteName:         aws.String(d.Get("name").(string)),
		VirtualRouterName: aws.String(d.Get("virtual_router_name").(string)),
	})
	if err != nil {
		if isAWSErr(err, appmesh.ErrCodeNotFoundException, "") {
			return nil
		}
		return fmt.Errorf("error deleting App Mesh route: %s", err)
	}

	return nil
}

func appmeshRouteWeightedTargetHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})
	if v, ok := m["virtual_node"].(string); ok {
		buf.WriteString(fmt.Sprintf("%s-", v))
	}
	if v, ok := m["weight"].(int); ok {
		buf.WriteString(fmt.Sprintf("%d-", v))
	}
	return hashcode.String(buf.String())
}
