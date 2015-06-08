package aws

import (
	"bytes"
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/terraform/helper/schema"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/hashicorp/terraform/helper/hashcode"
)

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
				// GSI names are the uniqueness constraint
				Set: func(v interface{}) int {
					var buf bytes.Buffer
					m := v.(map[string]interface{})
					buf.WriteString(fmt.Sprintf("%s-", m["name"].(string)))
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

	throughput := &dynamodb.ProvisionedThroughput{
		ReadCapacityUnits:  aws.Long(int64(d.Get("read_capacity").(int))),
		WriteCapacityUnits: aws.Long(int64(d.Get("write_capacity").(int))),
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

	if lsidata, ok := d.GetOk("local_secondary_index"); ok {
		fmt.Printf("[DEBUG] Adding LSI data to the table")

		lsiSet := lsidata.(*schema.Set)
		localSecondaryIndexes := []*dynamodb.LocalSecondaryIndex{}
		for _, lsiObject := range lsiSet.List() {
			lsi := lsiObject.(map[string]interface{})

			projection := &dynamodb.Projection{
				ProjectionType: aws.String(lsi["projection_type"].(string)),
			}

			if lsi["projection_type"] != "ALL" {
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

		fmt.Printf("[DEBUG] Added %d LSI definitions", len(localSecondaryIndexes))
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

	output, err := dynamodbconn.CreateTable(req)
	if err != nil {
		return fmt.Errorf("Error creating DynamoDB table: %s", err)
	}

	d.SetId(*output.TableDescription.TableName)

	// Creation complete, nothing to re-read
	return nil
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
		req := &dynamodb.UpdateTableInput{
			TableName: aws.String(d.Id()),
		}

		throughput := &dynamodb.ProvisionedThroughput{
			ReadCapacityUnits:  aws.Long(int64(d.Get("read_capacity").(int))),
			WriteCapacityUnits: aws.Long(int64(d.Get("write_capacity").(int))),
		}
		req.ProvisionedThroughput = throughput

		_, err := dynamodbconn.UpdateTable(req)

		if err != nil {
			return err
		}

		waitForTableToBeActive(d.Id(), meta)
	}

	if d.HasChange("global_secondary_index") {
		req := &dynamodb.UpdateTableInput{
			TableName: aws.String(d.Id()),
		}

		o, n := d.GetChange("global_secondary_index")

		oldSet := o.(*schema.Set)
		newSet := n.(*schema.Set)
		changedSet := newSet.Intersection(oldSet)

		// First determine what's new
		for _, newgsidata := range newSet.List() {
			updates := []*dynamodb.GlobalSecondaryIndexUpdate{}
			if !oldSet.Contains(newgsidata) {
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
				hashkey_type, err := getAttributeType(d, *(gsi.KeySchema[0].AttributeName))
				if err != nil {
					return err
				}

				rangekey_type, err := getAttributeType(d, *(gsi.KeySchema[1].AttributeName))
				if err != nil {
					return err
				}

				attributes = append(attributes, &dynamodb.AttributeDefinition{
					AttributeName: gsi.KeySchema[0].AttributeName,
					AttributeType: aws.String(hashkey_type),
				})
				attributes = append(attributes, &dynamodb.AttributeDefinition{
					AttributeName: gsi.KeySchema[1].AttributeName,
					AttributeType: aws.String(rangekey_type),
				})

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
			if !newSet.Contains(oldgsidata) {
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

		for _, updatedgsidata := range changedSet.List() {
			updates := []*dynamodb.GlobalSecondaryIndexUpdate{}
			gsidata := updatedgsidata.(map[string]interface{})
			log.Printf("[DEBUG] Updating GSI %s", gsidata["name"].(string))
			update := &dynamodb.GlobalSecondaryIndexUpdate{
				Update: &dynamodb.UpdateGlobalSecondaryIndexAction{
					IndexName: aws.String(gsidata["name"].(string)),
					ProvisionedThroughput: &dynamodb.ProvisionedThroughput{
						WriteCapacityUnits: aws.Long(int64(gsidata["write_capacity"].(int))),
						ReadCapacityUnits:  aws.Long(int64(gsidata["read_capacity"].(int))),
					},
				},
			}
			updates = append(updates, update)

			req.GlobalSecondaryIndexUpdates = updates

			_, err := dynamodbconn.UpdateTable(req)

			if err != nil {
				log.Printf("[DEBUG] Error updating table: %s", err)
				return err
			}
		}
	}

	return resourceAwsDynamoDbTableRead(d, meta)
}

func resourceAwsDynamoDbTableRead(d *schema.ResourceData, meta interface{}) error {
	dynamodbconn := meta.(*AWSClient).dynamodbconn
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
		attribute := make(map[string]string)
		attribute["name"] = *(attrdef.AttributeName)
		attribute["type"] = *(attrdef.AttributeType)
		attributes = append(attributes, attribute)
	}

	d.Set("attribute", attributes)

	gsiList := []interface{}{}
	for _, gsiObject := range table.GlobalSecondaryIndexes {
		gsi := make(map[string]interface{})
		gsi["write_capacity"] = gsiObject.ProvisionedThroughput.WriteCapacityUnits
		gsi["read_capacity"] = gsiObject.ProvisionedThroughput.ReadCapacityUnits
		gsi["name"] = gsiObject.IndexName
		gsiList = append(gsiList, gsi)
	}

	d.Set("global_secondary_index", gsiList)

	return nil
}

func resourceAwsDynamoDbTableDelete(d *schema.ResourceData, meta interface{}) error {
	dynamodbconn := meta.(*AWSClient).dynamodbconn

	waitForTableToBeActive(d.Id(), meta)

	log.Printf("[DEBUG] DynamoDB delete table: %s", d.Id())

	_, err := dynamodbconn.DeleteTable(&dynamodb.DeleteTableInput{
		TableName: aws.String(d.Id()),
	})
	if err != nil {
		return err
	}
	return nil
}

func createGSIFromData(data *map[string]interface{}) dynamodb.GlobalSecondaryIndex {

	projection := &dynamodb.Projection{
		ProjectionType: aws.String((*data)["projection_type"].(string)),
	}

	if (*data)["projection_type"] != "ALL" {
		non_key_attributes := []*string{}
		for _, attr := range (*data)["non_key_attributes"].([]interface{}) {
			non_key_attributes = append(non_key_attributes, aws.String(attr.(string)))
		}
		projection.NonKeyAttributes = non_key_attributes
	}

	writeCapacity := (*data)["write_capacity"].(int)
	readCapacity := (*data)["read_capacity"].(int)

	return dynamodb.GlobalSecondaryIndex{
		IndexName: aws.String((*data)["name"].(string)),
		KeySchema: []*dynamodb.KeySchemaElement{
			&dynamodb.KeySchemaElement{
				AttributeName: aws.String((*data)["hash_key"].(string)),
				KeyType:       aws.String("HASH"),
			},
			&dynamodb.KeySchemaElement{
				AttributeName: aws.String((*data)["range_key"].(string)),
				KeyType:       aws.String("RANGE"),
			},
		},
		Projection: projection,
		ProvisionedThroughput: &dynamodb.ProvisionedThroughput{
			WriteCapacityUnits: aws.Long(int64(writeCapacity)),
			ReadCapacityUnits:  aws.Long(int64(readCapacity)),
		},
	}
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
				log.Printf("[DEBUG] Sleeping for 3 seconds for %s GSI to become active", gsiName)
				time.Sleep(3 * time.Second)
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

		activeState = *(result.Table.TableStatus) == "ACTIVE"

		// Wait for a few seconds
		if !activeState {
			log.Printf("[DEBUG] Sleeping for 3 seconds for table to become active")
			time.Sleep(3 * time.Second)
		}
	}

	return nil

}
