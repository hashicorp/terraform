package aws

import (
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/glue"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
)

func dataSourceAwsGlueScript() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceAwsGlueScriptRead,
		Schema: map[string]*schema.Schema{
			"dag_edge": {
				Type:     schema.TypeList,
				Required: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"source": {
							Type:     schema.TypeString,
							Required: true,
						},
						"target": {
							Type:     schema.TypeString,
							Required: true,
						},
						"target_parameter": {
							Type:     schema.TypeString,
							Optional: true,
						},
					},
				},
			},
			"dag_node": {
				Type:     schema.TypeList,
				Required: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"args": {
							Type:     schema.TypeList,
							Required: true,
							MinItems: 1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"name": {
										Type:     schema.TypeString,
										Required: true,
									},
									"param": {
										Type:     schema.TypeBool,
										Optional: true,
									},
									"value": {
										Type:     schema.TypeString,
										Required: true,
									},
								},
							},
						},
						"id": {
							Type:     schema.TypeString,
							Required: true,
						},
						"line_number": {
							Type:     schema.TypeInt,
							Optional: true,
						},
						"node_type": {
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},
			"language": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  glue.LanguagePython,
				ValidateFunc: validation.StringInSlice([]string{
					glue.LanguagePython,
					glue.LanguageScala,
				}, false),
			},
			"python_script": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"scala_code": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func dataSourceAwsGlueScriptRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).glueconn

	dagEdge := d.Get("dag_edge").([]interface{})
	dagNode := d.Get("dag_node").([]interface{})

	input := &glue.CreateScriptInput{
		DagEdges: expandGlueCodeGenEdges(dagEdge),
		DagNodes: expandGlueCodeGenNodes(dagNode),
	}

	if v, ok := d.GetOk("language"); ok && v.(string) != "" {
		input.Language = aws.String(v.(string))
	}

	log.Printf("[DEBUG] Creating Glue Script: %s", input)
	output, err := conn.CreateScript(input)
	if err != nil {
		return fmt.Errorf("error creating Glue script: %s", err)
	}

	if output == nil {
		return errors.New("script not created")
	}

	d.SetId(time.Now().UTC().String())
	d.Set("python_script", output.PythonScript)
	d.Set("scala_code", output.ScalaCode)

	return nil
}

func expandGlueCodeGenNodeArgs(l []interface{}) []*glue.CodeGenNodeArg {
	args := []*glue.CodeGenNodeArg{}

	for _, mRaw := range l {
		m := mRaw.(map[string]interface{})
		arg := &glue.CodeGenNodeArg{
			Name:  aws.String(m["name"].(string)),
			Param: aws.Bool(m["param"].(bool)),
			Value: aws.String(m["value"].(string)),
		}
		args = append(args, arg)
	}

	return args
}

func expandGlueCodeGenEdges(l []interface{}) []*glue.CodeGenEdge {
	edges := []*glue.CodeGenEdge{}

	for _, mRaw := range l {
		m := mRaw.(map[string]interface{})
		edge := &glue.CodeGenEdge{
			Source: aws.String(m["source"].(string)),
			Target: aws.String(m["target"].(string)),
		}
		if v, ok := m["target_parameter"]; ok && v.(string) != "" {
			edge.TargetParameter = aws.String(v.(string))
		}
		edges = append(edges, edge)
	}

	return edges
}

func expandGlueCodeGenNodes(l []interface{}) []*glue.CodeGenNode {
	nodes := []*glue.CodeGenNode{}

	for _, mRaw := range l {
		m := mRaw.(map[string]interface{})
		node := &glue.CodeGenNode{
			Args:     expandGlueCodeGenNodeArgs(m["args"].([]interface{})),
			Id:       aws.String(m["id"].(string)),
			NodeType: aws.String(m["node_type"].(string)),
		}
		if v, ok := m["line_number"]; ok && v.(int) != 0 {
			node.LineNumber = aws.Int64(int64(v.(int)))
		}
		nodes = append(nodes, node)
	}

	return nodes
}
