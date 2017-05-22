package aws

import (
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/kinesisanalytics"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func shim(*schema.ResourceData, interface{}) error {
	return errors.New("func UPDATE Not implemented in resource_aws_kinesis_analytics.go")
}

func resourceAwsKinesisAnalytics() *schema.Resource {
	return &schema.Resource{

		Create: resourceAwsKinesisAnalyticsCreate,
		Read:   resourceAwsKinesisAnalyticsRead,
		Update: shim,
		Delete: resourceAwskinesisAnalyticsDelete,

		/*
			todo:
			some can trigger an Update instead of ForceNew
			be sure to add docs about the json record type making some optional fields into required
		*/
		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"application_description": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"application_code": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"inputs": {
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": { // NamePrefix
							Type:     schema.TypeString,
							Required: true,
						},
						"arn": { // KinesisStreamsInput.ResourceARN
							Type:     schema.TypeString,
							Required: true,
						},
						"role_arn": { // KinesisStreamsInput.RoleARN
							Type:     schema.TypeString,
							Required: true,
						},

						"record_format_type": { //RecordFormat.RecordFormatType ( JSON || CSV)
							Type:     schema.TypeString,
							Required: true,
						},

						"record_format_encoding": {
							Type:     schema.TypeString,
							Optional: true,
						},

						"record_row_path": { // MappingParameters.JSONMappingParameters.RecordRowPath
							Type:     schema.TypeString,
							Optional: true,
						},

						"record_row_delimiter": { //MappingParameters.CSVMappingParameters.RecordRowDelimiter
							Type:     schema.TypeString,
							Optional: true,
						},

						"record_column_delimiter": { //MappingParameters.CSVMappingParameters.RecordColumnDelimiter
							Type:     schema.TypeString,
							Optional: true,
						},

						"columns": { //InputSchema.RecordColumns
							Type:     schema.TypeSet,
							Required: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"name": {
										Type:     schema.TypeString,
										Required: true,
									},
									"sql_type": {
										Type:     schema.TypeString,
										Required: true,
									},
									"mapping": {
										Type:     schema.TypeString,
										Optional: true,
									},
								},
							},
						},
					},
				},
			},

			"outputs": {
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": { // Name
							Type:     schema.TypeString,
							Required: true,
						},
						"record_format_type": { // DestinationSchema.RecordFormatType ( JSON || CSV)
							Type:     schema.TypeString,
							Required: true,
						},
						"arn": { // KinesisStreamsOutput.ResourceARN || KinesisFirehoseOutput.ResourceARN
							Type:     schema.TypeString,
							Required: true,
						},
						"role_arn": { // KinesisStreamsOutput.RoleARN || KinesisFirehoseOutput.RoleARN
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},

			"arn": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"version_id": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"create_timestamp": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func discoverStreamType(arn string) (string, error) {

	streamType := ""

	if strings.Contains(arn, "aws:firehose") {
		streamType = "firehose"
	} else if strings.Contains(arn, "aws:kinesis") {
		streamType = "kinesis"
	}

	if streamType == "" {
		return "", fmt.Errorf("Error when attempting to determine stream type from AWS ARN. The arn (%s), does not appear to be kinesis nor firehose", arn)
	}

	return streamType, nil
}

func createOutputList(outputs []interface{}) ([]*kinesisanalytics.Output, error) {

	var outputStreams []*kinesisanalytics.Output

	for _, elem := range outputs {

		output := elem.(map[string]interface{})

		streamType, err := discoverStreamType(output["arn"].(string))

		if err != nil {
			return nil, err
		}

		if streamType == "kinesis" {
			outputStreams = append(outputStreams, &kinesisanalytics.Output{

				Name: aws.String(output["name"].(string)),

				DestinationSchema: &kinesisanalytics.DestinationSchema{
					RecordFormatType: aws.String(output["record_format_type"].(string)),
				},

				KinesisStreamsOutput: &kinesisanalytics.KinesisStreamsOutput{
					ResourceARN: aws.String(output["arn"].(string)),
					RoleARN:     aws.String(output["role_arn"].(string)),
				},
			})
		}
		if streamType == "firehose" {
			outputStreams = append(outputStreams, &kinesisanalytics.Output{

				Name: aws.String(output["name"].(string)),

				DestinationSchema: &kinesisanalytics.DestinationSchema{
					RecordFormatType: aws.String(output["record_format_type"].(string)),
				},

				KinesisFirehoseOutput: &kinesisanalytics.KinesisFirehoseOutput{
					ResourceARN: aws.String(output["arn"].(string)),
					RoleARN:     aws.String(output["role_arn"].(string)),
				},
			})
		}
	}

	return outputStreams, nil
}

func createInputList(inputs []interface{}) ([]*kinesisanalytics.Input, error) {

	var inputStreams []*kinesisanalytics.Input

	for _, elem := range inputs {

		input := elem.(map[string]interface{})

		var columns []*kinesisanalytics.RecordColumn

		cols := input["columns"].(*schema.Set).List()

		for _, c := range cols {

			col := c.(map[string]interface{})

			columns = append(columns, &kinesisanalytics.RecordColumn{
				Name:    aws.String(col["name"].(string)),
				SqlType: aws.String(col["sql_type"].(string)),
				Mapping: aws.String(col["mapping"].(string)),
			})
		}

		//todo: support firehose as inputs

		i := &kinesisanalytics.Input{

			NamePrefix: aws.String(input["name"].(string)),

			InputSchema: &kinesisanalytics.SourceSchema{

				RecordEncoding: aws.String(input["record_format_encoding"].(string)),

				RecordFormat: &kinesisanalytics.RecordFormat{
					RecordFormatType: aws.String(input["record_format_type"].(string)),
				},

				RecordColumns: columns,
			},

			KinesisStreamsInput: &kinesisanalytics.KinesisStreamsInput{
				ResourceARN: aws.String(input["arn"].(string)),
				RoleARN:     aws.String(input["role_arn"].(string)),
			},
		}

		if *i.InputSchema.RecordFormat.RecordFormatType == "JSON" {
			i.InputSchema.RecordFormat.MappingParameters = &kinesisanalytics.MappingParameters{
				JSONMappingParameters: &kinesisanalytics.JSONMappingParameters{
					RecordRowPath: aws.String(input["record_row_path"].(string)),
				},
			}
		} else if *i.InputSchema.RecordFormat.RecordFormatType == "CSV" {
			i.InputSchema.RecordFormat.MappingParameters = &kinesisanalytics.MappingParameters{
				CSVMappingParameters: &kinesisanalytics.CSVMappingParameters{
					RecordColumnDelimiter: aws.String(input["record_column_delimiter"].(string)),
					RecordRowDelimiter:    aws.String(input["record_row_delimiter"].(string)),
				},
			}
		} else {
			return nil, fmt.Errorf("format must be either 'JSON or CSV'. you gave an unsupported record format type: %s", i.InputSchema.RecordFormat.RecordFormatType)
		}

		inputStreams = append(inputStreams, i)
	}

	return inputStreams, nil
}

func resourceAwsKinesisAnalyticsCreate(d *schema.ResourceData, meta interface{}) error {

	conn := meta.(*AWSClient).kinesisanalyticsconn

	name := d.Get("name").(string)
	appDesc := d.Get("application_description").(string)
	appCode := d.Get("application_code").(string)
	inputs := d.Get("inputs").(*schema.Set).List()
	outputs := d.Get("outputs").(*schema.Set).List()

	inputStreams, err := createInputList(inputs)

	if err != nil {
		return err
	}

	outputStreams, err := createOutputList(outputs)

	if err != nil {
		return err
	}

	//fmt.Printf("KA.INPUTS:  %T\n\n", inputs)
	//fmt.Printf("KA.INPUTS:  %+v\n\n", inputs)
	//
	//element := inputs[0].(map[string]interface{})
	//fmt.Printf("element:  %T\n\n", element)
	//fmt.Printf("property access:  %+v\n\n", element["name"].(string))

	createOpts := &kinesisanalytics.CreateApplicationInput{
		ApplicationName:        aws.String(name),
		ApplicationDescription: aws.String(appDesc),
		ApplicationCode:        aws.String(appCode),
		Inputs:                 inputStreams,
		Outputs:                outputStreams,
	}

	_, err = conn.CreateApplication(createOpts)

	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok {
			return fmt.Errorf("[WARN] Error creating Kinesis Analytics Application: \"%s\", code: \"%s\"", awsErr.Message(), awsErr.Code())
		}
		return err
	}

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"CREATING"},
		Target:     []string{"READY"},
		Refresh:    applicationStateRefreshFunc(conn, name),
		Timeout:    3 * time.Minute,
		Delay:      5 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	state, err := stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf(
			"Error waiting for Kinesis Analytics Application (%s) to become active: %s",
			name, err)
	}

	s := state.(*KinesisAnalyticsState)
	d.SetId(s.arn)
	d.Set("arn", s.arn)
	d.Set("create_timestamp", strconv.FormatInt(s.createTimestamp, 10))
	d.Set("application_description", s.description)
	d.Set("application_code", s.code)
	d.Set("inputs", s.inputs)
	d.Set("outputs", s.outputs)

	return nil
}

func resourceAwsKinesisAnalyticsRead(d *schema.ResourceData, meta interface{}) error {

	conn := meta.(*AWSClient).kinesisanalyticsconn
	name := d.Get("name").(string)

	_, err := readKinesisAnalyticsState(conn, name)

	if err != nil {
		return err
	}

	return nil
}

func resourceAwskinesisAnalyticsDelete(d *schema.ResourceData, meta interface{}) error {

	conn := meta.(*AWSClient).kinesisanalyticsconn

	name := d.Get("name").(string)
	cerealizedTime := d.Get("create_timestamp").(string)
	ct, _ := strconv.ParseInt(cerealizedTime, 10, 64)

	createTime := time.Unix(ct, 0)

	input := &kinesisanalytics.DeleteApplicationInput{
		ApplicationName: aws.String(name),
		CreateTimestamp: aws.Time(createTime),
	}

	log.Printf("[DEBUG] Deleting Kinesis Analytics Application: %s", d.Id())

	_, err := conn.DeleteApplication(input)

	if err != nil {
		return fmt.Errorf("Error deleting Kinesis Analytics Application: %s", err)
	}

	return nil
}

type KinesisAnalyticsState struct {
	description     string
	code            string
	arn             string
	createTimestamp int64
	status          string
	inputs          []map[string]interface{}
	outputs         []map[string]interface{}
}

func readKinesisAnalyticsState(conn *kinesisanalytics.KinesisAnalytics, name string) (*KinesisAnalyticsState, error) {

	describeOpts := &kinesisanalytics.DescribeApplicationInput{
		ApplicationName: aws.String(name),
	}

	state := &KinesisAnalyticsState{}

	res, err := conn.DescribeApplication(describeOpts)

	if err != nil {
		return nil, err
	}

	state.description = aws.StringValue(res.ApplicationDetail.ApplicationDescription)

	state.code = aws.StringValue(res.ApplicationDetail.ApplicationCode)

	state.arn = aws.StringValue(res.ApplicationDetail.ApplicationARN)

	state.createTimestamp = aws.Time(*res.ApplicationDetail.CreateTimestamp).Unix()

	state.status = aws.StringValue(res.ApplicationDetail.ApplicationStatus)

	state.inputs = inputDescriptionToMap(res.ApplicationDetail.InputDescriptions)

	state.outputs = outputDescriptionToMap(res.ApplicationDetail.OutputDescriptions)

	return state, nil
}

func inputDescriptionToMap(inputs []*kinesisanalytics.InputDescription) []map[string]interface{} {

	var inputMap []map[string]interface{}

	for _, input := range inputs {

		i := map[string]interface{}{
			"name":                   aws.StringValue(input.NamePrefix),
			"record_format_type":     aws.StringValue(input.InputSchema.RecordFormat.RecordFormatType),
			"record_format_encoding": aws.StringValue(input.InputSchema.RecordEncoding),
		}

		if input.InputSchema.RecordFormat.MappingParameters.JSONMappingParameters != nil {
			i["record_row_path"] = aws.StringValue(input.InputSchema.RecordFormat.MappingParameters.JSONMappingParameters.RecordRowPath)
		} else if input.InputSchema.RecordFormat.MappingParameters.CSVMappingParameters != nil {
			i["record_row_delimiter"] = aws.StringValue(input.InputSchema.RecordFormat.MappingParameters.CSVMappingParameters.RecordRowDelimiter)
			i["record_column_delimiter"] = aws.StringValue(input.InputSchema.RecordFormat.MappingParameters.CSVMappingParameters.RecordColumnDelimiter)
		}

		if input.KinesisFirehoseInputDescription != nil {
			i["arn"] = aws.StringValue(input.KinesisFirehoseInputDescription.ResourceARN)
			i["role_arn"] = aws.StringValue(input.KinesisFirehoseInputDescription.RoleARN)
		} else if input.KinesisStreamsInputDescription != nil {
			i["arn"] = aws.StringValue(input.KinesisStreamsInputDescription.ResourceARN)
			i["role_arn"] = aws.StringValue(input.KinesisStreamsInputDescription.RoleARN)
		}

		i["columns"] = make([]map[string]interface{}, len(input.InputSchema.RecordColumns))

		for c, column := range input.InputSchema.RecordColumns {

			i["columns"].([]map[string]interface{})[c] = map[string]interface{}{
				"name":     aws.StringValue(column.Name),
				"sql_type": aws.StringValue(column.SqlType),
				"mapping":  aws.StringValue(column.Mapping),
			}
		}

		inputMap = append(inputMap, i)
	}

	return inputMap
}

func outputDescriptionToMap(outputs []*kinesisanalytics.OutputDescription) []map[string]interface{} {

	var outputMap []map[string]interface{}

	for _, output := range outputs {
		o := map[string]interface{}{
			"name":               aws.StringValue(output.Name),
			"record_format_type": aws.StringValue(output.DestinationSchema.RecordFormatType),
		}

		if output.KinesisFirehoseOutputDescription != nil {
			o["arn"] = aws.StringValue(output.KinesisFirehoseOutputDescription.ResourceARN)
			o["role_arn"] = aws.StringValue(output.KinesisFirehoseOutputDescription.RoleARN)
		} else if output.KinesisStreamsOutputDescription != nil {
			o["arn"] = aws.StringValue(output.KinesisStreamsOutputDescription.ResourceARN)
			o["role_arn"] = aws.StringValue(output.KinesisStreamsOutputDescription.RoleARN)
		}

		outputMap = append(outputMap, o)
	}

	return outputMap
}

func applicationStateRefreshFunc(conn *kinesisanalytics.KinesisAnalytics, name string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		state, err := readKinesisAnalyticsState(conn, name)
		if err != nil {
			if awsErr, ok := err.(awserr.Error); ok {
				if awsErr.Code() == "ResourceNotFoundException" {
					return 42, "DESTROYED", nil
				}
				return nil, awsErr.Code(), err
			}
			return nil, "failed", err
		}

		return state, state.status, nil
	}
}
