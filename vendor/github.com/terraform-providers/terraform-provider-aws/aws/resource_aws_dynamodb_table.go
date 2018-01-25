package aws

import (
	"bytes"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/hashicorp/terraform/helper/hashcode"
)

// Number of times to retry if a throttling-related exception occurs
const DYNAMODB_MAX_THROTTLE_RETRIES = 5

// How long to sleep when a throttle-event happens
const DYNAMODB_THROTTLE_SLEEP = 5 * time.Second

// How long to sleep if a limit-exceeded event happens
const DYNAMODB_LIMIT_EXCEEDED_SLEEP = 10 * time.Second

// A number of these are marked as computed because if you don't
// provide a value, DynamoDB will provide you with defaults (which are the
// default values specified below)
func resourceAwsDynamoDbTable() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsDynamoDbTableCreate,
		Read:   resourceAwsDynamoDbTableRead,
		Update: resourceAwsDynamoDbTableUpdate,
		Delete: resourceAwsDynamoDbTableDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		SchemaVersion: 1,
		MigrateState:  resourceAwsDynamoDbTableMigrateState,

		Schema: map[string]*schema.Schema{
			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"hash_key": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"range_key": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"write_capacity": {
				Type:     schema.TypeInt,
				Required: true,
			},
			"read_capacity": {
				Type:     schema.TypeInt,
				Required: true,
			},
			"attribute": {
				Type:     schema.TypeSet,
				Required: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:     schema.TypeString,
							Required: true,
						},
						"type": {
							Type:     schema.TypeString,
							Required: true,
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
			"ttl": {
				Type:     schema.TypeSet,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"attribute_name": {
							Type:     schema.TypeString,
							Required: true,
						},
						"enabled": {
							Type:     schema.TypeBool,
							Required: true,
						},
					},
				},
			},
			"local_secondary_index": {
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:     schema.TypeString,
							Required: true,
						},
						"range_key": {
							Type:     schema.TypeString,
							Required: true,
						},
						"projection_type": {
							Type:     schema.TypeString,
							Required: true,
						},
						"non_key_attributes": {
							Type:     schema.TypeList,
							Optional: true,
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
			"global_secondary_index": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:     schema.TypeString,
							Required: true,
						},
						"write_capacity": {
							Type:     schema.TypeInt,
							Required: true,
						},
						"read_capacity": {
							Type:     schema.TypeInt,
							Required: true,
						},
						"hash_key": {
							Type:     schema.TypeString,
							Required: true,
						},
						"range_key": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"projection_type": {
							Type:     schema.TypeString,
							Required: true,
						},
						"non_key_attributes": {
							Type:     schema.TypeList,
							Optional: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},
					},
				},
			},
			"stream_enabled": {
				Type:     schema.TypeBool,
				Optional: true,
				Computed: true,
			},
			"stream_view_type": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				StateFunc: func(v interface{}) string {
					value := v.(string)
					return strings.ToUpper(value)
				},
				ValidateFunc: validateStreamViewType,
			},
			"stream_arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"stream_label": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"tags": tagsSchema(),
		},
	}
}

func resourceAwsDynamoDbTableCreate(d *schema.ResourceData, meta interface{}) error {
	dynamodbconn := meta.(*AWSClient).dynamodbconn

	name := d.Get("name").(string)

	log.Printf("[DEBUG] DynamoDB table create: %s", name)

	throughput := &dynamodb.ProvisionedThroughput{
		ReadCapacityUnits:  aws.Int64(int64(d.Get("read_capacity").(int))),
		WriteCapacityUnits: aws.Int64(int64(d.Get("write_capacity").(int))),
	}

	hash_key_name := d.Get("hash_key").(string)
	keyschema := []*dynamodb.KeySchemaElement{
		{
			AttributeName: aws.String(hash_key_name),
			KeyType:       aws.String("HASH"),
		},
	}

	if range_key, ok := d.GetOk("range_key"); ok {
		range_schema_element := &dynamodb.KeySchemaElement{
			AttributeName: aws.String(range_key.(string)),
			KeyType:       aws.String("RANGE"),
		}
		keyschema = append(keyschema, range_schema_element)
	}

	req := &dynamodb.CreateTableInput{
		TableName:             aws.String(name),
		ProvisionedThroughput: throughput,
		KeySchema:             keyschema,
	}

	if attributedata, ok := d.GetOk("attribute"); ok {
		attributes := []*dynamodb.AttributeDefinition{}
		attributeSet := attributedata.(*schema.Set)
		for _, attribute := range attributeSet.List() {
			attr := attribute.(map[string]interface{})
			attributes = append(attributes, &dynamodb.AttributeDefinition{
				AttributeName: aws.String(attr["name"].(string)),
				AttributeType: aws.String(attr["type"].(string)),
			})
		}

		req.AttributeDefinitions = attributes
	}

	if lsidata, ok := d.GetOk("local_secondary_index"); ok {
		log.Printf("[DEBUG] Adding LSI data to the table")

		lsiSet := lsidata.(*schema.Set)
		localSecondaryIndexes := []*dynamodb.LocalSecondaryIndex{}
		for _, lsiObject := range lsiSet.List() {
			lsi := lsiObject.(map[string]interface{})

			projection := &dynamodb.Projection{
				ProjectionType: aws.String(lsi["projection_type"].(string)),
			}

			if lsi["projection_type"] == "INCLUDE" {
				non_key_attributes := []*string{}
				for _, attr := range lsi["non_key_attributes"].([]interface{}) {
					non_key_attributes = append(non_key_attributes, aws.String(attr.(string)))
				}
				projection.NonKeyAttributes = non_key_attributes
			}

			localSecondaryIndexes = append(localSecondaryIndexes, &dynamodb.LocalSecondaryIndex{
				IndexName: aws.String(lsi["name"].(string)),
				KeySchema: []*dynamodb.KeySchemaElement{
					{
						AttributeName: aws.String(hash_key_name),
						KeyType:       aws.String("HASH"),
					},
					{
						AttributeName: aws.String(lsi["range_key"].(string)),
						KeyType:       aws.String("RANGE"),
					},
				},
				Projection: projection,
			})
		}

		req.LocalSecondaryIndexes = localSecondaryIndexes

		log.Printf("[DEBUG] Added %d LSI definitions", len(localSecondaryIndexes))
	}

	if gsidata, ok := d.GetOk("global_secondary_index"); ok {
		globalSecondaryIndexes := []*dynamodb.GlobalSecondaryIndex{}

		gsiSet := gsidata.(*schema.Set)
		for _, gsiObject := range gsiSet.List() {
			gsi := gsiObject.(map[string]interface{})
			gsiObject := createGSIFromData(&gsi)
			globalSecondaryIndexes = append(globalSecondaryIndexes, &gsiObject)
		}
		req.GlobalSecondaryIndexes = globalSecondaryIndexes
	}

	if _, ok := d.GetOk("stream_enabled"); ok {

		req.StreamSpecification = &dynamodb.StreamSpecification{
			StreamEnabled:  aws.Bool(d.Get("stream_enabled").(bool)),
			StreamViewType: aws.String(d.Get("stream_view_type").(string)),
		}

		log.Printf("[DEBUG] Adding StreamSpecifications to the table")
	}

	_, timeToLiveOk := d.GetOk("ttl")
	_, tagsOk := d.GetOk("tags")

	attemptCount := 1
	for attemptCount <= DYNAMODB_MAX_THROTTLE_RETRIES {
		output, err := dynamodbconn.CreateTable(req)
		if err != nil {
			if awsErr, ok := err.(awserr.Error); ok {
				switch code := awsErr.Code(); code {
				case "ThrottlingException":
					log.Printf("[DEBUG] Attempt %d/%d: Sleeping for a bit to throttle back create request", attemptCount, DYNAMODB_MAX_THROTTLE_RETRIES)
					time.Sleep(DYNAMODB_THROTTLE_SLEEP)
					attemptCount += 1
				case "LimitExceededException":
					// If we're at resource capacity, error out without retry. e.g.
					// Subscriber limit exceeded: There is a limit of 256 tables per subscriber
					// Do not error out on this similar throttling message:
					// Subscriber limit exceeded: Only 10 tables can be created, updated, or deleted simultaneously
					if strings.Contains(awsErr.Message(), "Subscriber limit exceeded:") && !strings.Contains(awsErr.Message(), "can be created, updated, or deleted simultaneously") {
						return fmt.Errorf("AWS Error creating DynamoDB table: %s", err)
					}
					log.Printf("[DEBUG] Limit on concurrent table creations hit, sleeping for a bit")
					time.Sleep(DYNAMODB_LIMIT_EXCEEDED_SLEEP)
					attemptCount += 1
				default:
					// Some other non-retryable exception occurred
					return fmt.Errorf("AWS Error creating DynamoDB table: %s", err)
				}
			} else {
				// Non-AWS exception occurred, give up
				return fmt.Errorf("Error creating DynamoDB table: %s", err)
			}
		} else {
			// No error, set ID and return
			d.SetId(*output.TableDescription.TableName)
			tableArn := *output.TableDescription.TableArn
			if err := d.Set("arn", tableArn); err != nil {
				return err
			}

			// Wait, till table is active before imitating any TimeToLive changes
			if err := waitForTableToBeActive(d.Id(), meta); err != nil {
				log.Printf("[DEBUG] Error waiting for table to be active: %s", err)
				return err
			}

			log.Printf("[DEBUG] Setting DynamoDB TimeToLive on arn: %s", tableArn)
			if timeToLiveOk {
				if err := updateTimeToLive(d, meta); err != nil {
					log.Printf("[DEBUG] Error updating table TimeToLive: %s", err)
					return err
				}
			}

			if tagsOk {
				log.Printf("[DEBUG] Setting DynamoDB Tags on arn: %s", tableArn)
				if err := createTableTags(d, meta); err != nil {
					return err
				}
			}

			return resourceAwsDynamoDbTableRead(d, meta)
		}
	}

	// Too many throttling events occurred, give up
	return fmt.Errorf("Unable to create DynamoDB table '%s' after %d attempts", name, attemptCount)
}

func resourceAwsDynamoDbTableUpdate(d *schema.ResourceData, meta interface{}) error {

	log.Printf("[DEBUG] Updating DynamoDB table %s", d.Id())
	dynamodbconn := meta.(*AWSClient).dynamodbconn

	// Ensure table is active before trying to update
	if err := waitForTableToBeActive(d.Id(), meta); err != nil {
		return errwrap.Wrapf("Error waiting for Dynamo DB Table update: {{err}}", err)
	}

	if d.HasChange("read_capacity") || d.HasChange("write_capacity") {
		req := &dynamodb.UpdateTableInput{
			TableName: aws.String(d.Id()),
		}

		throughput := &dynamodb.ProvisionedThroughput{
			ReadCapacityUnits:  aws.Int64(int64(d.Get("read_capacity").(int))),
			WriteCapacityUnits: aws.Int64(int64(d.Get("write_capacity").(int))),
		}
		req.ProvisionedThroughput = throughput

		_, err := dynamodbconn.UpdateTable(req)

		if err != nil {
			return err
		}

		if err := waitForTableToBeActive(d.Id(), meta); err != nil {
			return errwrap.Wrapf("Error waiting for Dynamo DB Table update: {{err}}", err)
		}
	}

	if d.HasChange("stream_enabled") || d.HasChange("stream_view_type") {
		req := &dynamodb.UpdateTableInput{
			TableName: aws.String(d.Id()),
		}

		req.StreamSpecification = &dynamodb.StreamSpecification{
			StreamEnabled:  aws.Bool(d.Get("stream_enabled").(bool)),
			StreamViewType: aws.String(d.Get("stream_view_type").(string)),
		}

		_, err := dynamodbconn.UpdateTable(req)

		if err != nil {
			return err
		}

		if err := waitForTableToBeActive(d.Id(), meta); err != nil {
			return errwrap.Wrapf("Error waiting for Dynamo DB Table update: {{err}}", err)
		}
	}

	if d.HasChange("global_secondary_index") {
		log.Printf("[DEBUG] Changed GSI data")
		req := &dynamodb.UpdateTableInput{
			TableName: aws.String(d.Id()),
		}

		o, n := d.GetChange("global_secondary_index")

		oldSet := o.(*schema.Set)
		newSet := n.(*schema.Set)

		// Track old names so we can know which ones we need to just update based on
		// capacity changes, terraform appears to only diff on the set hash, not the
		// contents so we need to make sure we don't delete any indexes that we
		// just want to update the capacity for
		oldGsiNameSet := make(map[string]bool)
		newGsiNameSet := make(map[string]bool)

		for _, gsidata := range oldSet.List() {
			gsiName := gsidata.(map[string]interface{})["name"].(string)
			oldGsiNameSet[gsiName] = true
		}

		for _, gsidata := range newSet.List() {
			gsiName := gsidata.(map[string]interface{})["name"].(string)
			newGsiNameSet[gsiName] = true
		}

		// First determine what's new
		for _, newgsidata := range newSet.List() {
			updates := []*dynamodb.GlobalSecondaryIndexUpdate{}
			newGsiName := newgsidata.(map[string]interface{})["name"].(string)
			if _, exists := oldGsiNameSet[newGsiName]; !exists {
				attributes := []*dynamodb.AttributeDefinition{}
				gsidata := newgsidata.(map[string]interface{})
				gsi := createGSIFromData(&gsidata)
				log.Printf("[DEBUG] Adding GSI %s", *gsi.IndexName)
				update := &dynamodb.GlobalSecondaryIndexUpdate{
					Create: &dynamodb.CreateGlobalSecondaryIndexAction{
						IndexName:             gsi.IndexName,
						KeySchema:             gsi.KeySchema,
						ProvisionedThroughput: gsi.ProvisionedThroughput,
						Projection:            gsi.Projection,
					},
				}
				updates = append(updates, update)

				// Hash key is required, range key isn't
				hashkey_type, err := getAttributeType(d, *gsi.KeySchema[0].AttributeName)
				if err != nil {
					return err
				}

				attributes = append(attributes, &dynamodb.AttributeDefinition{
					AttributeName: gsi.KeySchema[0].AttributeName,
					AttributeType: aws.String(hashkey_type),
				})

				// If there's a range key, there will be 2 elements in KeySchema
				if len(gsi.KeySchema) == 2 {
					rangekey_type, err := getAttributeType(d, *gsi.KeySchema[1].AttributeName)
					if err != nil {
						return err
					}

					attributes = append(attributes, &dynamodb.AttributeDefinition{
						AttributeName: gsi.KeySchema[1].AttributeName,
						AttributeType: aws.String(rangekey_type),
					})
				}

				req.AttributeDefinitions = attributes
				req.GlobalSecondaryIndexUpdates = updates
				_, err = dynamodbconn.UpdateTable(req)

				if err != nil {
					return err
				}

				if err := waitForTableToBeActive(d.Id(), meta); err != nil {
					return errwrap.Wrapf("Error waiting for Dynamo DB Table update: {{err}}", err)
				}

				if err := waitForGSIToBeActive(d.Id(), *gsi.IndexName, meta); err != nil {
					return errwrap.Wrapf("Error waiting for Dynamo DB GSIT to be active: {{err}}", err)
				}

			}
		}

		for _, oldgsidata := range oldSet.List() {
			updates := []*dynamodb.GlobalSecondaryIndexUpdate{}
			oldGsiName := oldgsidata.(map[string]interface{})["name"].(string)
			if _, exists := newGsiNameSet[oldGsiName]; !exists {
				gsidata := oldgsidata.(map[string]interface{})
				log.Printf("[DEBUG] Deleting GSI %s", gsidata["name"].(string))
				update := &dynamodb.GlobalSecondaryIndexUpdate{
					Delete: &dynamodb.DeleteGlobalSecondaryIndexAction{
						IndexName: aws.String(gsidata["name"].(string)),
					},
				}
				updates = append(updates, update)

				req.GlobalSecondaryIndexUpdates = updates
				_, err := dynamodbconn.UpdateTable(req)

				if err != nil {
					return err
				}

				if err := waitForTableToBeActive(d.Id(), meta); err != nil {
					return errwrap.Wrapf("Error waiting for Dynamo DB Table update: {{err}}", err)
				}
			}
		}
	}

	// Update any out-of-date read / write capacity
	if gsiObjects, ok := d.GetOk("global_secondary_index"); ok {
		gsiSet := gsiObjects.(*schema.Set)
		if len(gsiSet.List()) > 0 {
			log.Printf("Updating capacity as needed!")

			// We can only change throughput, but we need to make sure it's actually changed
			tableDescription, err := dynamodbconn.DescribeTable(&dynamodb.DescribeTableInput{
				TableName: aws.String(d.Id()),
			})

			if err != nil {
				return err
			}

			table := tableDescription.Table

			for _, updatedgsidata := range gsiSet.List() {
				updates := []*dynamodb.GlobalSecondaryIndexUpdate{}
				gsidata := updatedgsidata.(map[string]interface{})
				gsiName := gsidata["name"].(string)
				gsiWriteCapacity := gsidata["write_capacity"].(int)
				gsiReadCapacity := gsidata["read_capacity"].(int)

				log.Printf("[DEBUG] Updating GSI %s", gsiName)
				gsi, err := getGlobalSecondaryIndex(gsiName, table.GlobalSecondaryIndexes)

				if err != nil {
					return err
				}

				capacityUpdated := false

				if int64(gsiReadCapacity) != *gsi.ProvisionedThroughput.ReadCapacityUnits ||
					int64(gsiWriteCapacity) != *gsi.ProvisionedThroughput.WriteCapacityUnits {
					capacityUpdated = true
				}

				if capacityUpdated {
					update := &dynamodb.GlobalSecondaryIndexUpdate{
						Update: &dynamodb.UpdateGlobalSecondaryIndexAction{
							IndexName: aws.String(gsidata["name"].(string)),
							ProvisionedThroughput: &dynamodb.ProvisionedThroughput{
								WriteCapacityUnits: aws.Int64(int64(gsiWriteCapacity)),
								ReadCapacityUnits:  aws.Int64(int64(gsiReadCapacity)),
							},
						},
					}
					updates = append(updates, update)

				}

				if len(updates) > 0 {

					req := &dynamodb.UpdateTableInput{
						TableName: aws.String(d.Id()),
					}

					req.GlobalSecondaryIndexUpdates = updates

					log.Printf("[DEBUG] Updating GSI read / write capacity on %s", d.Id())
					_, err := dynamodbconn.UpdateTable(req)

					if err != nil {
						log.Printf("[DEBUG] Error updating table: %s", err)
						return err
					}

					if err := waitForGSIToBeActive(d.Id(), gsiName, meta); err != nil {
						return errwrap.Wrapf("Error waiting for Dynamo DB GSI to be active: {{err}}", err)
					}
				}
			}
		}

	}

	if d.HasChange("ttl") {
		if err := updateTimeToLive(d, meta); err != nil {
			log.Printf("[DEBUG] Error updating table TimeToLive: %s", err)
			return err
		}
	}

	// Update tags
	if err := setTagsDynamoDb(dynamodbconn, d); err != nil {
		return err
	}

	return resourceAwsDynamoDbTableRead(d, meta)
}

func updateTimeToLive(d *schema.ResourceData, meta interface{}) error {
	dynamodbconn := meta.(*AWSClient).dynamodbconn

	if ttl, ok := d.GetOk("ttl"); ok {

		timeToLiveSet := ttl.(*schema.Set)

		spec := &dynamodb.TimeToLiveSpecification{}

		timeToLive := timeToLiveSet.List()[0].(map[string]interface{})
		spec.AttributeName = aws.String(timeToLive["attribute_name"].(string))
		spec.Enabled = aws.Bool(timeToLive["enabled"].(bool))

		req := &dynamodb.UpdateTimeToLiveInput{
			TableName:               aws.String(d.Id()),
			TimeToLiveSpecification: spec,
		}

		_, err := dynamodbconn.UpdateTimeToLive(req)

		if err != nil {
			// If ttl was not set within the .tf file before and has now been added we still run this command to update
			// But there has been no change so lets continue
			if awsErr, ok := err.(awserr.Error); ok && awsErr.Code() == "ValidationException" && awsErr.Message() == "TimeToLive is already disabled" {
				return nil
			}
			log.Printf("[DEBUG] Error updating TimeToLive on table: %s", err)
			return err
		}

		log.Printf("[DEBUG] Updated TimeToLive on table")

		if err := waitForTimeToLiveUpdateToBeCompleted(d.Id(), timeToLive["enabled"].(bool), meta); err != nil {
			return errwrap.Wrapf("Error waiting for Dynamo DB TimeToLive to be updated: {{err}}", err)
		}
	}

	return nil
}

func resourceAwsDynamoDbTableRead(d *schema.ResourceData, meta interface{}) error {
	dynamodbconn := meta.(*AWSClient).dynamodbconn
	log.Printf("[DEBUG] Loading data for DynamoDB table '%s'", d.Id())
	req := &dynamodb.DescribeTableInput{
		TableName: aws.String(d.Id()),
	}

	result, err := dynamodbconn.DescribeTable(req)

	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok && awsErr.Code() == "ResourceNotFoundException" {
			log.Printf("[WARN] Dynamodb Table (%s) not found, error code (404)", d.Id())
			d.SetId("")
			return nil
		}
		return err
	}

	return flattenAwsDynamoDbTableResource(d, meta, result.Table)
}

func flattenAwsDynamoDbTableResource(d *schema.ResourceData, meta interface{}, table *dynamodb.TableDescription) error {
	dynamodbconn := meta.(*AWSClient).dynamodbconn

	d.Set("write_capacity", table.ProvisionedThroughput.WriteCapacityUnits)
	d.Set("read_capacity", table.ProvisionedThroughput.ReadCapacityUnits)

	attributes := []interface{}{}
	for _, attrdef := range table.AttributeDefinitions {
		attribute := map[string]string{
			"name": *attrdef.AttributeName,
			"type": *attrdef.AttributeType,
		}
		attributes = append(attributes, attribute)
		log.Printf("[DEBUG] Added Attribute: %s", attribute["name"])
	}

	d.Set("attribute", attributes)
	d.Set("name", table.TableName)

	for _, attribute := range table.KeySchema {
		if *attribute.KeyType == "HASH" {
			d.Set("hash_key", attribute.AttributeName)
		}

		if *attribute.KeyType == "RANGE" {
			d.Set("range_key", attribute.AttributeName)
		}
	}

	lsiList := make([]map[string]interface{}, 0, len(table.LocalSecondaryIndexes))
	for _, lsiObject := range table.LocalSecondaryIndexes {
		lsi := map[string]interface{}{
			"name":            *lsiObject.IndexName,
			"projection_type": *lsiObject.Projection.ProjectionType,
		}

		for _, attribute := range lsiObject.KeySchema {

			if *attribute.KeyType == "RANGE" {
				lsi["range_key"] = *attribute.AttributeName
			}
		}
		nkaList := make([]string, len(lsiObject.Projection.NonKeyAttributes))
		for _, nka := range lsiObject.Projection.NonKeyAttributes {
			nkaList = append(nkaList, *nka)
		}
		lsi["non_key_attributes"] = nkaList

		lsiList = append(lsiList, lsi)
	}

	err := d.Set("local_secondary_index", lsiList)
	if err != nil {
		return err
	}

	gsiList := make([]map[string]interface{}, 0, len(table.GlobalSecondaryIndexes))
	for _, gsiObject := range table.GlobalSecondaryIndexes {
		gsi := map[string]interface{}{
			"write_capacity": *gsiObject.ProvisionedThroughput.WriteCapacityUnits,
			"read_capacity":  *gsiObject.ProvisionedThroughput.ReadCapacityUnits,
			"name":           *gsiObject.IndexName,
		}

		for _, attribute := range gsiObject.KeySchema {
			if *attribute.KeyType == "HASH" {
				gsi["hash_key"] = *attribute.AttributeName
			}

			if *attribute.KeyType == "RANGE" {
				gsi["range_key"] = *attribute.AttributeName
			}
		}

		gsi["projection_type"] = *(gsiObject.Projection.ProjectionType)

		nonKeyAttrs := make([]string, 0, len(gsiObject.Projection.NonKeyAttributes))
		for _, nonKeyAttr := range gsiObject.Projection.NonKeyAttributes {
			nonKeyAttrs = append(nonKeyAttrs, *nonKeyAttr)
		}
		gsi["non_key_attributes"] = nonKeyAttrs

		gsiList = append(gsiList, gsi)
		log.Printf("[DEBUG] Added GSI: %s - Read: %d / Write: %d", gsi["name"], gsi["read_capacity"], gsi["write_capacity"])
	}

	if table.StreamSpecification != nil {
		d.Set("stream_view_type", table.StreamSpecification.StreamViewType)
		d.Set("stream_enabled", table.StreamSpecification.StreamEnabled)
		d.Set("stream_arn", table.LatestStreamArn)
		d.Set("stream_label", table.LatestStreamLabel)
	}

	err = d.Set("global_secondary_index", gsiList)
	if err != nil {
		return err
	}

	d.Set("arn", table.TableArn)

	timeToLiveReq := &dynamodb.DescribeTimeToLiveInput{
		TableName: aws.String(d.Id()),
	}
	timeToLiveOutput, err := dynamodbconn.DescribeTimeToLive(timeToLiveReq)
	if err != nil {
		return err
	}

	if timeToLiveOutput.TimeToLiveDescription != nil && timeToLiveOutput.TimeToLiveDescription.AttributeName != nil {
		timeToLiveList := []interface{}{
			map[string]interface{}{
				"attribute_name": *timeToLiveOutput.TimeToLiveDescription.AttributeName,
				"enabled":        (*timeToLiveOutput.TimeToLiveDescription.TimeToLiveStatus == dynamodb.TimeToLiveStatusEnabled),
			},
		}
		err := d.Set("ttl", timeToLiveList)
		if err != nil {
			return err
		}

		log.Printf("[DEBUG] Loaded TimeToLive data for DynamoDB table '%s'", d.Id())
	}

	tags, err := readTableTags(d, meta)
	if err != nil {
		return err
	}
	if len(tags) != 0 {
		d.Set("tags", tags)
	}

	return nil
}

func resourceAwsDynamoDbTableDelete(d *schema.ResourceData, meta interface{}) error {
	dynamodbconn := meta.(*AWSClient).dynamodbconn

	if err := waitForTableToBeActive(d.Id(), meta); err != nil {
		return errwrap.Wrapf("Error waiting for Dynamo DB Table update: {{err}}", err)
	}

	log.Printf("[DEBUG] DynamoDB delete table: %s", d.Id())

	_, err := dynamodbconn.DeleteTable(&dynamodb.DeleteTableInput{
		TableName: aws.String(d.Id()),
	})
	if err != nil {
		return err
	}

	params := &dynamodb.DescribeTableInput{
		TableName: aws.String(d.Id()),
	}

	err = resource.Retry(10*time.Minute, func() *resource.RetryError {
		t, err := dynamodbconn.DescribeTable(params)
		if err != nil {
			if awserr, ok := err.(awserr.Error); ok && awserr.Code() == "ResourceNotFoundException" {
				return nil
			}
			// Didn't recognize the error, so shouldn't retry.
			return resource.NonRetryableError(err)
		}

		if t != nil {
			if t.Table.TableStatus != nil && strings.ToLower(*t.Table.TableStatus) == "deleting" {
				log.Printf("[DEBUG] AWS Dynamo DB table (%s) is still deleting", d.Id())
				return resource.RetryableError(fmt.Errorf("still deleting"))
			}
		}

		// we should be not found or deleting, so error here
		return resource.NonRetryableError(err)
	})

	// check error from retry
	if err != nil {
		return err
	}

	return nil
}

func createGSIFromData(data *map[string]interface{}) dynamodb.GlobalSecondaryIndex {

	projection := &dynamodb.Projection{
		ProjectionType: aws.String((*data)["projection_type"].(string)),
	}

	if (*data)["projection_type"] == "INCLUDE" {
		non_key_attributes := []*string{}
		for _, attr := range (*data)["non_key_attributes"].([]interface{}) {
			non_key_attributes = append(non_key_attributes, aws.String(attr.(string)))
		}
		projection.NonKeyAttributes = non_key_attributes
	}

	writeCapacity := (*data)["write_capacity"].(int)
	readCapacity := (*data)["read_capacity"].(int)

	key_schema := []*dynamodb.KeySchemaElement{
		{
			AttributeName: aws.String((*data)["hash_key"].(string)),
			KeyType:       aws.String("HASH"),
		},
	}

	range_key_name := (*data)["range_key"]
	if range_key_name != "" {
		range_key_element := &dynamodb.KeySchemaElement{
			AttributeName: aws.String(range_key_name.(string)),
			KeyType:       aws.String("RANGE"),
		}

		key_schema = append(key_schema, range_key_element)
	}

	return dynamodb.GlobalSecondaryIndex{
		IndexName:  aws.String((*data)["name"].(string)),
		KeySchema:  key_schema,
		Projection: projection,
		ProvisionedThroughput: &dynamodb.ProvisionedThroughput{
			WriteCapacityUnits: aws.Int64(int64(writeCapacity)),
			ReadCapacityUnits:  aws.Int64(int64(readCapacity)),
		},
	}
}

func getGlobalSecondaryIndex(indexName string, indexList []*dynamodb.GlobalSecondaryIndexDescription) (*dynamodb.GlobalSecondaryIndexDescription, error) {
	for _, gsi := range indexList {
		if *gsi.IndexName == indexName {
			return gsi, nil
		}
	}

	return &dynamodb.GlobalSecondaryIndexDescription{}, fmt.Errorf("Can't find a GSI by that name...")
}

func getAttributeType(d *schema.ResourceData, attributeName string) (string, error) {
	if attributedata, ok := d.GetOk("attribute"); ok {
		attributeSet := attributedata.(*schema.Set)
		for _, attribute := range attributeSet.List() {
			attr := attribute.(map[string]interface{})
			if attr["name"] == attributeName {
				return attr["type"].(string), nil
			}
		}
	}

	return "", fmt.Errorf("Unable to find an attribute named %s", attributeName)
}

func waitForGSIToBeActive(tableName string, gsiName string, meta interface{}) error {
	dynamodbconn := meta.(*AWSClient).dynamodbconn
	req := &dynamodb.DescribeTableInput{
		TableName: aws.String(tableName),
	}

	activeIndex := false

	for activeIndex == false {

		result, err := dynamodbconn.DescribeTable(req)

		if err != nil {
			return err
		}

		table := result.Table
		var targetGSI *dynamodb.GlobalSecondaryIndexDescription = nil

		for _, gsi := range table.GlobalSecondaryIndexes {
			if *gsi.IndexName == gsiName {
				targetGSI = gsi
			}
		}

		if targetGSI != nil {
			activeIndex = *targetGSI.IndexStatus == "ACTIVE"

			if !activeIndex {
				log.Printf("[DEBUG] Sleeping for 5 seconds for %s GSI to become active", gsiName)
				time.Sleep(5 * time.Second)
			}
		} else {
			log.Printf("[DEBUG] GSI %s did not exist, giving up", gsiName)
			break
		}
	}

	return nil

}

func waitForTableToBeActive(tableName string, meta interface{}) error {
	dynamodbconn := meta.(*AWSClient).dynamodbconn
	req := &dynamodb.DescribeTableInput{
		TableName: aws.String(tableName),
	}

	activeState := false

	for activeState == false {
		result, err := dynamodbconn.DescribeTable(req)

		if err != nil {
			return err
		}

		activeState = *result.Table.TableStatus == "ACTIVE"

		// Wait for a few seconds
		if !activeState {
			log.Printf("[DEBUG] Sleeping for 5 seconds for table to become active")
			time.Sleep(5 * time.Second)
		}
	}

	return nil

}

func waitForTimeToLiveUpdateToBeCompleted(tableName string, enabled bool, meta interface{}) error {
	dynamodbconn := meta.(*AWSClient).dynamodbconn
	req := &dynamodb.DescribeTimeToLiveInput{
		TableName: aws.String(tableName),
	}

	stateMatched := false
	for stateMatched == false {
		result, err := dynamodbconn.DescribeTimeToLive(req)

		if err != nil {
			return err
		}

		if enabled {
			stateMatched = *result.TimeToLiveDescription.TimeToLiveStatus == dynamodb.TimeToLiveStatusEnabled
		} else {
			stateMatched = *result.TimeToLiveDescription.TimeToLiveStatus == dynamodb.TimeToLiveStatusDisabled
		}

		// Wait for a few seconds, this may take a long time...
		if !stateMatched {
			log.Printf("[DEBUG] Sleeping for 5 seconds before checking TimeToLive state again")
			time.Sleep(5 * time.Second)
		}
	}

	log.Printf("[DEBUG] TimeToLive update complete")

	return nil

}

func createTableTags(d *schema.ResourceData, meta interface{}) error {
	// DynamoDB Table has to be in the ACTIVE state in order to tag the resource
	if err := waitForTableToBeActive(d.Id(), meta); err != nil {
		return err
	}
	tags := d.Get("tags").(map[string]interface{})
	arn := d.Get("arn").(string)
	dynamodbconn := meta.(*AWSClient).dynamodbconn
	req := &dynamodb.TagResourceInput{
		ResourceArn: aws.String(arn),
		Tags:        tagsFromMapDynamoDb(tags),
	}
	_, err := dynamodbconn.TagResource(req)
	if err != nil {
		return fmt.Errorf("Error tagging dynamodb resource: %s", err)
	}
	return nil
}

func readTableTags(d *schema.ResourceData, meta interface{}) (map[string]string, error) {
	if err := waitForTableToBeActive(d.Id(), meta); err != nil {
		return nil, err
	}
	arn := d.Get("arn").(string)
	//result := make(map[string]string)

	dynamodbconn := meta.(*AWSClient).dynamodbconn
	req := &dynamodb.ListTagsOfResourceInput{
		ResourceArn: aws.String(arn),
	}

	output, err := dynamodbconn.ListTagsOfResource(req)
	if err != nil {
		return nil, fmt.Errorf("Error reading tags from dynamodb resource: %s", err)
	}
	result := tagsToMapDynamoDb(output.Tags)
	// TODO Read NextToken if avail
	return result, nil
}
