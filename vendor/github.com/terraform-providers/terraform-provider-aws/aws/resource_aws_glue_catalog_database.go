package aws

import (
	"fmt"
	"log"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/glue"

	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsGlueCatalogDatabase() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsGlueCatalogDatabaseCreate,
		Read:   resourceAwsGlueCatalogDatabaseRead,
		Update: resourceAwsGlueCatalogDatabaseUpdate,
		Delete: resourceAwsGlueCatalogDatabaseDelete,
		Exists: resourceAwsGlueCatalogDatabaseExists,
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
			"name": {
				Type:     schema.TypeString,
				ForceNew: true,
				Required: true,
			},
			"description": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"location_uri": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"parameters": {
				Type:     schema.TypeMap,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Optional: true,
			},
		},
	}
}

func resourceAwsGlueCatalogDatabaseCreate(d *schema.ResourceData, meta interface{}) error {
	glueconn := meta.(*AWSClient).glueconn
	catalogID := createAwsGlueCatalogID(d, meta.(*AWSClient).accountid)
	name := d.Get("name").(string)

	input := &glue.CreateDatabaseInput{
		CatalogId: aws.String(catalogID),
		DatabaseInput: &glue.DatabaseInput{
			Name: aws.String(name),
		},
	}

	_, err := glueconn.CreateDatabase(input)
	if err != nil {
		return fmt.Errorf("Error creating Catalog Database: %s", err)
	}

	d.SetId(fmt.Sprintf("%s:%s", catalogID, name))

	return resourceAwsGlueCatalogDatabaseUpdate(d, meta)
}

func resourceAwsGlueCatalogDatabaseUpdate(d *schema.ResourceData, meta interface{}) error {
	glueconn := meta.(*AWSClient).glueconn

	catalogID, name, err := readAwsGlueCatalogID(d.Id())
	if err != nil {
		return err
	}

	dbUpdateInput := &glue.UpdateDatabaseInput{
		CatalogId: aws.String(catalogID),
		Name:      aws.String(name),
	}

	dbInput := &glue.DatabaseInput{
		Name: aws.String(name),
	}

	if desc, ok := d.GetOk("description"); ok {
		dbInput.Description = aws.String(desc.(string))
	}

	if loc, ok := d.GetOk("location_uri"); ok {
		dbInput.LocationUri = aws.String(loc.(string))
	}

	if params, ok := d.GetOk("parameters"); ok {
		parametersInput := make(map[string]*string)
		for key, value := range params.(map[string]interface{}) {
			parametersInput[key] = aws.String(value.(string))
		}
		dbInput.Parameters = parametersInput
	}

	dbUpdateInput.DatabaseInput = dbInput

	if d.HasChange("description") || d.HasChange("location_uri") || d.HasChange("parameters") {
		if _, err := glueconn.UpdateDatabase(dbUpdateInput); err != nil {
			return err
		}
	}

	return resourceAwsGlueCatalogDatabaseRead(d, meta)
}

func resourceAwsGlueCatalogDatabaseRead(d *schema.ResourceData, meta interface{}) error {
	glueconn := meta.(*AWSClient).glueconn

	catalogID, name, err := readAwsGlueCatalogID(d.Id())
	if err != nil {
		return err
	}

	input := &glue.GetDatabaseInput{
		CatalogId: aws.String(catalogID),
		Name:      aws.String(name),
	}

	out, err := glueconn.GetDatabase(input)
	if err != nil {

		if isAWSErr(err, glue.ErrCodeEntityNotFoundException, "") {
			log.Printf("[WARN] Glue Catalog Database (%s) not found, removing from state", d.Id())
			d.SetId("")
			return nil
		}

		return fmt.Errorf("Error reading Glue Catalog Database: %s", err.Error())
	}

	d.Set("name", out.Database.Name)
	d.Set("catalog_id", catalogID)
	d.Set("description", out.Database.Description)
	d.Set("location_uri", out.Database.LocationUri)

	dParams := make(map[string]string)
	if len(out.Database.Parameters) > 0 {
		for key, value := range out.Database.Parameters {
			dParams[key] = *value
		}
	}
	d.Set("parameters", dParams)

	return nil
}

func resourceAwsGlueCatalogDatabaseDelete(d *schema.ResourceData, meta interface{}) error {
	glueconn := meta.(*AWSClient).glueconn
	catalogID, name, err := readAwsGlueCatalogID(d.Id())
	if err != nil {
		return err
	}

	log.Printf("[DEBUG] Glue Catalog Database: %s:%s", catalogID, name)
	_, err = glueconn.DeleteDatabase(&glue.DeleteDatabaseInput{
		Name: aws.String(name),
	})
	if err != nil {
		return fmt.Errorf("Error deleting Glue Catalog Database: %s", err.Error())
	}
	return nil
}

func resourceAwsGlueCatalogDatabaseExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	glueconn := meta.(*AWSClient).glueconn
	catalogID, name, err := readAwsGlueCatalogID(d.Id())
	if err != nil {
		return false, err
	}

	input := &glue.GetDatabaseInput{
		CatalogId: aws.String(catalogID),
		Name:      aws.String(name),
	}

	_, err = glueconn.GetDatabase(input)
	if err != nil {
		if isAWSErr(err, glue.ErrCodeEntityNotFoundException, "") {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func readAwsGlueCatalogID(id string) (catalogID string, name string, err error) {
	idParts := strings.Split(id, ":")
	if len(idParts) != 2 {
		return "", "", fmt.Errorf("Unexpected format of ID (%q), expected CATALOG-ID:DATABASE-NAME", id)
	}
	return idParts[0], idParts[1], nil
}

func createAwsGlueCatalogID(d *schema.ResourceData, accountid string) (catalogID string) {
	if rawCatalogID, ok := d.GetOkExists("catalog_id"); ok {
		catalogID = rawCatalogID.(string)
	} else {
		catalogID = accountid
	}
	return
}
