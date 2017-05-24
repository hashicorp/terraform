package google

import (
	"fmt"
	"log"
	"regexp"
	"strings"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
	"google.golang.org/api/bigquery/v2"
)

func resourceBigQueryDataset() *schema.Resource {
	return &schema.Resource{
		Create: resourceBigQueryDatasetCreate,
		Read:   resourceBigQueryDatasetRead,
		Update: resourceBigQueryDatasetUpdate,
		Delete: resourceBigQueryDatasetDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},
		Schema: map[string]*schema.Schema{
			// DatasetId: [Required] A unique ID for this dataset, without the
			// project name. The ID must contain only letters (a-z, A-Z), numbers
			// (0-9), or underscores (_). The maximum length is 1,024 characters.
			"dataset_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
				ValidateFunc: func(v interface{}, k string) (ws []string, errors []error) {
					value := v.(string)
					if !regexp.MustCompile(`^[0-9A-Za-z_]+$`).MatchString(value) {
						errors = append(errors, fmt.Errorf(
							"%q must contain only letters (a-z, A-Z), numbers (0-9), or underscores (_)", k))
					}

					if len(value) > 1024 {
						errors = append(errors, fmt.Errorf(
							"%q cannot be greater than 1,024 characters", k))
					}

					return
				},
			},

			// ProjectId: [Optional] The ID of the project containing this dataset.
			"project": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			// FriendlyName: [Optional] A descriptive name for the dataset.
			"friendly_name": {
				Type:     schema.TypeString,
				Optional: true,
			},

			// Description: [Optional] A user-friendly description of the dataset.
			"description": {
				Type:     schema.TypeString,
				Optional: true,
			},

			// Location: [Experimental] The geographic location where the dataset
			// should reside. Possible values include EU and US. The default value
			// is US.
			"location": {
				Type:         schema.TypeString,
				Optional:     true,
				ForceNew:     true,
				Default:      "US",
				ValidateFunc: validation.StringInSlice([]string{"US", "EU"}, false),
			},

			// DefaultTableExpirationMs: [Optional] The default lifetime of all
			// tables in the dataset, in milliseconds. The minimum value is 3600000
			// milliseconds (one hour). Once this property is set, all newly-created
			// tables in the dataset will have an expirationTime property set to the
			// creation time plus the value in this property, and changing the value
			// will only affect new tables, not existing ones. When the
			// expirationTime for a given table is reached, that table will be
			// deleted automatically. If a table's expirationTime is modified or
			// removed before the table expires, or if you provide an explicit
			// expirationTime when creating a table, that value takes precedence
			// over the default expiration time indicated by this property.
			"default_table_expiration_ms": {
				Type:     schema.TypeInt,
				Optional: true,
				ValidateFunc: func(v interface{}, k string) (ws []string, errors []error) {
					value := v.(int)
					if value < 3600000 {
						errors = append(errors, fmt.Errorf("%q cannot be shorter than 3600000 milliseconds (one hour)", k))
					}

					return
				},
			},

			// Labels: [Experimental] The labels associated with this dataset. You
			// can use these to organize and group your datasets. You can set this
			// property when inserting or updating a dataset.
			"labels": &schema.Schema{
				Type:     schema.TypeMap,
				Optional: true,
				Elem:     schema.TypeString,
			},

			// SelfLink: [Output-only] A URL that can be used to access the resource
			// again. You can use this URL in Get or Update requests to the
			// resource.
			"self_link": {
				Type:     schema.TypeString,
				Computed: true,
			},

			// Etag: [Output-only] A hash of the resource.
			"etag": {
				Type:     schema.TypeString,
				Computed: true,
			},

			// CreationTime: [Output-only] The time when this dataset was created,
			// in milliseconds since the epoch.
			"creation_time": {
				Type:     schema.TypeInt,
				Computed: true,
			},

			// LastModifiedTime: [Output-only] The date when this dataset or any of
			// its tables was last modified, in milliseconds since the epoch.
			"last_modified_time": {
				Type:     schema.TypeInt,
				Computed: true,
			},
		},
	}
}

func resourceDataset(d *schema.ResourceData, meta interface{}) (*bigquery.Dataset, error) {
	config := meta.(*Config)

	project, err := getProject(d, config)
	if err != nil {
		return nil, err
	}

	dataset := &bigquery.Dataset{
		DatasetReference: &bigquery.DatasetReference{
			DatasetId: d.Get("dataset_id").(string),
			ProjectId: project,
		},
	}

	if v, ok := d.GetOk("friendly_name"); ok {
		dataset.FriendlyName = v.(string)
	}

	if v, ok := d.GetOk("description"); ok {
		dataset.Description = v.(string)
	}

	if v, ok := d.GetOk("location"); ok {
		dataset.Location = v.(string)
	}

	if v, ok := d.GetOk("default_table_expiration_ms"); ok {
		dataset.DefaultTableExpirationMs = int64(v.(int))
	}

	if v, ok := d.GetOk("labels"); ok {
		labels := map[string]string{}

		for k, v := range v.(map[string]interface{}) {
			labels[k] = v.(string)
		}

		dataset.Labels = labels
	}

	return dataset, nil
}

func resourceBigQueryDatasetCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	project, err := getProject(d, config)
	if err != nil {
		return err
	}

	dataset, err := resourceDataset(d, meta)
	if err != nil {
		return err
	}

	log.Printf("[INFO] Creating BigQuery dataset: %s", dataset.DatasetReference.DatasetId)

	res, err := config.clientBigQuery.Datasets.Insert(project, dataset).Do()
	if err != nil {
		return err
	}

	log.Printf("[INFO] BigQuery dataset %s has been created", res.Id)

	d.SetId(res.Id)

	return resourceBigQueryDatasetRead(d, meta)
}

func resourceBigQueryDatasetParseID(id string) (string, string) {
	// projectID, datasetID
	parts := strings.Split(id, ":")
	return parts[0], parts[1]
}

func resourceBigQueryDatasetRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	log.Printf("[INFO] Reading BigQuery dataset: %s", d.Id())

	projectID, datasetID := resourceBigQueryDatasetParseID(d.Id())

	res, err := config.clientBigQuery.Datasets.Get(projectID, datasetID).Do()
	if err != nil {
		return handleNotFoundError(err, d, fmt.Sprintf("BigQuery dataset %q", datasetID))
	}

	d.Set("etag", res.Etag)
	d.Set("labels", res.Labels)
	d.Set("location", res.Location)
	d.Set("self_link", res.SelfLink)
	d.Set("description", res.Description)
	d.Set("friendly_name", res.FriendlyName)
	d.Set("creation_time", res.CreationTime)
	d.Set("last_modified_time", res.LastModifiedTime)
	d.Set("dataset_id", res.DatasetReference.DatasetId)
	d.Set("default_table_expiration_ms", res.DefaultTableExpirationMs)

	return nil
}

func resourceBigQueryDatasetUpdate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	dataset, err := resourceDataset(d, meta)
	if err != nil {
		return err
	}

	log.Printf("[INFO] Updating BigQuery dataset: %s", d.Id())

	projectID, datasetID := resourceBigQueryDatasetParseID(d.Id())

	if _, err = config.clientBigQuery.Datasets.Update(projectID, datasetID, dataset).Do(); err != nil {
		return err
	}

	return resourceBigQueryDatasetRead(d, meta)
}

func resourceBigQueryDatasetDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	log.Printf("[INFO] Deleting BigQuery dataset: %s", d.Id())

	projectID, datasetID := resourceBigQueryDatasetParseID(d.Id())

	if err := config.clientBigQuery.Datasets.Delete(projectID, datasetID).Do(); err != nil {
		return err
	}

	d.SetId("")
	return nil
}
