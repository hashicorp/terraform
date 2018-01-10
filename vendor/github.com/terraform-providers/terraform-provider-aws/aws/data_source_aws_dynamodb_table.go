package aws

import (
	"bytes"
	"fmt"
	"log"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourceAwsDynamoDbTable() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceAwsDynamoDbTableRead,
		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},

			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"attribute": {
				Type:     schema.TypeSet,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"type": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
				Set: func(v interface{}) int {
					var buf bytes.Buffer
					m := v.(map[string]interface{})
					buf.WriteString(fmt.Sprintf("%s-", m["name"].(string)))
					return hashcode.String(buf.String())
				},
			},
			"global_secondary_index": {
				Type:     schema.TypeSet,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"write_capacity": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"read_capacity": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"hash_key": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"range_key": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"projection_type": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"non_key_attributes": {
							Type:     schema.TypeList,
							Computed: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},
					},
				},
			},
			"hash_key": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"local_secondary_index": {
				Type:     schema.TypeSet,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"range_key": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"projection_type": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"non_key_attributes": {
							Type:     schema.TypeList,
							Computed: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},
					},
				},
				Set: func(v interface{}) int {
					var buf bytes.Buffer
					m := v.(map[string]interface{})
					buf.WriteString(fmt.Sprintf("%s-", m["name"].(string)))
					return hashcode.String(buf.String())
				},
			},
			"range_key": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"read_capacity": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"stream_arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"stream_enabled": {
				Type:     schema.TypeBool,
				Computed: true,
			},
			"stream_label": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"stream_view_type": {
				Type:     schema.TypeString,
				Computed: true,
				StateFunc: func(v interface{}) string {
					value := v.(string)
					return strings.ToUpper(value)
				},
			},
			"tags": tagsSchemaComputed(),
			"ttl": {
				Type:     schema.TypeSet,
				Computed: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"attribute_name": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"enabled": {
							Type:     schema.TypeBool,
							Computed: true,
						},
					},
				},
			},
			"write_capacity": {
				Type:     schema.TypeInt,
				Computed: true,
			},
		},
	}
}

func dataSourceAwsDynamoDbTableRead(d *schema.ResourceData, meta interface{}) error {
	dynamodbconn := meta.(*AWSClient).dynamodbconn

	name := d.Get("name").(string)
	log.Printf("[DEBUG] Loading data for DynamoDB table %q", name)
	req := &dynamodb.DescribeTableInput{
		TableName: aws.String(name),
	}

	result, err := dynamodbconn.DescribeTable(req)

	if err != nil {
		return errwrap.Wrapf("Error retrieving DynamoDB table: {{err}}", err)
	}

	d.SetId(*result.Table.TableName)

	return flattenAwsDynamoDbTableResource(d, meta, result.Table)
}
