package aws

import (
	"fmt"
	"log"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/glue"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsGlueCatalogTable() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsGlueCatalogTableCreate,
		Read:   resourceAwsGlueCatalogTableRead,
		Update: resourceAwsGlueCatalogTableUpdate,
		Delete: resourceAwsGlueCatalogTableDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"catalog_id": {
				Type:     schema.TypeString,
				ForceNew: true,
				Optional: true,
				Computed: true,
			},
			"database_name": {
				Type:     schema.TypeString,
				ForceNew: true,
				Required: true,
			},
			"description": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"name": {
				Type:     schema.TypeString,
				ForceNew: true,
				Required: true,
			},
			"owner": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"parameters": {
				Type:     schema.TypeMap,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"partition_keys": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"comment": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"name": {
							Type:     schema.TypeString,
							Required: true,
						},
						"type": {
							Type:     schema.TypeString,
							Optional: true,
						},
					},
				},
			},
			"retention": {
				Type:     schema.TypeInt,
				Optional: true,
			},
			"storage_descriptor": {
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"bucket_columns": {
							Type:     schema.TypeList,
							Optional: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},
						"columns": {
							Type:     schema.TypeList,
							Optional: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"comment": {
										Type:     schema.TypeString,
										Optional: true,
									},
									"name": {
										Type:     schema.TypeString,
										Required: true,
									},
									"type": {
										Type:     schema.TypeString,
										Optional: true,
									},
								},
							},
						},
						"compressed": {
							Type:     schema.TypeBool,
							Optional: true,
						},
						"input_format": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"location": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"number_of_buckets": {
							Type:     schema.TypeInt,
							Optional: true,
						},
						"output_format": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"parameters": {
							Type:     schema.TypeMap,
							Optional: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},
						"ser_de_info": {
							Type:     schema.TypeList,
							Optional: true,
							MaxItems: 1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"name": {
										Type:     schema.TypeString,
										Optional: true,
									},
									"parameters": {
										Type:     schema.TypeMap,
										Optional: true,
										Elem:     &schema.Schema{Type: schema.TypeString},
									},
									"serialization_library": {
										Type:     schema.TypeString,
										Optional: true,
									},
								},
							},
						},
						"skewed_info": {
							Type:     schema.TypeList,
							Optional: true,
							MaxItems: 1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"skewed_column_names": {
										Type:     schema.TypeList,
										Optional: true,
										Elem:     &schema.Schema{Type: schema.TypeString},
									},
									"skewed_column_values": {
										Type:     schema.TypeList,
										Optional: true,
										Elem:     &schema.Schema{Type: schema.TypeString},
									},
									"skewed_column_value_location_maps": {
										Type:     schema.TypeMap,
										Optional: true,
										Elem:     &schema.Schema{Type: schema.TypeString},
									},
								},
							},
						},
						"sort_columns": {
							Type:     schema.TypeList,
							Optional: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"column": {
										Type:     schema.TypeString,
										Required: true,
									},
									"sort_order": {
										Type:     schema.TypeInt,
										Required: true,
									},
								},
							},
						},
						"stored_as_sub_directories": {
							Type:     schema.TypeBool,
							Optional: true,
						},
					},
				},
			},
			"table_type": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"view_original_text": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"view_expanded_text": {
				Type:     schema.TypeString,
				Optional: true,
			},
		},
	}
}

func readAwsGlueTableID(id string) (catalogID string, dbName string, name string, error error) {
	idParts := strings.Split(id, ":")
	if len(idParts) != 3 {
		return "", "", "", fmt.Errorf("expected ID in format catalog-id:database-name:table-name, received: %s", id)
	}
	return idParts[0], idParts[1], idParts[2], nil
}

func resourceAwsGlueCatalogTableCreate(d *schema.ResourceData, meta interface{}) error {
	glueconn := meta.(*AWSClient).glueconn
	catalogID := createAwsGlueCatalogID(d, meta.(*AWSClient).accountid)
	dbName := d.Get("database_name").(string)
	name := d.Get("name").(string)

	input := &glue.CreateTableInput{
		CatalogId:    aws.String(catalogID),
		DatabaseName: aws.String(dbName),
		TableInput:   expandGlueTableInput(d),
	}

	_, err := glueconn.CreateTable(input)
	if err != nil {
		return fmt.Errorf("Error creating Catalog Table: %s", err)
	}

	d.SetId(fmt.Sprintf("%s:%s:%s", catalogID, dbName, name))

	return resourceAwsGlueCatalogTableRead(d, meta)
}

func resourceAwsGlueCatalogTableRead(d *schema.ResourceData, meta interface{}) error {
	glueconn := meta.(*AWSClient).glueconn

	catalogID, dbName, name, err := readAwsGlueTableID(d.Id())
	if err != nil {
		return err
	}

	input := &glue.GetTableInput{
		CatalogId:    aws.String(catalogID),
		DatabaseName: aws.String(dbName),
		Name:         aws.String(name),
	}

	out, err := glueconn.GetTable(input)
	if err != nil {

		if isAWSErr(err, glue.ErrCodeEntityNotFoundException, "") {
			log.Printf("[WARN] Glue Catalog Table (%s) not found, removing from state", d.Id())
			d.SetId("")
		}

		return fmt.Errorf("Error reading Glue Catalog Table: %s", err)
	}

	d.Set("name", out.Table.Name)
	d.Set("catalog_id", catalogID)
	d.Set("database_name", dbName)
	d.Set("description", out.Table.Description)
	d.Set("owner", out.Table.Owner)
	d.Set("retention", out.Table.Retention)

	if err := d.Set("storage_descriptor", flattenGlueStorageDescriptor(out.Table.StorageDescriptor)); err != nil {
		return fmt.Errorf("error setting storage_descriptor: %s", err)
	}

	if err := d.Set("partition_keys", flattenGlueColumns(out.Table.PartitionKeys)); err != nil {
		return fmt.Errorf("error setting partition_keys: %s", err)
	}

	d.Set("view_original_text", out.Table.ViewOriginalText)
	d.Set("view_expanded_text", out.Table.ViewExpandedText)
	d.Set("table_type", out.Table.TableType)

	if err := d.Set("parameters", aws.StringValueMap(out.Table.Parameters)); err != nil {
		return fmt.Errorf("error setting parameters: %s", err)
	}

	return nil
}

func resourceAwsGlueCatalogTableUpdate(d *schema.ResourceData, meta interface{}) error {
	glueconn := meta.(*AWSClient).glueconn

	catalogID, dbName, _, err := readAwsGlueTableID(d.Id())
	if err != nil {
		return err
	}

	updateTableInput := &glue.UpdateTableInput{
		CatalogId:    aws.String(catalogID),
		DatabaseName: aws.String(dbName),
		TableInput:   expandGlueTableInput(d),
	}

	if _, err := glueconn.UpdateTable(updateTableInput); err != nil {
		return fmt.Errorf("Error updating Glue Catalog Table: %s", err)
	}

	return resourceAwsGlueCatalogTableRead(d, meta)
}

func resourceAwsGlueCatalogTableDelete(d *schema.ResourceData, meta interface{}) error {
	glueconn := meta.(*AWSClient).glueconn

	catalogID, dbName, name, tableIdErr := readAwsGlueTableID(d.Id())
	if tableIdErr != nil {
		return tableIdErr
	}

	log.Printf("[DEBUG] Glue Catalog Table: %s:%s:%s", catalogID, dbName, name)
	_, err := glueconn.DeleteTable(&glue.DeleteTableInput{
		CatalogId:    aws.String(catalogID),
		Name:         aws.String(name),
		DatabaseName: aws.String(dbName),
	})
	if err != nil {
		return fmt.Errorf("Error deleting Glue Catalog Table: %s", err.Error())
	}
	return nil
}

func expandGlueTableInput(d *schema.ResourceData) *glue.TableInput {
	tableInput := &glue.TableInput{
		Name: aws.String(d.Get("name").(string)),
	}

	if v, ok := d.GetOk("description"); ok {
		tableInput.Description = aws.String(v.(string))
	}

	if v, ok := d.GetOk("owner"); ok {
		tableInput.Owner = aws.String(v.(string))
	}

	if v, ok := d.GetOk("retention"); ok {
		tableInput.Retention = aws.Int64(int64(v.(int)))
	}

	if v, ok := d.GetOk("storage_descriptor"); ok {
		tableInput.StorageDescriptor = expandGlueStorageDescriptor(v.([]interface{}))
	}

	if v, ok := d.GetOk("partition_keys"); ok {
		columns := expandGlueColumns(v.([]interface{}))
		tableInput.PartitionKeys = columns
	}

	if v, ok := d.GetOk("view_original_text"); ok {
		tableInput.ViewOriginalText = aws.String(v.(string))
	}

	if v, ok := d.GetOk("view_expanded_text"); ok {
		tableInput.ViewExpandedText = aws.String(v.(string))
	}

	if v, ok := d.GetOk("table_type"); ok {
		tableInput.TableType = aws.String(v.(string))
	}

	if v, ok := d.GetOk("parameters"); ok {
		paramsMap := map[string]string{}
		for key, value := range v.(map[string]interface{}) {
			paramsMap[key] = value.(string)
		}
		tableInput.Parameters = aws.StringMap(paramsMap)
	}

	return tableInput
}

func expandGlueStorageDescriptor(l []interface{}) *glue.StorageDescriptor {
	if len(l) == 0 {
		return nil
	}

	s := l[0].(map[string]interface{})
	storageDescriptor := &glue.StorageDescriptor{}

	if v, ok := s["columns"]; ok {
		columns := expandGlueColumns(v.([]interface{}))
		storageDescriptor.Columns = columns
	}

	if v, ok := s["location"]; ok {
		storageDescriptor.Location = aws.String(v.(string))
	}

	if v, ok := s["input_format"]; ok {
		storageDescriptor.InputFormat = aws.String(v.(string))
	}

	if v, ok := s["output_format"]; ok {
		storageDescriptor.OutputFormat = aws.String(v.(string))
	}

	if v, ok := s["compressed"]; ok {
		storageDescriptor.Compressed = aws.Bool(v.(bool))
	}

	if v, ok := s["number_of_buckets"]; ok {
		storageDescriptor.NumberOfBuckets = aws.Int64(int64(v.(int)))
	}

	if v, ok := s["ser_de_info"]; ok {
		storageDescriptor.SerdeInfo = expandGlueSerDeInfo(v.([]interface{}))
	}

	if v, ok := s["bucket_columns"]; ok {
		bucketColumns := make([]string, len(v.([]interface{})))
		for i, item := range v.([]interface{}) {
			bucketColumns[i] = fmt.Sprint(item)
		}
		storageDescriptor.BucketColumns = aws.StringSlice(bucketColumns)
	}

	if v, ok := s["sort_columns"]; ok {
		storageDescriptor.SortColumns = expandGlueSortColumns(v.([]interface{}))
	}

	if v, ok := s["skewed_info"]; ok {
		storageDescriptor.SkewedInfo = expandGlueSkewedInfo(v.([]interface{}))
	}

	if v, ok := s["parameters"]; ok {
		paramsMap := map[string]string{}
		for key, value := range v.(map[string]interface{}) {
			paramsMap[key] = value.(string)
		}
		storageDescriptor.Parameters = aws.StringMap(paramsMap)
	}

	if v, ok := s["stored_as_sub_directories"]; ok {
		storageDescriptor.StoredAsSubDirectories = aws.Bool(v.(bool))
	}

	return storageDescriptor
}

func expandGlueColumns(columns []interface{}) []*glue.Column {
	columnSlice := []*glue.Column{}
	for _, element := range columns {
		elementMap := element.(map[string]interface{})

		column := &glue.Column{
			Name: aws.String(elementMap["name"].(string)),
		}

		if v, ok := elementMap["comment"]; ok {
			column.Comment = aws.String(v.(string))
		}

		if v, ok := elementMap["type"]; ok {
			column.Type = aws.String(v.(string))
		}

		columnSlice = append(columnSlice, column)
	}

	return columnSlice
}

func expandGlueSerDeInfo(l []interface{}) *glue.SerDeInfo {
	if len(l) == 0 {
		return nil
	}

	s := l[0].(map[string]interface{})
	serDeInfo := &glue.SerDeInfo{}

	if v, ok := s["name"]; ok {
		serDeInfo.Name = aws.String(v.(string))
	}

	if v, ok := s["parameters"]; ok {
		paramsMap := map[string]string{}
		for key, value := range v.(map[string]interface{}) {
			paramsMap[key] = value.(string)
		}
		serDeInfo.Parameters = aws.StringMap(paramsMap)
	}

	if v, ok := s["serialization_library"]; ok {
		serDeInfo.SerializationLibrary = aws.String(v.(string))
	}

	return serDeInfo
}

func expandGlueSortColumns(columns []interface{}) []*glue.Order {
	orderSlice := make([]*glue.Order, len(columns))

	for i, element := range columns {
		elementMap := element.(map[string]interface{})

		order := &glue.Order{
			Column: aws.String(elementMap["column"].(string)),
		}

		if v, ok := elementMap["sort_order"]; ok {
			order.SortOrder = aws.Int64(int64(v.(int)))
		}

		orderSlice[i] = order
	}

	return orderSlice
}

func expandGlueSkewedInfo(l []interface{}) *glue.SkewedInfo {
	if len(l) == 0 {
		return nil
	}

	s := l[0].(map[string]interface{})
	skewedInfo := &glue.SkewedInfo{}

	if v, ok := s["skewed_column_names"]; ok {
		columnsSlice := make([]string, len(v.([]interface{})))
		for i, item := range v.([]interface{}) {
			columnsSlice[i] = fmt.Sprint(item)
		}
		skewedInfo.SkewedColumnNames = aws.StringSlice(columnsSlice)
	}

	if v, ok := s["skewed_column_value_location_maps"]; ok {
		typeMap := map[string]string{}
		for key, value := range v.(map[string]interface{}) {
			typeMap[key] = value.(string)
		}
		skewedInfo.SkewedColumnValueLocationMaps = aws.StringMap(typeMap)
	}

	if v, ok := s["skewed_column_values"]; ok {
		columnsSlice := make([]string, len(v.([]interface{})))
		for i, item := range v.([]interface{}) {
			columnsSlice[i] = fmt.Sprint(item)
		}
		skewedInfo.SkewedColumnValues = aws.StringSlice(columnsSlice)
	}

	return skewedInfo
}

func flattenGlueStorageDescriptor(s *glue.StorageDescriptor) []map[string]interface{} {
	if s == nil {
		storageDescriptors := make([]map[string]interface{}, 0)
		return storageDescriptors
	}

	storageDescriptors := make([]map[string]interface{}, 1)

	storageDescriptor := make(map[string]interface{})

	storageDescriptor["columns"] = flattenGlueColumns(s.Columns)
	storageDescriptor["location"] = aws.StringValue(s.Location)
	storageDescriptor["input_format"] = aws.StringValue(s.InputFormat)
	storageDescriptor["output_format"] = aws.StringValue(s.OutputFormat)
	storageDescriptor["compressed"] = aws.BoolValue(s.Compressed)
	storageDescriptor["number_of_buckets"] = aws.Int64Value(s.NumberOfBuckets)
	storageDescriptor["ser_de_info"] = flattenGlueSerDeInfo(s.SerdeInfo)
	storageDescriptor["bucket_columns"] = flattenStringList(s.BucketColumns)
	storageDescriptor["sort_columns"] = flattenGlueOrders(s.SortColumns)
	storageDescriptor["parameters"] = aws.StringValueMap(s.Parameters)
	storageDescriptor["skewed_info"] = flattenGlueSkewedInfo(s.SkewedInfo)
	storageDescriptor["stored_as_sub_directories"] = aws.BoolValue(s.StoredAsSubDirectories)

	storageDescriptors[0] = storageDescriptor

	return storageDescriptors
}

func flattenGlueColumns(cs []*glue.Column) []map[string]string {
	columnsSlice := make([]map[string]string, len(cs))
	if len(cs) > 0 {
		for i, v := range cs {
			columnsSlice[i] = flattenGlueColumn(v)
		}
	}

	return columnsSlice
}

func flattenGlueColumn(c *glue.Column) map[string]string {
	column := make(map[string]string)

	if c == nil {
		return column
	}

	if v := aws.StringValue(c.Name); v != "" {
		column["name"] = v
	}

	if v := aws.StringValue(c.Type); v != "" {
		column["type"] = v
	}

	if v := aws.StringValue(c.Comment); v != "" {
		column["comment"] = v
	}

	return column
}

func flattenGlueSerDeInfo(s *glue.SerDeInfo) []map[string]interface{} {
	if s == nil {
		serDeInfos := make([]map[string]interface{}, 0)
		return serDeInfos
	}

	serDeInfos := make([]map[string]interface{}, 1)
	serDeInfo := make(map[string]interface{})

	serDeInfo["name"] = aws.StringValue(s.Name)
	serDeInfo["parameters"] = aws.StringValueMap(s.Parameters)
	serDeInfo["serialization_library"] = aws.StringValue(s.SerializationLibrary)

	serDeInfos[0] = serDeInfo
	return serDeInfos
}

func flattenGlueOrders(os []*glue.Order) []map[string]interface{} {
	orders := make([]map[string]interface{}, len(os))
	for i, v := range os {
		order := make(map[string]interface{})
		order["column"] = aws.StringValue(v.Column)
		order["sort_order"] = int(aws.Int64Value(v.SortOrder))
		orders[i] = order
	}

	return orders
}

func flattenGlueSkewedInfo(s *glue.SkewedInfo) []map[string]interface{} {
	if s == nil {
		skewedInfoSlice := make([]map[string]interface{}, 0)
		return skewedInfoSlice
	}

	skewedInfoSlice := make([]map[string]interface{}, 1)

	skewedInfo := make(map[string]interface{})
	skewedInfo["skewed_column_names"] = flattenStringList(s.SkewedColumnNames)
	skewedInfo["skewed_column_value_location_maps"] = aws.StringValueMap(s.SkewedColumnValueLocationMaps)
	skewedInfo["skewed_column_values"] = flattenStringList(s.SkewedColumnValues)
	skewedInfoSlice[0] = skewedInfo

	return skewedInfoSlice
}
