package azure

import (
	"fmt"

	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/utils"

	"github.com/hashicorp/terraform/helper/validation"

	"github.com/Azure/azure-sdk-for-go/services/streamanalytics/mgmt/2016-03-01/streamanalytics"
	"github.com/hashicorp/terraform/helper/schema"
)

func SchemaStreamAnalyticsOutputSerialization() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		Required: true,
		MaxItems: 1,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"type": {
					Type:     schema.TypeString,
					Required: true,
					ValidateFunc: validation.StringInSlice([]string{
						string(streamanalytics.TypeAvro),
						string(streamanalytics.TypeCsv),
						string(streamanalytics.TypeJSON),
					}, false),
				},

				"field_delimiter": {
					Type:     schema.TypeString,
					Optional: true,
					ValidateFunc: validation.StringInSlice([]string{
						" ",
						",",
						"	",
						"|",
						";",
					}, false),
				},

				"encoding": {
					Type:     schema.TypeString,
					Optional: true,
					ValidateFunc: validation.StringInSlice([]string{
						string(streamanalytics.UTF8),
					}, false),
				},

				"format": {
					Type:     schema.TypeString,
					Optional: true,
					ValidateFunc: validation.StringInSlice([]string{
						string(streamanalytics.Array),
						string(streamanalytics.LineSeparated),
					}, false),
				},
			},
		},
	}
}

func ExpandStreamAnalyticsOutputSerialization(input []interface{}) (streamanalytics.BasicSerialization, error) {
	v := input[0].(map[string]interface{})

	outputType := streamanalytics.Type(v["type"].(string))
	encoding := v["encoding"].(string)
	fieldDelimiter := v["field_delimiter"].(string)
	format := v["format"].(string)

	switch outputType {
	case streamanalytics.TypeAvro:
		if encoding != "" {
			return nil, fmt.Errorf("`encoding` cannot be set when `type` is set to `Avro`")
		}
		if fieldDelimiter != "" {
			return nil, fmt.Errorf("`field_delimiter` cannot be set when `type` is set to `Avro`")
		}
		if format != "" {
			return nil, fmt.Errorf("`format` cannot be set when `type` is set to `Avro`")
		}
		return streamanalytics.AvroSerialization{
			Type:       streamanalytics.TypeAvro,
			Properties: map[string]interface{}{},
		}, nil

	case streamanalytics.TypeCsv:
		if encoding == "" {
			return nil, fmt.Errorf("`encoding` must be specified when `type` is set to `Csv`")
		}
		if fieldDelimiter == "" {
			return nil, fmt.Errorf("`field_delimiter` must be set when `type` is set to `Csv`")
		}
		if format != "" {
			return nil, fmt.Errorf("`format` cannot be set when `type` is set to `Csv`")
		}
		return streamanalytics.CsvSerialization{
			Type: streamanalytics.TypeCsv,
			CsvSerializationProperties: &streamanalytics.CsvSerializationProperties{
				Encoding:       streamanalytics.Encoding(encoding),
				FieldDelimiter: utils.String(fieldDelimiter),
			},
		}, nil

	case streamanalytics.TypeJSON:
		if encoding == "" {
			return nil, fmt.Errorf("`encoding` must be specified when `type` is set to `Json`")
		}
		if format == "" {
			return nil, fmt.Errorf("`format` must be specified when `type` is set to `Json`")
		}
		if fieldDelimiter != "" {
			return nil, fmt.Errorf("`field_delimiter` cannot be set when `type` is set to `Json`")
		}

		return streamanalytics.JSONSerialization{
			Type: streamanalytics.TypeJSON,
			JSONSerializationProperties: &streamanalytics.JSONSerializationProperties{
				Encoding: streamanalytics.Encoding(encoding),
				Format:   streamanalytics.JSONOutputSerializationFormat(format),
			},
		}, nil
	}

	return nil, fmt.Errorf("Unsupported Output Type %q", outputType)
}

func FlattenStreamAnalyticsOutputSerialization(input streamanalytics.BasicSerialization) []interface{} {
	var encoding string
	var outputType string
	var fieldDelimiter string
	var format string

	if _, ok := input.AsAvroSerialization(); ok {
		outputType = string(streamanalytics.TypeAvro)
	}

	if v, ok := input.AsCsvSerialization(); ok {
		if props := v.CsvSerializationProperties; props != nil {
			encoding = string(props.Encoding)
			if props.FieldDelimiter != nil {
				fieldDelimiter = *props.FieldDelimiter
			}
		}

		outputType = string(streamanalytics.TypeCsv)
	}

	if v, ok := input.AsJSONSerialization(); ok {
		if props := v.JSONSerializationProperties; props != nil {
			encoding = string(props.Encoding)
			format = string(props.Format)
		}

		outputType = string(streamanalytics.TypeJSON)
	}
	return []interface{}{
		map[string]interface{}{
			"encoding":        encoding,
			"type":            outputType,
			"format":          format,
			"field_delimiter": fieldDelimiter,
		},
	}
}
