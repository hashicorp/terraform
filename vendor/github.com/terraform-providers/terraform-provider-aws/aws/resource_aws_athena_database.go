package aws

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/hashicorp/terraform/helper/validation"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/athena"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsAthenaDatabase() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsAthenaDatabaseCreate,
		Read:   resourceAwsAthenaDatabaseRead,
		Update: resourceAwsAthenaDatabaseUpdate,
		Delete: resourceAwsAthenaDatabaseDelete,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringMatch(regexp.MustCompile("^[_a-z0-9]+$"), "see https://docs.aws.amazon.com/athena/latest/ug/tables-databases-columns-names.html"),
			},
			"bucket": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"force_destroy": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"encryption_configuration": {
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"kms_key": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"encryption_option": {
							Type:     schema.TypeString,
							Required: true,
							ValidateFunc: validation.StringInSlice([]string{
								athena.EncryptionOptionCseKms,
								athena.EncryptionOptionSseKms,
								athena.EncryptionOptionSseS3,
							}, false),
						},
					},
				},
			},
		},
	}
}

func expandAthenaResultConfiguration(bucket string, encryptionConfigurationList []interface{}) (*athena.ResultConfiguration, error) {
	resultConfig := athena.ResultConfiguration{
		OutputLocation: aws.String("s3://" + bucket),
	}

	if len(encryptionConfigurationList) <= 0 {
		return &resultConfig, nil
	}

	data := encryptionConfigurationList[0].(map[string]interface{})
	keyType := data["encryption_option"].(string)
	keyID := data["kms_key"].(string)

	encryptionConfig := athena.EncryptionConfiguration{
		EncryptionOption: aws.String(keyType),
	}

	if len(keyID) > 0 {
		encryptionConfig.KmsKey = aws.String(keyID)
	}

	resultConfig.EncryptionConfiguration = &encryptionConfig

	return &resultConfig, nil
}

func resourceAwsAthenaDatabaseCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).athenaconn

	resultConfig, err := expandAthenaResultConfiguration(d.Get("bucket").(string), d.Get("encryption_configuration").([]interface{}))
	if err != nil {
		return err
	}

	input := &athena.StartQueryExecutionInput{
		QueryString:         aws.String(fmt.Sprintf("create database `%s`;", d.Get("name").(string))),
		ResultConfiguration: resultConfig,
	}

	resp, err := conn.StartQueryExecution(input)
	if err != nil {
		return err
	}

	if err := executeAndExpectNoRowsWhenCreate(*resp.QueryExecutionId, d, conn); err != nil {
		return err
	}
	d.SetId(d.Get("name").(string))
	return resourceAwsAthenaDatabaseRead(d, meta)
}

func resourceAwsAthenaDatabaseRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).athenaconn

	resultConfig, err := expandAthenaResultConfiguration(d.Get("bucket").(string), d.Get("encryption_configuration").([]interface{}))
	if err != nil {
		return err
	}

	input := &athena.StartQueryExecutionInput{
		QueryString:         aws.String(fmt.Sprint("show databases;")),
		ResultConfiguration: resultConfig,
	}

	resp, err := conn.StartQueryExecution(input)
	if err != nil {
		return err
	}

	if err := executeAndExpectMatchingRow(*resp.QueryExecutionId, d.Get("name").(string), conn); err != nil {
		return err
	}
	return nil
}

func resourceAwsAthenaDatabaseUpdate(d *schema.ResourceData, meta interface{}) error {
	return resourceAwsAthenaDatabaseRead(d, meta)
}

func resourceAwsAthenaDatabaseDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).athenaconn

	resultConfig, err := expandAthenaResultConfiguration(d.Get("bucket").(string), d.Get("encryption_configuration").([]interface{}))
	if err != nil {
		return err
	}

	name := d.Get("name").(string)

	queryString := fmt.Sprintf("drop database `%s`", name)
	if d.Get("force_destroy").(bool) {
		queryString += " cascade"
	}
	queryString += ";"

	input := &athena.StartQueryExecutionInput{
		QueryString:         aws.String(queryString),
		ResultConfiguration: resultConfig,
	}

	resp, err := conn.StartQueryExecution(input)
	if err != nil {
		return err
	}

	if err := executeAndExpectNoRowsWhenDrop(*resp.QueryExecutionId, d, conn); err != nil {
		return err
	}
	return nil
}

func executeAndExpectNoRowsWhenCreate(qeid string, d *schema.ResourceData, conn *athena.Athena) error {
	rs, err := queryExecutionResult(qeid, conn)
	if err != nil {
		return err
	}
	if len(rs.Rows) != 0 {
		return fmt.Errorf("Athena create database, unexpected query result: %s", flattenAthenaResultSet(rs))
	}
	return nil
}

func executeAndExpectMatchingRow(qeid string, dbName string, conn *athena.Athena) error {
	rs, err := queryExecutionResult(qeid, conn)
	if err != nil {
		return err
	}
	for _, row := range rs.Rows {
		for _, datum := range row.Data {
			if *datum.VarCharValue == dbName {
				return nil
			}
		}
	}
	return fmt.Errorf("Athena not found database: %s, query result: %s", dbName, flattenAthenaResultSet(rs))
}

func executeAndExpectNoRowsWhenDrop(qeid string, d *schema.ResourceData, conn *athena.Athena) error {
	rs, err := queryExecutionResult(qeid, conn)
	if err != nil {
		return err
	}
	if len(rs.Rows) != 0 {
		return fmt.Errorf("Athena drop database, unexpected query result: %s", flattenAthenaResultSet(rs))
	}
	return nil
}

func queryExecutionResult(qeid string, conn *athena.Athena) (*athena.ResultSet, error) {
	executionStateConf := &resource.StateChangeConf{
		Pending:    []string{athena.QueryExecutionStateQueued, athena.QueryExecutionStateRunning},
		Target:     []string{athena.QueryExecutionStateSucceeded},
		Refresh:    queryExecutionStateRefreshFunc(qeid, conn),
		Timeout:    10 * time.Minute,
		Delay:      3 * time.Second,
		MinTimeout: 3 * time.Second,
	}
	_, err := executionStateConf.WaitForState()
	if err != nil {
		return nil, err
	}

	qrinput := &athena.GetQueryResultsInput{
		QueryExecutionId: aws.String(qeid),
	}
	resp, err := conn.GetQueryResults(qrinput)
	if err != nil {
		return nil, err
	}
	return resp.ResultSet, nil
}

func queryExecutionStateRefreshFunc(qeid string, conn *athena.Athena) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		input := &athena.GetQueryExecutionInput{
			QueryExecutionId: aws.String(qeid),
		}
		out, err := conn.GetQueryExecution(input)
		if err != nil {
			return nil, "failed", err
		}
		status := out.QueryExecution.Status
		if *status.State == athena.QueryExecutionStateFailed &&
			status.StateChangeReason != nil {
			err = fmt.Errorf("reason: %s", *status.StateChangeReason)
		}
		return out, *out.QueryExecution.Status.State, err
	}
}

func flattenAthenaResultSet(rs *athena.ResultSet) string {
	ss := make([]string, 0)
	for _, row := range rs.Rows {
		for _, datum := range row.Data {
			ss = append(ss, *datum.VarCharValue)
		}
	}
	return strings.Join(ss, "\n")
}
