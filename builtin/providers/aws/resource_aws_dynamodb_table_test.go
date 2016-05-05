package aws

import (
	"fmt"
	"log"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSDynamoDbTable_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSDynamoDbTableDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSDynamoDbConfigInitialState(),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckInitialAWSDynamoDbTableExists("aws_dynamodb_table.basic-dynamodb-table"),
				),
			},
			resource.TestStep{
				Config: testAccAWSDynamoDbConfigAddSecondaryGSI,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDynamoDbTableWasUpdated("aws_dynamodb_table.basic-dynamodb-table"),
				),
			},
		},
	})
}

func TestAccAWSDynamoDbTable_streamSpecification(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSDynamoDbTableDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSDynamoDbConfigStreamSpecification(),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckInitialAWSDynamoDbTableExists("aws_dynamodb_table.basic-dynamodb-table"),
					resource.TestCheckResourceAttr(
						"aws_dynamodb_table.basic-dynamodb-table", "stream_enabled", "true"),
					resource.TestCheckResourceAttr(
						"aws_dynamodb_table.basic-dynamodb-table", "stream_view_type", "KEYS_ONLY"),
				),
			},
		},
	})
}

func TestResourceAWSDynamoDbTableStreamViewType_validation(t *testing.T) {
	cases := []struct {
		Value    string
		ErrCount int
	}{
		{
			Value:    "KEYS-ONLY",
			ErrCount: 1,
		},
		{
			Value:    "RANDOM-STRING",
			ErrCount: 1,
		},
		{
			Value:    "KEYS_ONLY",
			ErrCount: 0,
		},
		{
			Value:    "NEW_AND_OLD_IMAGES",
			ErrCount: 0,
		},
		{
			Value:    "NEW_IMAGE",
			ErrCount: 0,
		},
		{
			Value:    "OLD_IMAGE",
			ErrCount: 0,
		},
	}

	for _, tc := range cases {
		_, errors := validateStreamViewType(tc.Value, "aws_dynamodb_table_stream_view_type")

		if len(errors) != tc.ErrCount {
			t.Fatalf("Expected the DynamoDB stream_view_type to trigger a validation error")
		}
	}
}

func testAccCheckAWSDynamoDbTableDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).dynamodbconn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_dynamodb_table" {
			continue
		}

		log.Printf("[DEBUG] Checking if DynamoDB table %s exists", rs.Primary.ID)
		// Check if queue exists by checking for its attributes
		params := &dynamodb.DescribeTableInput{
			TableName: aws.String(rs.Primary.ID),
		}

		_, err := conn.DescribeTable(params)
		if err == nil {
			return fmt.Errorf("DynamoDB table %s still exists. Failing!", rs.Primary.ID)
		}

		// Verify the error is what we want
		if dbErr, ok := err.(awserr.Error); ok && dbErr.Code() == "ResourceNotFoundException" {
			return nil
		}

		return err
	}

	return nil
}

func testAccCheckInitialAWSDynamoDbTableExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		fmt.Printf("[DEBUG] Trying to create initial table state!")
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No DynamoDB table name specified!")
		}

		conn := testAccProvider.Meta().(*AWSClient).dynamodbconn

		params := &dynamodb.DescribeTableInput{
			TableName: aws.String(rs.Primary.ID),
		}

		resp, err := conn.DescribeTable(params)

		if err != nil {
			fmt.Printf("[ERROR] Problem describing table '%s': %s", rs.Primary.ID, err)
			return err
		}

		table := resp.Table

		fmt.Printf("[DEBUG] Checking on table %s", rs.Primary.ID)

		if *table.ProvisionedThroughput.WriteCapacityUnits != 20 {
			return fmt.Errorf("Provisioned write capacity was %d, not 20!", table.ProvisionedThroughput.WriteCapacityUnits)
		}

		if *table.ProvisionedThroughput.ReadCapacityUnits != 10 {
			return fmt.Errorf("Provisioned read capacity was %d, not 10!", table.ProvisionedThroughput.ReadCapacityUnits)
		}

		attrCount := len(table.AttributeDefinitions)
		gsiCount := len(table.GlobalSecondaryIndexes)
		lsiCount := len(table.LocalSecondaryIndexes)

		if attrCount != 4 {
			return fmt.Errorf("There were %d attributes, not 4 like there should have been!", attrCount)
		}

		if gsiCount != 1 {
			return fmt.Errorf("There were %d GSIs, not 1 like there should have been!", gsiCount)
		}

		if lsiCount != 1 {
			return fmt.Errorf("There were %d LSIs, not 1 like there should have been!", lsiCount)
		}

		attrmap := dynamoDbAttributesToMap(&table.AttributeDefinitions)
		if attrmap["TestTableHashKey"] != "S" {
			return fmt.Errorf("Test table hash key was of type %s instead of S!", attrmap["TestTableHashKey"])
		}
		if attrmap["TestTableRangeKey"] != "S" {
			return fmt.Errorf("Test table range key was of type %s instead of S!", attrmap["TestTableRangeKey"])
		}
		if attrmap["TestLSIRangeKey"] != "N" {
			return fmt.Errorf("Test table LSI range key was of type %s instead of N!", attrmap["TestLSIRangeKey"])
		}
		if attrmap["TestGSIRangeKey"] != "S" {
			return fmt.Errorf("Test table GSI range key was of type %s instead of S!", attrmap["TestGSIRangeKey"])
		}

		return nil
	}
}

func testAccCheckDynamoDbTableWasUpdated(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No DynamoDB table name specified!")
		}

		conn := testAccProvider.Meta().(*AWSClient).dynamodbconn

		params := &dynamodb.DescribeTableInput{
			TableName: aws.String(rs.Primary.ID),
		}
		resp, err := conn.DescribeTable(params)
		table := resp.Table

		if err != nil {
			return err
		}

		attrCount := len(table.AttributeDefinitions)
		gsiCount := len(table.GlobalSecondaryIndexes)
		lsiCount := len(table.LocalSecondaryIndexes)

		if attrCount != 4 {
			return fmt.Errorf("There were %d attributes, not 4 like there should have been!", attrCount)
		}

		if gsiCount != 1 {
			return fmt.Errorf("There were %d GSIs, not 1 like there should have been!", gsiCount)
		}

		if lsiCount != 1 {
			return fmt.Errorf("There were %d LSIs, not 1 like there should have been!", lsiCount)
		}

		if dynamoDbGetGSIIndex(&table.GlobalSecondaryIndexes, "ReplacementTestTableGSI") == -1 {
			return fmt.Errorf("Could not find GSI named 'ReplacementTestTableGSI' in the table!")
		}

		if dynamoDbGetGSIIndex(&table.GlobalSecondaryIndexes, "InitialTestTableGSI") != -1 {
			return fmt.Errorf("Should have removed 'InitialTestTableGSI' but it still exists!")
		}

		attrmap := dynamoDbAttributesToMap(&table.AttributeDefinitions)
		if attrmap["TestTableHashKey"] != "S" {
			return fmt.Errorf("Test table hash key was of type %s instead of S!", attrmap["TestTableHashKey"])
		}
		if attrmap["TestTableRangeKey"] != "S" {
			return fmt.Errorf("Test table range key was of type %s instead of S!", attrmap["TestTableRangeKey"])
		}
		if attrmap["TestLSIRangeKey"] != "N" {
			return fmt.Errorf("Test table LSI range key was of type %s instead of N!", attrmap["TestLSIRangeKey"])
		}
		if attrmap["ReplacementGSIRangeKey"] != "N" {
			return fmt.Errorf("Test table replacement GSI range key was of type %s instead of N!", attrmap["ReplacementGSIRangeKey"])
		}

		return nil
	}
}

func dynamoDbGetGSIIndex(gsiList *[]*dynamodb.GlobalSecondaryIndexDescription, target string) int {
	for idx, gsiObject := range *gsiList {
		if *gsiObject.IndexName == target {
			return idx
		}
	}

	return -1
}

func dynamoDbAttributesToMap(attributes *[]*dynamodb.AttributeDefinition) map[string]string {
	attrmap := make(map[string]string)

	for _, attrdef := range *attributes {
		attrmap[*attrdef.AttributeName] = *attrdef.AttributeType
	}

	return attrmap
}

func testAccAWSDynamoDbConfigInitialState() string {
	return fmt.Sprintf(`
resource "aws_dynamodb_table" "basic-dynamodb-table" {
    name = "TerraformTestTable-%d"
		read_capacity = 10
		write_capacity = 20
		hash_key = "TestTableHashKey"
		range_key = "TestTableRangeKey"
		attribute {
			name = "TestTableHashKey"
			type = "S"
		}
		attribute {
			name = "TestTableRangeKey"
			type = "S"
		}
		attribute {
			name = "TestLSIRangeKey"
			type = "N"
		}
		attribute {
			name = "TestGSIRangeKey"
			type = "S"
		}
		local_secondary_index {
			name = "TestTableLSI"
			range_key = "TestLSIRangeKey"
			projection_type = "ALL"
		}
		global_secondary_index {
			name = "InitialTestTableGSI"
			hash_key = "TestTableHashKey"
			range_key = "TestGSIRangeKey"
			write_capacity = 10
			read_capacity = 10
			projection_type = "KEYS_ONLY"
		}
}
`, acctest.RandInt())
}

const testAccAWSDynamoDbConfigAddSecondaryGSI = `
resource "aws_dynamodb_table" "basic-dynamodb-table" {
    name = "TerraformTestTable"
		read_capacity = 20
		write_capacity = 20
		hash_key = "TestTableHashKey"
		range_key = "TestTableRangeKey"
		attribute {
			name = "TestTableHashKey"
			type = "S"
		}
		attribute {
			name = "TestTableRangeKey"
			type = "S"
		}
		attribute {
			name = "TestLSIRangeKey"
			type = "N"
		}
		attribute {
			name = "ReplacementGSIRangeKey"
			type = "N"
		}
		local_secondary_index {
			name = "TestTableLSI"
			range_key = "TestLSIRangeKey"
			projection_type = "ALL"
		}
		global_secondary_index {
			name = "ReplacementTestTableGSI"
			hash_key = "TestTableHashKey"
			range_key = "ReplacementGSIRangeKey"
			write_capacity = 5
			read_capacity = 5
			projection_type = "INCLUDE"
			non_key_attributes = ["TestNonKeyAttribute"]
		}
}
`

func testAccAWSDynamoDbConfigStreamSpecification() string {
	return fmt.Sprintf(`
resource "aws_dynamodb_table" "basic-dynamodb-table" {
    name = "TerraformTestStreamTable-%d"
	read_capacity = 10
	write_capacity = 20
	hash_key = "TestTableHashKey"
	range_key = "TestTableRangeKey"
	attribute {
		name = "TestTableHashKey"
		type = "S"
	}
	attribute {
		name = "TestTableRangeKey"
		type = "S"
	}
	attribute {
		name = "TestLSIRangeKey"
		type = "N"
	}
	attribute {
		name = "TestGSIRangeKey"
		type = "S"
	}
	local_secondary_index {
		name = "TestTableLSI"
		range_key = "TestLSIRangeKey"
		projection_type = "ALL"
	}
	global_secondary_index {
		name = "InitialTestTableGSI"
		hash_key = "TestTableHashKey"
		range_key = "TestGSIRangeKey"
		write_capacity = 10
		read_capacity = 10
		projection_type = "KEYS_ONLY"
	}
	stream_enabled = true
	stream_view_type = "KEYS_ONLY"
}
`, acctest.RandInt())
}
