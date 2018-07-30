package aws

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	dms "github.com/aws/aws-sdk-go/service/databasemigrationservice"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
)

func resourceAwsDmsEndpoint() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsDmsEndpointCreate,
		Read:   resourceAwsDmsEndpointRead,
		Update: resourceAwsDmsEndpointUpdate,
		Delete: resourceAwsDmsEndpointDelete,

		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"certificate_arn": {
				Type:         schema.TypeString,
				Computed:     true,
				Optional:     true,
				ValidateFunc: validateArn,
			},
			"database_name": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"endpoint_arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"endpoint_id": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validateDmsEndpointId,
			},
			"service_access_role": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"endpoint_type": {
				Type:     schema.TypeString,
				Required: true,
				ValidateFunc: validation.StringInSlice([]string{
					dms.ReplicationEndpointTypeValueSource,
					dms.ReplicationEndpointTypeValueTarget,
				}, false),
			},
			"engine_name": {
				Type:     schema.TypeString,
				Required: true,
				ValidateFunc: validation.StringInSlice([]string{
					"mysql",
					"oracle",
					"postgres",
					"dynamodb",
					"mariadb",
					"aurora",
					"aurora-postgresql",
					"redshift",
					"sybase",
					"sqlserver",
					"mongodb",
					"s3",
					"azuredb",
				}, false),
			},
			"extra_connection_attributes": {
				Type:     schema.TypeString,
				Computed: true,
				Optional: true,
			},
			"kms_key_arn": {
				Type:         schema.TypeString,
				Computed:     true,
				Optional:     true,
				ForceNew:     true,
				ValidateFunc: validateArn,
			},
			"password": {
				Type:      schema.TypeString,
				Optional:  true,
				Sensitive: true,
			},
			"port": {
				Type:     schema.TypeInt,
				Optional: true,
			},
			"server_name": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"ssl_mode": {
				Type:     schema.TypeString,
				Computed: true,
				Optional: true,
				ValidateFunc: validation.StringInSlice([]string{
					dms.DmsSslModeValueNone,
					dms.DmsSslModeValueRequire,
					dms.DmsSslModeValueVerifyCa,
					dms.DmsSslModeValueVerifyFull,
				}, false),
			},
			"tags": {
				Type:     schema.TypeMap,
				Optional: true,
			},
			"username": {
				Type:     schema.TypeString,
				Optional: true,
			},
			// With default values as per https://docs.aws.amazon.com/dms/latest/userguide/CHAP_Source.MongoDB.html
			"mongodb_settings": {
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
					if old == "1" && new == "0" {
						return true
					}
					return false
				},
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"auth_type": {
							Type:     schema.TypeString,
							Optional: true,
							Default:  "PASSWORD",
						},
						"auth_mechanism": {
							Type:     schema.TypeString,
							Optional: true,
							Default:  "DEFAULT",
						},
						"nesting_level": {
							Type:     schema.TypeString,
							Optional: true,
							Default:  "NONE",
						},
						"extract_doc_id": {
							Type:     schema.TypeString,
							Optional: true,
							Default:  "false",
						},
						"docs_to_investigate": {
							Type:     schema.TypeString,
							Optional: true,
							Default:  "1000",
						},
						"auth_source": {
							Type:     schema.TypeString,
							Optional: true,
							Default:  "admin",
						},
					},
				},
			},
			"s3_settings": {
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
					if old == "1" && new == "0" {
						return true
					}
					return false
				},
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"service_access_role_arn": {
							Type:     schema.TypeString,
							Optional: true,
							Default:  "",
						},
						"external_table_definition": {
							Type:     schema.TypeString,
							Optional: true,
							Default:  "",
						},
						"csv_row_delimiter": {
							Type:     schema.TypeString,
							Optional: true,
							Default:  "\\n",
						},
						"csv_delimiter": {
							Type:     schema.TypeString,
							Optional: true,
							Default:  ",",
						},
						"bucket_folder": {
							Type:     schema.TypeString,
							Optional: true,
							Default:  "",
						},
						"bucket_name": {
							Type:     schema.TypeString,
							Optional: true,
							Default:  "",
						},
						"compression_type": {
							Type:     schema.TypeString,
							Optional: true,
							Default:  "NONE",
						},
					},
				},
			},
		},
	}
}

func resourceAwsDmsEndpointCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).dmsconn

	request := &dms.CreateEndpointInput{
		EndpointIdentifier: aws.String(d.Get("endpoint_id").(string)),
		EndpointType:       aws.String(d.Get("endpoint_type").(string)),
		EngineName:         aws.String(d.Get("engine_name").(string)),
		Tags:               dmsTagsFromMap(d.Get("tags").(map[string]interface{})),
	}

	switch d.Get("engine_name").(string) {
	// if dynamodb then add required params
	case "dynamodb":
		request.DynamoDbSettings = &dms.DynamoDbSettings{
			ServiceAccessRoleArn: aws.String(d.Get("service_access_role").(string)),
		}
	case "mongodb":
		request.MongoDbSettings = &dms.MongoDbSettings{
			Username:     aws.String(d.Get("username").(string)),
			Password:     aws.String(d.Get("password").(string)),
			ServerName:   aws.String(d.Get("server_name").(string)),
			Port:         aws.Int64(int64(d.Get("port").(int))),
			DatabaseName: aws.String(d.Get("database_name").(string)),
			KmsKeyId:     aws.String(d.Get("kms_key_arn").(string)),

			AuthType:          aws.String(d.Get("mongodb_settings.0.auth_type").(string)),
			AuthMechanism:     aws.String(d.Get("mongodb_settings.0.auth_mechanism").(string)),
			NestingLevel:      aws.String(d.Get("mongodb_settings.0.nesting_level").(string)),
			ExtractDocId:      aws.String(d.Get("mongodb_settings.0.extract_doc_id").(string)),
			DocsToInvestigate: aws.String(d.Get("mongodb_settings.0.docs_to_investigate").(string)),
			AuthSource:        aws.String(d.Get("mongodb_settings.0.auth_source").(string)),
		}

		// Set connection info in top-level namespace as well
		request.Username = aws.String(d.Get("username").(string))
		request.Password = aws.String(d.Get("password").(string))
		request.ServerName = aws.String(d.Get("server_name").(string))
		request.Port = aws.Int64(int64(d.Get("port").(int)))
		request.DatabaseName = aws.String(d.Get("database_name").(string))
	case "s3":
		request.S3Settings = &dms.S3Settings{
			ServiceAccessRoleArn:    aws.String(d.Get("s3_settings.0.service_access_role_arn").(string)),
			ExternalTableDefinition: aws.String(d.Get("s3_settings.0.external_table_definition").(string)),
			CsvRowDelimiter:         aws.String(d.Get("s3_settings.0.csv_row_delimiter").(string)),
			CsvDelimiter:            aws.String(d.Get("s3_settings.0.csv_delimiter").(string)),
			BucketFolder:            aws.String(d.Get("s3_settings.0.bucket_folder").(string)),
			BucketName:              aws.String(d.Get("s3_settings.0.bucket_name").(string)),
			CompressionType:         aws.String(d.Get("s3_settings.0.compression_type").(string)),
		}
	default:
		request.Password = aws.String(d.Get("password").(string))
		request.Port = aws.Int64(int64(d.Get("port").(int)))
		request.ServerName = aws.String(d.Get("server_name").(string))
		request.Username = aws.String(d.Get("username").(string))

		if v, ok := d.GetOk("database_name"); ok {
			request.DatabaseName = aws.String(v.(string))
		}
		if v, ok := d.GetOk("extra_connection_attributes"); ok {
			request.ExtraConnectionAttributes = aws.String(v.(string))
		}
		if v, ok := d.GetOk("kms_key_arn"); ok {
			request.KmsKeyId = aws.String(v.(string))
		}
	}

	if v, ok := d.GetOk("certificate_arn"); ok {
		request.CertificateArn = aws.String(v.(string))
	}
	if v, ok := d.GetOk("ssl_mode"); ok {
		request.SslMode = aws.String(v.(string))
	}

	log.Println("[DEBUG] DMS create endpoint:", request)

	err := resource.Retry(5*time.Minute, func() *resource.RetryError {
		if _, err := conn.CreateEndpoint(request); err != nil {
			if awserr, ok := err.(awserr.Error); ok {
				switch awserr.Code() {
				case "AccessDeniedFault":
					return resource.RetryableError(awserr)
				}
			}
			// Didn't recognize the error, so shouldn't retry.
			return resource.NonRetryableError(err)
		}
		// Successful delete
		return nil
	})
	if err != nil {
		return err
	}

	d.SetId(d.Get("endpoint_id").(string))
	return resourceAwsDmsEndpointRead(d, meta)
}

func resourceAwsDmsEndpointRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).dmsconn

	response, err := conn.DescribeEndpoints(&dms.DescribeEndpointsInput{
		Filters: []*dms.Filter{
			{
				Name:   aws.String("endpoint-id"),
				Values: []*string{aws.String(d.Id())}, // Must use d.Id() to work with import.
			},
		},
	})
	if err != nil {
		if dmserr, ok := err.(awserr.Error); ok && dmserr.Code() == "ResourceNotFoundFault" {
			log.Printf("[DEBUG] DMS Replication Endpoint %q Not Found", d.Id())
			d.SetId("")
			return nil
		}
		return err
	}

	err = resourceAwsDmsEndpointSetState(d, response.Endpoints[0])
	if err != nil {
		return err
	}

	tagsResp, err := conn.ListTagsForResource(&dms.ListTagsForResourceInput{
		ResourceArn: aws.String(d.Get("endpoint_arn").(string)),
	})
	if err != nil {
		return err
	}
	return d.Set("tags", dmsTagsToMap(tagsResp.TagList))
}

func resourceAwsDmsEndpointUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).dmsconn

	request := &dms.ModifyEndpointInput{
		EndpointArn: aws.String(d.Get("endpoint_arn").(string)),
	}
	hasChanges := false

	if d.HasChange("endpoint_type") {
		request.EndpointType = aws.String(d.Get("endpoint_type").(string))
		hasChanges = true
	}

	if d.HasChange("certificate_arn") {
		request.CertificateArn = aws.String(d.Get("certificate_arn").(string))
		hasChanges = true
	}

	if d.HasChange("service_access_role") {
		request.DynamoDbSettings = &dms.DynamoDbSettings{
			ServiceAccessRoleArn: aws.String(d.Get("service_access_role").(string)),
		}
		hasChanges = true
	}

	if d.HasChange("endpoint_type") {
		request.EndpointType = aws.String(d.Get("endpoint_type").(string))
		hasChanges = true
	}

	if d.HasChange("engine_name") {
		request.EngineName = aws.String(d.Get("engine_name").(string))
		hasChanges = true
	}

	if d.HasChange("extra_connection_attributes") {
		request.ExtraConnectionAttributes = aws.String(d.Get("extra_connection_attributes").(string))
		hasChanges = true
	}

	if d.HasChange("ssl_mode") {
		request.SslMode = aws.String(d.Get("ssl_mode").(string))
		hasChanges = true
	}

	if d.HasChange("tags") {
		err := dmsSetTags(d.Get("endpoint_arn").(string), d, meta)
		if err != nil {
			return err
		}
	}

	switch d.Get("engine_name").(string) {
	case "dynamodb":
		if d.HasChange("service_access_role") {
			request.DynamoDbSettings = &dms.DynamoDbSettings{
				ServiceAccessRoleArn: aws.String(d.Get("service_access_role").(string)),
			}
			hasChanges = true
		}
	case "mongodb":
		if d.HasChange("username") ||
			d.HasChange("password") ||
			d.HasChange("server_name") ||
			d.HasChange("port") ||
			d.HasChange("database_name") ||
			d.HasChange("mongodb_settings.0.auth_type") ||
			d.HasChange("mongodb_settings.0.auth_mechanism") ||
			d.HasChange("mongodb_settings.0.nesting_level") ||
			d.HasChange("mongodb_settings.0.extract_doc_id") ||
			d.HasChange("mongodb_settings.0.docs_to_investigate") ||
			d.HasChange("mongodb_settings.0.auth_source") {
			request.MongoDbSettings = &dms.MongoDbSettings{
				Username:     aws.String(d.Get("username").(string)),
				Password:     aws.String(d.Get("password").(string)),
				ServerName:   aws.String(d.Get("server_name").(string)),
				Port:         aws.Int64(int64(d.Get("port").(int))),
				DatabaseName: aws.String(d.Get("database_name").(string)),
				KmsKeyId:     aws.String(d.Get("kms_key_arn").(string)),

				AuthType:          aws.String(d.Get("mongodb_settings.0.auth_type").(string)),
				AuthMechanism:     aws.String(d.Get("mongodb_settings.0.auth_mechanism").(string)),
				NestingLevel:      aws.String(d.Get("mongodb_settings.0.nesting_level").(string)),
				ExtractDocId:      aws.String(d.Get("mongodb_settings.0.extract_doc_id").(string)),
				DocsToInvestigate: aws.String(d.Get("mongodb_settings.0.docs_to_investigate").(string)),
				AuthSource:        aws.String(d.Get("mongodb_settings.0.auth_source").(string)),
			}
			request.EngineName = aws.String(d.Get("engine_name").(string)) // Must be included (should be 'mongodb')

			// Update connection info in top-level namespace as well
			request.Username = aws.String(d.Get("username").(string))
			request.Password = aws.String(d.Get("password").(string))
			request.ServerName = aws.String(d.Get("server_name").(string))
			request.Port = aws.Int64(int64(d.Get("port").(int)))
			request.DatabaseName = aws.String(d.Get("database_name").(string))

			hasChanges = true
		}
	case "s3":
		if d.HasChange("s3_settings.0.service_access_role_arn") ||
			d.HasChange("s3_settings.0.external_table_definition") ||
			d.HasChange("s3_settings.0.csv_row_delimiter") ||
			d.HasChange("s3_settings.0.csv_delimiter") ||
			d.HasChange("s3_settings.0.bucket_folder") ||
			d.HasChange("s3_settings.0.bucket_name") ||
			d.HasChange("s3_settings.0.compression_type") {
			request.S3Settings = &dms.S3Settings{
				ServiceAccessRoleArn:    aws.String(d.Get("s3_settings.0.service_access_role_arn").(string)),
				ExternalTableDefinition: aws.String(d.Get("s3_settings.0.external_table_definition").(string)),
				CsvRowDelimiter:         aws.String(d.Get("s3_settings.0.csv_row_delimiter").(string)),
				CsvDelimiter:            aws.String(d.Get("s3_settings.0.csv_delimiter").(string)),
				BucketFolder:            aws.String(d.Get("s3_settings.0.bucket_folder").(string)),
				BucketName:              aws.String(d.Get("s3_settings.0.bucket_name").(string)),
				CompressionType:         aws.String(d.Get("s3_settings.0.compression_type").(string)),
			}
			request.EngineName = aws.String(d.Get("engine_name").(string)) // Must be included (should be 's3')
			hasChanges = true
		}
	default:
		if d.HasChange("database_name") {
			request.DatabaseName = aws.String(d.Get("database_name").(string))
			hasChanges = true
		}

		if d.HasChange("password") {
			request.Password = aws.String(d.Get("password").(string))
			hasChanges = true
		}

		if d.HasChange("port") {
			request.Port = aws.Int64(int64(d.Get("port").(int)))
			hasChanges = true
		}

		if d.HasChange("server_name") {
			request.ServerName = aws.String(d.Get("server_name").(string))
			hasChanges = true
		}

		if d.HasChange("username") {
			request.Username = aws.String(d.Get("username").(string))
			hasChanges = true
		}
	}

	if hasChanges {
		log.Println("[DEBUG] DMS update endpoint:", request)

		_, err := conn.ModifyEndpoint(request)
		if err != nil {
			return err
		}

		return resourceAwsDmsEndpointRead(d, meta)
	}

	return nil
}

func resourceAwsDmsEndpointDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).dmsconn

	request := &dms.DeleteEndpointInput{
		EndpointArn: aws.String(d.Get("endpoint_arn").(string)),
	}

	log.Printf("[DEBUG] DMS delete endpoint: %#v", request)

	_, err := conn.DeleteEndpoint(request)
	return err
}

func resourceAwsDmsEndpointSetState(d *schema.ResourceData, endpoint *dms.Endpoint) error {
	d.SetId(*endpoint.EndpointIdentifier)

	d.Set("certificate_arn", endpoint.CertificateArn)
	d.Set("endpoint_arn", endpoint.EndpointArn)
	d.Set("endpoint_id", endpoint.EndpointIdentifier)
	// For some reason the AWS API only accepts lowercase type but returns it as uppercase
	d.Set("endpoint_type", strings.ToLower(*endpoint.EndpointType))
	d.Set("engine_name", endpoint.EngineName)

	switch *endpoint.EngineName {
	case "dynamodb":
		if endpoint.DynamoDbSettings != nil {
			d.Set("service_access_role", endpoint.DynamoDbSettings.ServiceAccessRoleArn)
		} else {
			d.Set("service_access_role", "")
		}
	case "mongodb":
		if endpoint.MongoDbSettings != nil {
			d.Set("username", endpoint.MongoDbSettings.Username)
			d.Set("server_name", endpoint.MongoDbSettings.ServerName)
			d.Set("port", endpoint.MongoDbSettings.Port)
			d.Set("database_name", endpoint.MongoDbSettings.DatabaseName)
		} else {
			d.Set("username", endpoint.Username)
			d.Set("server_name", endpoint.ServerName)
			d.Set("port", endpoint.Port)
			d.Set("database_name", endpoint.DatabaseName)
		}
		if err := d.Set("mongodb_settings", flattenDmsMongoDbSettings(endpoint.MongoDbSettings)); err != nil {
			return fmt.Errorf("Error setting mongodb_settings for DMS: %s", err)
		}
	case "s3":
		if err := d.Set("s3_settings", flattenDmsS3Settings(endpoint.S3Settings)); err != nil {
			return fmt.Errorf("Error setting s3_settings for DMS: %s", err)
		}
	default:
		d.Set("database_name", endpoint.DatabaseName)
		d.Set("extra_connection_attributes", endpoint.ExtraConnectionAttributes)
		d.Set("port", endpoint.Port)
		d.Set("server_name", endpoint.ServerName)
		d.Set("username", endpoint.Username)
	}

	d.Set("kms_key_arn", endpoint.KmsKeyId)
	d.Set("ssl_mode", endpoint.SslMode)

	return nil
}

func flattenDmsMongoDbSettings(settings *dms.MongoDbSettings) []map[string]interface{} {
	if settings == nil {
		return []map[string]interface{}{}
	}

	m := map[string]interface{}{
		"auth_type":           aws.StringValue(settings.AuthType),
		"auth_mechanism":      aws.StringValue(settings.AuthMechanism),
		"nesting_level":       aws.StringValue(settings.NestingLevel),
		"extract_doc_id":      aws.StringValue(settings.ExtractDocId),
		"docs_to_investigate": aws.StringValue(settings.DocsToInvestigate),
		"auth_source":         aws.StringValue(settings.AuthSource),
	}

	return []map[string]interface{}{m}
}

func flattenDmsS3Settings(settings *dms.S3Settings) []map[string]interface{} {
	if settings == nil {
		return []map[string]interface{}{}
	}

	m := map[string]interface{}{
		"service_access_role_arn":   aws.StringValue(settings.ServiceAccessRoleArn),
		"external_table_definition": aws.StringValue(settings.ExternalTableDefinition),
		"csv_row_delimiter":         aws.StringValue(settings.CsvRowDelimiter),
		"csv_delimiter":             aws.StringValue(settings.CsvDelimiter),
		"bucket_folder":             aws.StringValue(settings.BucketFolder),
		"bucket_name":               aws.StringValue(settings.BucketName),
		"compression_type":          aws.StringValue(settings.CompressionType),
	}

	return []map[string]interface{}{m}
}
