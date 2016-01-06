package google

import (
	"encoding/json"
	"fmt"
	"github.com/hashicorp/terraform/helper/schema"
	"google.golang.org/api/bigquery/v2"
	"io/ioutil"
)

func resourceBigQueryTable() *schema.Resource {
	return &schema.Resource{
		Create: resourceBigQueryTableCreate,
		Read:   resourceBigQueryTableRead,
		Delete: resourceBigQueryTableDelete,
		Update: resourceBigQueryTableUpdate,

		Schema: map[string]*schema.Schema{
			"tableId": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"datasetId": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"description": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},

			"expirationTime": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
			},

			"friendlyName": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},

			"schemaFile": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},

			"schema": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"description": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"mode": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"name": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"type": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},

			"self_link": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"id": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"kind": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"lastModifiedTime": &schema.Schema{
				Type:     schema.TypeInt,
				Computed: true,
			},
		},
	}
}

// convert raw field config into TableFieldSchema ref
func parseField(fieldDef map[string]interface{}) (*bigquery.TableFieldSchema, error) {
	fieldParsed := &bigquery.TableFieldSchema{}
	if description, ok := fieldDef["description"]; ok {
		fieldParsed.Description = description.(string)
	}

	if mode, ok := fieldDef["mode"]; ok {
		fieldParsed.Mode = mode.(string)
	}

	if name, ok := fieldDef["name"]; ok {
		fieldParsed.Name = name.(string)
	} else {
		return nil, fmt.Errorf("All fields must have 'name' defined.  The following field did not:  %q\n", fieldDef)
	}

	if fieldType, ok := fieldDef["type"]; ok {
		fieldParsed.Type = fieldType.(string)
	} else {
		return nil, fmt.Errorf("All fields must have 'type' defined.  The following field did not:  %q\n", fieldDef)
	}

	if tableFieldSchema, ok := fieldDef["fields"]; ok {
		fieldList, err := parseFieldList(tableFieldSchema.([]interface{}))
		if err != nil {
			return nil, err
		}
		fieldParsed.Fields = fieldList
	}

	return fieldParsed, nil
}

//  convert list of raw field data into list of TableFieldSchema refs
func parseFieldList(schema []interface{}) ([]*bigquery.TableFieldSchema, error) {
	tableFieldList := make([]*bigquery.TableFieldSchema, 0)
	for _, fieldInterface := range schema {
		fieldParsed, err := parseField(fieldInterface.(map[string]interface{}))
		if err != nil {
			return nil, err
		}
		tableFieldList = append(tableFieldList, fieldParsed)
	}
	return tableFieldList, nil
}

func resourceBigQueryTableCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	// build tableRef
	datasetId := d.Get("datasetId").(string)
	tableId := d.Get("tableId").(string)
	tRef := &bigquery.TableReference{DatasetId: datasetId, ProjectId: config.Project, TableId: tableId}

	// build the table
	table := &bigquery.Table{TableReference: tRef}

	if description, ok := d.GetOk("description"); ok {
		table.Description = description.(string)
	}

	if expirationTime, ok := d.GetOk("expirationTime"); ok {
		table.ExpirationTime = expirationTime.(int64)
	}

	if friendlyName, ok := d.GetOk("friendlyName"); ok {
		table.FriendlyName = friendlyName.(string)
	}

	// build arbitrarily deep table schema
	//   first check that didn't specify both schema and schemaFile
	schema, schemaOk := d.GetOk("schema")
	schemaFile, schemaFileOk := d.GetOk("schemaFile")
	if schemaOk && schemaFileOk {
		return fmt.Errorf("Config contains both schema and schemaFile.  Specify at most one\n")
	} else if schemaOk {
		fieldList, err := parseFieldList(schema.([]interface{}))
		if err != nil {
			return err
		}
		table.Schema = &bigquery.TableSchema{Fields: fieldList}
	} else if schemaFileOk {
		schemaJson, err := ioutil.ReadFile(schemaFile.(string))
		if err != nil {
			return err
		}

		var schemaJsonInterface []interface{}
		err = json.Unmarshal(schemaJson, &schemaJsonInterface)
		if err != nil {
			return fmt.Errorf("Failed to decode json file with error: %q", err)
		}

		fieldList, err := parseFieldList(schemaJsonInterface)
		if err != nil {
			return err
		}
		table.Schema = &bigquery.TableSchema{Fields: fieldList}
	}

	call := config.clientBigQuery.Tables.Insert(config.Project, datasetId, table)
	_, err := call.Do()
	if err != nil {
		return err
	}

	err = resourceBigQueryTableRead(d, meta)
	if err != nil {
		return err
	}

	return nil
}

func resourceBigQueryTableRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	call := config.clientBigQuery.Tables.Get(config.Project, d.Get("datasetId").(string), d.Get("tableId").(string))
	res, err := call.Do()
	if err != nil {
		return err
	}

	d.SetId(res.Id)
	d.Set("self_link", res.SelfLink)
	d.Set("lastModifiedTime", res.LastModifiedTime)
	d.Set("id", res.Id)
	d.Set("kind", res.Kind)
	return nil
}

func resourceBigQueryTableUpdate(d *schema.ResourceData, meta interface{}) error {
	return nil
}

func resourceBigQueryTableDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	call := config.clientBigQuery.Tables.Delete(config.Project, d.Get("datasetId").(string), d.Get("tableId").(string))
	err := call.Do()
	if err != nil {
		return err
	}

	d.SetId("")
	return nil
}
