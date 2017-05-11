package google

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/structure"
	"github.com/hashicorp/terraform/helper/validation"
	"google.golang.org/api/bigquery/v2"
)

func resourceBigQueryTable() *schema.Resource {
	return &schema.Resource{
		Create: resourceBigQueryTableCreate,
		Read:   resourceBigQueryTableRead,
		Delete: resourceBigQueryTableDelete,
		Update: resourceBigQueryTableUpdate,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},
		Schema: map[string]*schema.Schema{
			// TableId: [Required] The ID of the table. The ID must contain only
			// letters (a-z, A-Z), numbers (0-9), or underscores (_). The maximum
			// length is 1,024 characters.
			"table_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			// DatasetId: [Required] The ID of the dataset containing this table.
			"dataset_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			// ProjectId: [Required] The ID of the project containing this table.
			"project": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			// Description: [Optional] A user-friendly description of this table.
			"description": {
				Type:     schema.TypeString,
				Optional: true,
			},

			// ExpirationTime: [Optional] The time when this table expires, in
			// milliseconds since the epoch. If not present, the table will persist
			// indefinitely. Expired tables will be deleted and their storage
			// reclaimed.
			"expiration_time": {
				Type:     schema.TypeInt,
				Optional: true,
				Computed: true,
			},

			// FriendlyName: [Optional] A descriptive name for this table.
			"friendly_name": {
				Type:     schema.TypeString,
				Optional: true,
			},

			// Labels: [Experimental] The labels associated with this table. You can
			// use these to organize and group your tables. Label keys and values
			// can be no longer than 63 characters, can only contain lowercase
			// letters, numeric characters, underscores and dashes. International
			// characters are allowed. Label values are optional. Label keys must
			// start with a letter and each label in the list must have a different
			// key.
			"labels": &schema.Schema{
				Type:     schema.TypeMap,
				Optional: true,
				Elem:     schema.TypeString,
			},

			// Schema: [Optional] Describes the schema of this table.
			"schema": {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				ValidateFunc: validation.ValidateJsonString,
				StateFunc: func(v interface{}) string {
					json, _ := structure.NormalizeJsonString(v)
					return json
				},
			},

			// TimePartitioning: [Experimental] If specified, configures time-based
			// partitioning for this table.
			"time_partitioning": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						// ExpirationMs: [Optional] Number of milliseconds for which to keep the
						// storage for a partition.
						"expiration_ms": {
							Type:     schema.TypeInt,
							Optional: true,
						},

						// Type: [Required] The only type supported is DAY, which will generate
						// one partition per day based on data loading time.
						"type": {
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: validation.StringInSlice([]string{"DAY"}, false),
						},
					},
				},
			},

			// CreationTime: [Output-only] The time when this table was created, in
			// milliseconds since the epoch.
			"creation_time": {
				Type:     schema.TypeInt,
				Computed: true,
			},

			// Etag: [Output-only] A hash of this resource.
			"etag": {
				Type:     schema.TypeString,
				Computed: true,
			},

			// LastModifiedTime: [Output-only] The time when this table was last
			// modified, in milliseconds since the epoch.
			"last_modified_time": {
				Type:     schema.TypeInt,
				Computed: true,
			},

			// Location: [Output-only] The geographic location where the table
			// resides. This value is inherited from the dataset.
			"location": {
				Type:     schema.TypeString,
				Computed: true,
			},

			// NumBytes: [Output-only] The size of this table in bytes, excluding
			// any data in the streaming buffer.
			"num_bytes": {
				Type:     schema.TypeInt,
				Computed: true,
			},

			// NumLongTermBytes: [Output-only] The number of bytes in the table that
			// are considered "long-term storage".
			"num_long_term_bytes": {
				Type:     schema.TypeInt,
				Computed: true,
			},

			// NumRows: [Output-only] The number of rows of data in this table,
			// excluding any data in the streaming buffer.
			"num_rows": {
				Type:     schema.TypeInt,
				Computed: true,
			},

			// SelfLink: [Output-only] A URL that can be used to access this
			// resource again.
			"self_link": {
				Type:     schema.TypeString,
				Computed: true,
			},

			// Type: [Output-only] Describes the table type. The following values
			// are supported: TABLE: A normal BigQuery table. VIEW: A virtual table
			// defined by a SQL query. EXTERNAL: A table that references data stored
			// in an external storage system, such as Google Cloud Storage. The
			// default value is TABLE.
			"type": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceTable(d *schema.ResourceData, meta interface{}) (*bigquery.Table, error) {
	config := meta.(*Config)

	project, err := getProject(d, config)
	if err != nil {
		return nil, err
	}

	table := &bigquery.Table{
		TableReference: &bigquery.TableReference{
			DatasetId: d.Get("dataset_id").(string),
			TableId:   d.Get("table_id").(string),
			ProjectId: project,
		},
	}

	if v, ok := d.GetOk("description"); ok {
		table.Description = v.(string)
	}

	if v, ok := d.GetOk("expiration_time"); ok {
		table.ExpirationTime = v.(int64)
	}

	if v, ok := d.GetOk("friendly_name"); ok {
		table.FriendlyName = v.(string)
	}

	if v, ok := d.GetOk("labels"); ok {
		labels := map[string]string{}

		for k, v := range v.(map[string]interface{}) {
			labels[k] = v.(string)
		}

		table.Labels = labels
	}

	if v, ok := d.GetOk("schema"); ok {
		schema, err := expandSchema(v)
		if err != nil {
			return nil, err
		}

		table.Schema = schema
	}

	if v, ok := d.GetOk("time_partitioning"); ok {
		table.TimePartitioning = expandTimePartitioning(v)
	}

	return table, nil
}

func resourceBigQueryTableCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	project, err := getProject(d, config)
	if err != nil {
		return err
	}

	table, err := resourceTable(d, meta)
	if err != nil {
		return err
	}

	datasetID := d.Get("dataset_id").(string)

	log.Printf("[INFO] Creating BigQuery table: %s", table.TableReference.TableId)

	res, err := config.clientBigQuery.Tables.Insert(project, datasetID, table).Do()
	if err != nil {
		return err
	}

	log.Printf("[INFO] BigQuery table %s has been created", res.Id)

	d.SetId(fmt.Sprintf("%s:%s.%s", res.TableReference.ProjectId, res.TableReference.DatasetId, res.TableReference.TableId))

	return resourceBigQueryTableRead(d, meta)
}

func resourceBigQueryTableParseID(id string) (string, string, string) {
	parts := strings.FieldsFunc(id, func(r rune) bool { return r == ':' || r == '.' })
	return parts[0], parts[1], parts[2] // projectID, datasetID, tableID
}

func resourceBigQueryTableRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	log.Printf("[INFO] Reading BigQuery table: %s", d.Id())

	projectID, datasetID, tableID := resourceBigQueryTableParseID(d.Id())

	res, err := config.clientBigQuery.Tables.Get(projectID, datasetID, tableID).Do()
	if err != nil {
		return handleNotFoundError(err, d, fmt.Sprintf("BigQuery table %q", tableID))
	}

	d.Set("description", res.Description)
	d.Set("expiration_time", res.ExpirationTime)
	d.Set("friendly_name", res.FriendlyName)
	d.Set("labels", res.Labels)
	d.Set("creation_time", res.CreationTime)
	d.Set("etag", res.Etag)
	d.Set("last_modified_time", res.LastModifiedTime)
	d.Set("location", res.Location)
	d.Set("num_bytes", res.NumBytes)
	d.Set("table_id", res.TableReference.TableId)
	d.Set("dataset_id", res.TableReference.DatasetId)
	d.Set("num_long_term_bytes", res.NumLongTermBytes)
	d.Set("num_rows", res.NumRows)
	d.Set("self_link", res.SelfLink)
	d.Set("type", res.Type)

	if res.TimePartitioning != nil {
		if err := d.Set("time_partitioning", flattenTimePartitioning(res.TimePartitioning)); err != nil {
			return err
		}
	}

	if res.Schema != nil {
		schema, err := flattenSchema(res.Schema)
		if err != nil {
			return err
		}

		d.Set("schema", schema)
	}

	return nil
}

func resourceBigQueryTableUpdate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	table, err := resourceTable(d, meta)
	if err != nil {
		return err
	}

	log.Printf("[INFO] Updating BigQuery table: %s", d.Id())

	projectID, datasetID, tableID := resourceBigQueryTableParseID(d.Id())

	if _, err = config.clientBigQuery.Tables.Update(projectID, datasetID, tableID, table).Do(); err != nil {
		return err
	}

	return resourceBigQueryTableRead(d, meta)
}

func resourceBigQueryTableDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	log.Printf("[INFO] Deleting BigQuery table: %s", d.Id())

	projectID, datasetID, tableID := resourceBigQueryTableParseID(d.Id())

	if err := config.clientBigQuery.Tables.Delete(projectID, datasetID, tableID).Do(); err != nil {
		return err
	}

	d.SetId("")

	return nil
}

func expandSchema(raw interface{}) (*bigquery.TableSchema, error) {
	var fields []*bigquery.TableFieldSchema

	if err := json.Unmarshal([]byte(raw.(string)), &fields); err != nil {
		return nil, err
	}

	return &bigquery.TableSchema{Fields: fields}, nil
}

func flattenSchema(tableSchema *bigquery.TableSchema) (string, error) {
	schema, err := json.Marshal(tableSchema.Fields)
	if err != nil {
		return "", err
	}

	return string(schema), nil
}

func expandTimePartitioning(configured interface{}) *bigquery.TimePartitioning {
	raw := configured.([]interface{})[0].(map[string]interface{})
	tp := &bigquery.TimePartitioning{Type: raw["type"].(string)}

	if v, ok := raw["expiration_ms"]; ok {
		tp.ExpirationMs = int64(v.(int))
	}

	return tp
}

func flattenTimePartitioning(tp *bigquery.TimePartitioning) []map[string]interface{} {
	result := map[string]interface{}{"type": tp.Type}

	if tp.ExpirationMs != 0 {
		result["expiration_ms"] = tp.ExpirationMs
	}

	return []map[string]interface{}{result}
}
