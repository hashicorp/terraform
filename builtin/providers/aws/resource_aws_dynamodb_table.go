package aws

import (
	"bytes"
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/terraform/helper/schema"

	"github.com/aws/aws-sdk-go/aws"
    "github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/hashicorp/terraform/helper/hashcode"

)

// Number of times to retry if a throttling-related exception occurs
const DYNAMODB_MAX_THROTTLE_RETRIES = 10

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

		Schema: map[string]*schema.Schema{
			"arn": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"latest_stream_arn": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"hash_key": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"range_key": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"write_capacity": &schema.Schema{
				Type:     schema.TypeInt,
				Required: true,
			},
			"read_capacity": &schema.Schema{
				Type:     schema.TypeInt,
				Required: true,
			},
			"only_scale_up": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
			},
			"attribute": &schema.Schema{
				Type:     schema.TypeSet,
				Required: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
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
				Set: func(v interface{}) int {
					var buf bytes.Buffer
					m := v.(map[string]interface{})
					buf.WriteString(fmt.Sprintf("%s-", m["name"].(string)))
					return hashcode.String(buf.String())
				},
			},

			"stream_specification": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"enabled": &schema.Schema{
							Type:     schema.TypeBool,
							Required: true,
						},
						"view_type": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
				Set: resourceAwsDynamoDBTableStreamSpecificationHash,
			},

			"local_secondary_index": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"range_key": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"projection_type": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"non_key_attributes": &schema.Schema{
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
			"global_secondary_index": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"write_capacity": &schema.Schema{
							Type:     schema.TypeInt,
							Required: true,
						},
						"read_capacity": &schema.Schema{
							Type:     schema.TypeInt,
							Required: true,
						},
						"hash_key": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"range_key": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"projection_type": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"non_key_attributes": &schema.Schema{
							Type:     schema.TypeList,
							Optional: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},
					},
				},
				// GSI names are the uniqueness constraint
				Set: func(v interface{}) int {
					var buf bytes.Buffer
					m := v.(map[string]interface{})
					buf.WriteString(fmt.Sprintf("%s-", m["name"].(string)))
					buf.WriteString(fmt.Sprintf("%d-", m["write_capacity"].(int)))
					buf.WriteString(fmt.Sprintf("%d-", m["read_capacity"].(int)))
					return hashcode.String(buf.String())
				},
			},
		},
	}
}

func resourceAwsDynamoDbTableCreate(d *schema.ResourceData, meta interface{}) error {
	dynamodbconn := meta.(*AWSClient).dynamodbconn

	name := d.Get("name").(string)

	log.Printf("[DEBUG] DynamoDB table create: %s", name)

	only_scale_up := d.Get("only_scale_up").(bool)
	log.Printf("[DEBUG] only_scale_up flag create %v", only_scale_up)

	throughput := &dynamodb.ProvisionedThroughput{
		ReadCapacityUnits:  aws.Int64(int64(d.Get("read_capacity").(int))),
		WriteCapacityUnits: aws.Int64(int64(d.Get("write_capacity").(int))),
	}

	hash_key_name := d.Get("hash_key").(string)
	keyschema := []*dynamodb.KeySchemaElement{
		&dynamodb.KeySchemaElement{
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

	if streamspecificationdata, ok := d.GetOk("stream_specification"); ok {
		log.Printf("[DEBUG] Adding StreamSpecification data to the table")
		streamspecificationSet := streamspecificationdata.(*schema.Set)
		if len(streamspecificationSet.List()) > 0 {
			// Grab the first element
			spec := streamspecificationSet.List()[0].(map[string]interface{})
			streamSpecification := &dynamodb.StreamSpecification{
				StreamEnabled:  aws.Bool(spec["enabled"].(bool)),
				StreamViewType: aws.String(spec["view_type"].(string)),
			}
			req.StreamSpecification = streamSpecification
		}
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
					&dynamodb.KeySchemaElement{
						AttributeName: aws.String(hash_key_name),
						KeyType:       aws.String("HASH"),
					},
					&dynamodb.KeySchemaElement{
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

	attemptCount := 1
	for attemptCount <= DYNAMODB_MAX_THROTTLE_RETRIES {
		attemptCount += 1
		output, err := dynamodbconn.CreateTable(req)
		if err != nil {
			if awsErr, ok := err.(awserr.Error); ok {
				if awsErr.Code() == "ThrottlingException" {
					log.Printf("[DEBUG] Attempt %d/%d: Sleeping for a bit to throttle back create request", attemptCount, DYNAMODB_MAX_THROTTLE_RETRIES)
					time.Sleep(DYNAMODB_THROTTLE_SLEEP)
				} else if awsErr.Code() == "LimitExceededException" {
					log.Printf("[DEBUG] Limit on concurrent table creations hit, sleeping for a bit")
					time.Sleep(DYNAMODB_LIMIT_EXCEEDED_SLEEP)
				} else {
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
			if err := d.Set("arn", *output.TableDescription.TableArn); err != nil {
				return err
			}
			if output.TableDescription.StreamSpecification != nil {
				if *output.TableDescription.StreamSpecification.StreamEnabled {
					if err := d.Set("latest_stream_arn", *output.TableDescription.LatestStreamArn); err != nil {
						return err
					}
				}
			}

			return resourceAwsDynamoDbTableRead(d, meta)

		}
	}
	return fmt.Errorf("Exiting after maximum number of retry counts for DynamoDB table: %s", name)
}

func getConditionallyScalingCapacity(d *schema.ResourceData, key string) (int, int) {
	only_scale_up := d.Get("only_scale_up").(bool)
	old_capacity_param, new_capacity_param := d.GetChange(key)

	old_capacity := old_capacity_param.(int)
	var new_capacity = new_capacity_param.(int)

	if (old_capacity > new_capacity) && only_scale_up {
		new_capacity = old_capacity
		d.Set(key, new_capacity)
	}

	return old_capacity, new_capacity
}

func resourceAwsDynamoDbTableUpdate(d *schema.ResourceData, meta interface{}) error {

	log.Printf("[DEBUG] Updating DynamoDB table %s", d.Id())
	dynamodbconn := meta.(*AWSClient).dynamodbconn

	// Ensure table is active before trying to update
	waitForTableToBeActive(d.Id(), meta)

	// LSI can only be done at create-time, abort if it's been changed
	if d.HasChange("local_secondary_index") {
		return fmt.Errorf("Local secondary indexes can only be built at creation, you cannot update them!")
	}

	if d.HasChange("hash_key") {
		return fmt.Errorf("Hash key can only be specified at creation, you cannot modify it.")
	}

	if d.HasChange("range_key") {
		return fmt.Errorf("Range key can only be specified at creation, you cannot modify it.")
	}

	if d.HasChange("read_capacity") || d.HasChange("write_capacity") {
		old_read_capacity, new_read_capacity := getConditionallyScalingCapacity(d, "read_capacity")
		old_write_capacity, new_write_capacity := getConditionallyScalingCapacity(d, "write_capacity")

		// should we actually change the capacity of the table?
		if new_read_capacity != old_read_capacity || new_write_capacity != old_write_capacity {
			req := &dynamodb.UpdateTableInput{
				TableName: aws.String(d.Id()),
			}

			throughput := &dynamodb.ProvisionedThroughput{
				ReadCapacityUnits:  aws.Int64(int64(new_read_capacity)),
				WriteCapacityUnits: aws.Int64(int64(new_write_capacity)),
			}
			req.ProvisionedThroughput = throughput

			// Updates for capacity needs to be done before updating
			// the streamspecification - it cannot be done in the same call
			_, err := dynamodbconn.UpdateTable(req)
			if err != nil {
				return err
			}

			waitForTableToBeActive(d.Id(), meta)
		}
	}

	if d.HasChange("stream_specification") {
		log.Printf("[DEBUG] Changed Stream Specification")
		req := &dynamodb.UpdateTableInput{
			TableName: aws.String(d.Id()),
		}

		vs := d.Get("stream_specification").(*schema.Set).List()
		if len(vs) > 0 {
			spec := vs[0].(map[string]interface{})
			streamSpecification := &dynamodb.StreamSpecification{
				StreamEnabled:  aws.Bool(bool(spec["enabled"].(bool))),
				StreamViewType: aws.String(string(spec["view_type"].(string))),
			}
			req.StreamSpecification = streamSpecification
		}

		_, err := dynamodbconn.UpdateTable(req)
		if err != nil {
			return err
		}

		waitForTableToBeActive(d.Id(), meta)
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

				waitForTableToBeActive(d.Id(), meta)
				waitForGSIToBeActive(d.Id(), *gsi.IndexName, meta)

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

				waitForTableToBeActive(d.Id(), meta)
			}
		}
	}

	// Update any out-of-date read / write capacity
	if gsiObjects, ok := d.GetOk("global_secondary_index"); ok {
		gsiSet := gsiObjects.(*schema.Set)
		if len(gsiSet.List()) > 0 {
			log.Printf("[DEBUG] Updating capacity as needed!")


			for _, updatedgsidata := range gsiSet.List() {
				gsidata := updatedgsidata.(map[string]interface{})
				gsiName := gsidata["name"].(string)
				gsiWriteCapacity := int64(gsidata["write_capacity"].(int))
				gsiReadCapacity := int64(gsidata["read_capacity"].(int))
				only_scale_up := d.Get("only_scale_up").(bool)

				attemptCount := 1
				for attemptCount <= DYNAMODB_MAX_THROTTLE_RETRIES {
					attemptCount += 1

					// We can only change throughput, but we need to make sure it actually needs changing
					tableDescription, err := dynamodbconn.DescribeTable(&dynamodb.DescribeTableInput{
						TableName: aws.String(d.Id()),
					})

					if err != nil {
						return err
					}

					table := tableDescription.Table


					updates := []*dynamodb.GlobalSecondaryIndexUpdate{}


					log.Printf("[DEBUG] Updating GSI %s", gsiName)
					gsi, err := getGlobalSecondaryIndex(gsiName, table.GlobalSecondaryIndexes)

					if err != nil {
						return err
					}

					provisionedRead := *gsi.ProvisionedThroughput.ReadCapacityUnits
					provisionedWrite := *gsi.ProvisionedThroughput.WriteCapacityUnits
					if only_scale_up {
						if gsiReadCapacity < provisionedRead {
							gsiReadCapacity = provisionedRead
						}
						if gsiWriteCapacity < provisionedWrite {
							gsiWriteCapacity = provisionedWrite
						}
					}

					log.Printf("[DEBUG] values: pr/pw %v/%v, nr/nw %v/%v",provisionedRead,provisionedWrite, gsiReadCapacity, gsiWriteCapacity)
					// Should we update
					if gsiReadCapacity != provisionedRead  || gsiWriteCapacity != provisionedWrite {
						update := &dynamodb.GlobalSecondaryIndexUpdate{
							Update: &dynamodb.UpdateGlobalSecondaryIndexAction{
								IndexName: aws.String(gsidata["name"].(string)),
								ProvisionedThroughput: &dynamodb.ProvisionedThroughput{
									WriteCapacityUnits: aws.Int64(gsiWriteCapacity),
									ReadCapacityUnits:  aws.Int64(gsiReadCapacity),
								},
							},
						}
						updates = append(updates, update)

					} else {
						log.Printf("[DEBUG] GSI %s already at desired capacity. Skipping", gsiName)
						break
					};

					if len(updates) > 0 {

						req := &dynamodb.UpdateTableInput{
							TableName: aws.String(d.Id()),
						}

						req.GlobalSecondaryIndexUpdates = updates

						// Retry logic
						log.Printf("[DEBUG] Updating GSI read / write capacity on %s.%s to %v/%v", d.Id(), gsiName, gsiReadCapacity,gsiWriteCapacity)
						_, err := dynamodbconn.UpdateTable(req)
						if err != nil {
							if awsErr, ok := err.(awserr.Error); ok {
								if awsErr.Code() == "ResourceInUseException" {
									log.Printf("[DEBUG] Resource in use,  sleeping for a bit table=%s, index=%s", d.Id(), gsiName)
									time.Sleep(DYNAMODB_LIMIT_EXCEEDED_SLEEP)
								} else /*if !(awsErr.Code() == "ValidationException" &&
								            strings.HasPrefix(awsErr.Message(), "The provisioned throughput for the index"))*/ {
									log.Printf("[ERROR] Unhandled error: %s, %s", awsErr.Code(), awsErr.Message())
									return err
								}
							}
						}
					}

				}

				if attemptCount > DYNAMODB_MAX_THROTTLE_RETRIES {
					log.Printf("[DEBUG] Error updating gsi: %s on table %s", gsiName, d.Id())
					return fmt.Errorf("Error updating gsi %s.%s after many retries",d.Id(), gsiName)
				}
			}
		}

	}

	return resourceAwsDynamoDbTableRead(d, meta)
}

func resourceAwsDynamoDbTableRead(d *schema.ResourceData, meta interface{}) error {
	dynamodbconn := meta.(*AWSClient).dynamodbconn

	only_scale_up := d.Get("only_scale_up").(bool)
	log.Printf("[DEBUG] only_scale_up flag read %v", only_scale_up)

	log.Printf("[DEBUG] Loading data for DynamoDB table '%s'", d.Id())
	req := &dynamodb.DescribeTableInput{
		TableName: aws.String(d.Id()),
	}

	result, err := dynamodbconn.DescribeTable(req)

	if err != nil {
		return err
	}

	table := result.Table

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

	err = d.Set("global_secondary_index", gsiList)
	if err != nil {
		return err
	}

	d.Set("arn", table.TableArn)

	// There's only one stream specification, so save that to state as we
	// currently can
	if table.StreamSpecification != nil {
		d.Set("stream_specification", flattenDynamoDBStreamSpecification(table.StreamSpecification))

		if *(table.StreamSpecification.StreamEnabled) {
			d.Set("latest_stream_arn", table.LatestStreamArn)
		}
	} else {
		log.Printf("[DEBUG] StreamSpecification does not exist - will populate as false")
		disabledStreamSpecification := &dynamodb.StreamSpecification{
			StreamEnabled:  aws.Bool(false),
			StreamViewType: aws.String(""),
		}
		d.Set("stream_specification", flattenDynamoDBStreamSpecification(disabledStreamSpecification))
	}
	return nil
}

func resourceAwsDynamoDbTableDelete(d *schema.ResourceData, meta interface{}) error {
	dynamodbconn := meta.(*AWSClient).dynamodbconn

	waitForTableToBeActive(d.Id(), meta)

	log.Printf("[DEBUG] DynamoDB delete table: %s", d.Id())

	only_scale_up := d.Get("only_scale_up").(bool)
	log.Printf("[DEBUG] only_scale_up flag delete %v", only_scale_up)

	_, err := dynamodbconn.DeleteTable(&dynamodb.DeleteTableInput{
		TableName: aws.String(d.Id()),
	})
	if err != nil {
		return fmt.Errorf("AWS Error deleting DynamoDB table '%s': error %s", d.Id(), err)
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
		&dynamodb.KeySchemaElement{
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

func resourceAwsDynamoDBTableStreamSpecificationHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})
	buf.WriteString(fmt.Sprintf("%s-", map[bool]string{true: "true", false: "false"}[m["enabled"].(bool)]))
	buf.WriteString(fmt.Sprintf("%s-", m["view_type"].(string)))
	return hashcode.String(buf.String())
}
