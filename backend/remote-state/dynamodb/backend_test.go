package dynamodb

import (
	"fmt"
	"os"
	"reflect"
	"testing"
	"time"
	//"strconv"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/configs/hcl2shim"
	"github.com/hashicorp/terraform/state/remote"
	"github.com/hashicorp/terraform/states"

	//"github.com/hashicorp/hcl/v2/hcldec"
)

// verify that we are doing ACC tests or the S3 tests specifically
func testACC(t *testing.T) {
	skip := os.Getenv("TF_ACC") == "" && os.Getenv("TF_DYNAMODB_TEST") == ""
	if skip {
		t.Log("dynamodb backend tests require setting TF_ACC or TF_DYNAMODB_TEST")
		t.Skip()
	}
	if os.Getenv("AWS_DEFAULT_REGION") == "" {
		os.Setenv("AWS_DEFAULT_REGION", "us-west-2")
	}
}

func TestBackend_impl(t *testing.T) {
	var _ backend.Backend = new(Backend)
}

func TestBackendConfig(t *testing.T) {
	testACC(t)

	config := map[string]interface{}{
		"state_table": "tf-test",
		"hash":        "state",
		"region":      "eu-west-1",
		"lock_table":  "dynamoTable",
	}

	b := backend.TestBackendConfig(t, New(), backend.TestWrapConfig(config)).(*Backend)

	if b.tableName != "tf-test" {
		t.Fatalf("Incorrect tableName was populated")
	}
	if b.hashName != "state" {
		t.Fatalf("Incorrect hashName was populated")
	}

	credentials, err := b.dynClient.Config.Credentials.Get()
	if err != nil {
		t.Fatalf("Error when requesting credentials")
	}
	if credentials.AccessKeyID == "" {
		t.Fatalf("No Access Key Id was populated")
	}
	if credentials.SecretAccessKey == "" {
		t.Fatalf("No Secret Access Key was populated")
	}
}

func TestBackendSchema(t *testing.T) {
	testACC(t)

	config0 := map[string]interface{}{
		"state_table": "tf-test",
		"hash":        "state",
		"region":      "eu-west-1",
		"lock_table":  "dynamoTable",
	}

	b0 := backend.TestBackendConfig(t, New(), backend.TestWrapConfig(config0)).(*Backend)

	createDynamoDBTable(t, b0.dynClient, "tf-test", "state")
	defer deleteDynamoDBTable(t, b0.dynClient, "tf-test")
	createDynamoDBTable(t, b0.dynClient, "dynamoTable", "lock")
	defer deleteDynamoDBTable(t, b0.dynClient, "dynamoTable")

	b0 = backend.TestBackendConfig(t, New(), backend.TestWrapConfig(config0)).(*Backend)

	//config1 := map[string]interface{}{
	//	"state_table": "dynamoTable",
	//	"hash":        "state",
	//	"region":      "eu-west-1",
	//	"lock_table":  "tf-test",
	//}

	//b0 = backend.TestBackendConfig(t, New(), backend.TestWrapConfig(config1)).(*Backend)
	//schema := b0.ConfigSchema()
	//spec := schema.DecoderSpec()
	//obj, _ := hcldec.Decode(backend.TestWrapConfig(config1), spec, nil)
	//diags = diags.Append(decDiags)

	//confDiags := b0.Configure(obj)


	//fmt.Println(schema)
}

func TestBigScan(t *testing.T) {
	testACC(t)

	config := map[string]interface{}{
		"state_table": "tf-test",
		"hash":        "state",
		"region":      "eu-west-1",
	}

	b := backend.TestBackendConfig(t, New(), backend.TestWrapConfig(config)).(*Backend)

	createDynamoDBTable(t, b.dynClient, "tf-test", "state")
	defer deleteDynamoDBTable(t, b.dynClient, "tf-test")

	s := states.NewState()
	fmt.Println(s)
	//client := &RemoteClient{
	//	dynClient: b.dynClient,
	//	tableName: b.tableName,
	//	path:      b.path("s1"),
	//}
	//N := 1000
	//for i := 0; i < N; i++ {
	//	client.path = b.path("s"+strconv.Itoa(i))
	//	stateMgr := &remote.State{Client: client}
	//	stateMgr.WriteState(s)
	//	if err := stateMgr.PersistState(); err != nil {
	//		t.Fatal(err)
	//	}	
	//}
	//b = backend.TestBackendConfig(t, New(), backend.TestWrapConfig(config)).(*Backend)
	//fmt.Println(b)
}


func TestBackendConfig_invalidKey(t *testing.T) {
	testACC(t)

	cfg0 := hcl2shim.HCL2ValueFromConfigValue(map[string]interface{}{
		"state_table": "tf-test",
		"hash":        "/leading-slash",
		"region":      "eu-west-1",
		"lock_table":  "dynamoTable",
	})

	_, diags := New().PrepareConfig(cfg0)
	if !diags.HasErrors() {
		t.Fatal("expected config validation error")
	}

	cfg1 := hcl2shim.HCL2ValueFromConfigValue(map[string]interface{}{
		"state_table": "tf-test",
		"hash":        "leading-slash=",
		"region":      "eu-west-1",
		"lock_table":  "dynamoTable",
	})

	_, diags = New().PrepareConfig(cfg1)
	if !diags.HasErrors() {
		t.Fatal("expected config validation error")
	}

	cfg2 := hcl2shim.HCL2ValueFromConfigValue(map[string]interface{}{
		"state_table": "tf-test=",
		"hash":        "leading-slash",
		"region":      "eu-west-1",
		"lock_table":  "dynamoTable",
	})

	_, diags = New().PrepareConfig(cfg2)
	if !diags.HasErrors() {
		t.Fatal("expected config validation error")
	}

	cfg3 := hcl2shim.HCL2ValueFromConfigValue(map[string]interface{}{
		"state_table": "tf-test",
		"hash":        "leading-slash",
		"region":      "eu-west-1",
		"lock_table":  "dynamoTable/",
	})

	_, diags = New().PrepareConfig(cfg3)
	if !diags.HasErrors() {
		t.Fatal("expected config validation error")
	}

	cfg4 := hcl2shim.HCL2ValueFromConfigValue(map[string]interface{}{
		"state_table":          "tf-test",
		"hash":                 "leading-slash",
		"region":               "eu-west-1",
		"lock_table":           "dynamoTable",
		"workspace_key_prefix": "=/",
	})

	_, diags = New().PrepareConfig(cfg4)
	if !diags.HasErrors() {
		t.Fatal("expected config validation error")
	}
}

func TestBackend(t *testing.T) {
	testACC(t)

	tableName := fmt.Sprintf("terraform-remote-dynamodb-state-%x", time.Now().Unix())
	hashName := "testState"

	b := backend.TestBackendConfig(t, New(), backend.TestWrapConfig(map[string]interface{}{
		"state_table": tableName,
		"hash":        hashName,
	})).(*Backend)

	createDynamoDBTable(t, b.dynClient, tableName, "state")
	defer deleteDynamoDBTable(t, b.dynClient, tableName)

	backend.TestBackendStates(t, b)
}

func TestBackendLocked(t *testing.T) {
	testACC(t)

	tableName := fmt.Sprintf("terraform-remote-dynamodb-state-%x", time.Now().Unix())
	lockName := fmt.Sprintf("terraform-remote-dynamodb-lock-%x", time.Now().Unix())
	hashName := "testState"

	b1 := backend.TestBackendConfig(t, New(), backend.TestWrapConfig(map[string]interface{}{
		"state_table": tableName,
		"hash":        hashName,
		"lock_table":  lockName,
	})).(*Backend)

	b2 := backend.TestBackendConfig(t, New(), backend.TestWrapConfig(map[string]interface{}{
		"state_table": tableName,
		"hash":        hashName,
		"lock_table":  lockName,
	})).(*Backend)

	createDynamoDBTable(t, b1.dynClient, tableName, "state")
	defer deleteDynamoDBTable(t, b1.dynClient, tableName)
	createDynamoDBTable(t, b1.dynClient, lockName, "lock")
	defer deleteDynamoDBTable(t, b1.dynClient, lockName)

	backend.TestBackendStateLocks(t, b1, b2)
	backend.TestBackendStateForceUnlock(t, b1, b2)
}

// add some extra junk in S3 to try and confuse the env listing.
func TestBackendWorkspaces(t *testing.T) {
	testACC(t)

	tableName := fmt.Sprintf("terraform-remote-dynamodb-state-%x", time.Now().Unix())
	hashName := "test_state_tfstate"

	b := backend.TestBackendConfig(t, New(), backend.TestWrapConfig(map[string]interface{}{
		"state_table": tableName,
		"hash":        hashName,
	})).(*Backend)

	createDynamoDBTable(t, b.dynClient, tableName, "state")
	defer deleteDynamoDBTable(t, b.dynClient, tableName)

	// put multiple states in old env paths.
	s1 := states.NewState()
	s2 := states.NewState()

	// RemoteClient to Put things in various paths
	client := &RemoteClient{
		dynClient: b.dynClient,
		tableName: b.tableName,
		path:      b.path("s1"),
	}

	stateMgr := &remote.State{Client: client}
	stateMgr.WriteState(s1)
	if err := stateMgr.PersistState(); err != nil {
		t.Fatal(err)
	}

	client = &RemoteClient{
		dynClient: b.dynClient,
		tableName: b.tableName,
		path:      b.path("s2"),
	}

	stateMgr = &remote.State{Client: client}
	stateMgr.WriteState(s2)
	if err := stateMgr.PersistState(); err != nil {
		t.Fatal(err)
	}

	s2Lineage := stateMgr.StateSnapshotMeta().Lineage

	if err := checkStateList(b, []string{"default", "s1", "s2"}); err != nil {
		t.Fatal(err)
	}

	// delete the real workspace
	if err := b.DeleteWorkspace("s2"); err != nil {
		t.Fatal(err)
	}

	if err := checkStateList(b, []string{"default", "s1"}); err != nil {
		t.Fatal(err)
	}

	// fetch that state again, which should produce a new lineage
	s2Mgr, err := b.StateMgr("s2")
	if err != nil {
		t.Fatal(err)
	}
	if err := s2Mgr.RefreshState(); err != nil {
		t.Fatal(err)
	}

	if s2Mgr.(*remote.State).StateSnapshotMeta().Lineage == s2Lineage {
		t.Fatal("state s2 was not deleted")
	}
	s2 = s2Mgr.State()
	s2Lineage = stateMgr.StateSnapshotMeta().Lineage

	// make sure s2 is OK
	s2Mgr, err = b.StateMgr("s2")
	if err != nil {
		t.Fatal(err)
	}
	if err := s2Mgr.RefreshState(); err != nil {
		t.Fatal(err)
	}
	if stateMgr.StateSnapshotMeta().Lineage != s2Lineage {
		t.Fatal("we got the wrong state for s2")
	}
	if err := checkStateList(b, []string{"default", "s1", "s2"}); err != nil {
		t.Fatal(err)
	}
}

// ensure we can separate the workspace prefix when it also matches the prefix
// of the workspace name itself.
func TestBackendPrefixInWorkspace(t *testing.T) {
	testACC(t)
	tableName := fmt.Sprintf("terraform-remote-dynamodb-state-%x", time.Now().Unix())
	hashName := "test-env.tfstate"

	b := backend.TestBackendConfig(t, New(), backend.TestWrapConfig(map[string]interface{}{
		"state_table":          tableName,
		"hash":                 hashName,
		"workspace_key_prefix": "env",
	})).(*Backend)

	createDynamoDBTable(t, b.dynClient, tableName, "state")
	defer deleteDynamoDBTable(t, b.dynClient, tableName)

	// get a state that contains the prefix as a substring
	sMgr, err := b.StateMgr("env-1")
	if err != nil {
		t.Fatal(err)
	}
	if err := sMgr.RefreshState(); err != nil {
		t.Fatal(err)
	}

	if err := checkStateList(b, []string{"default", "env-1"}); err != nil {
		t.Fatal(err)
	}
}

func TestKeyEnv(t *testing.T) {
	testACC(t)
	table0Name := fmt.Sprintf("terraform-remote-dynamodb-state-%x-0", time.Now().Unix())
	hashName := "some_paths_tfstate"

	b0 := backend.TestBackendConfig(t, New(), backend.TestWrapConfig(map[string]interface{}{
		"state_table":          table0Name,
		"hash":                 hashName,
		"workspace_key_prefix": "",
	})).(*Backend)

	createDynamoDBTable(t, b0.dynClient, table0Name, "state")
	defer deleteDynamoDBTable(t, b0.dynClient, table0Name)

	table1Name := fmt.Sprintf("terraform-remote-dynamodb-state-%x-1", time.Now().Unix())
	workspaceKeyPrefix := "project_env:"

	b1 := backend.TestBackendConfig(t, New(), backend.TestWrapConfig(map[string]interface{}{
		"state_table":          table1Name,
		"hash":                 hashName,
		"workspace_key_prefix": workspaceKeyPrefix,
	})).(*Backend)

	createDynamoDBTable(t, b1.dynClient, table1Name, "state")
	defer deleteDynamoDBTable(t, b1.dynClient, table1Name)

	table2Name := fmt.Sprintf("terraform-remote-dynamodb-state-%x-2", time.Now().Unix())
	b2 := backend.TestBackendConfig(t, New(), backend.TestWrapConfig(map[string]interface{}{
		"state_table": table2Name,
		"hash":        hashName,
	})).(*Backend)

	createDynamoDBTable(t, b2.dynClient, table2Name, "state")
	defer deleteDynamoDBTable(t, b2.dynClient, table2Name)

	if err := testGetWorkspaceForKey(b0, hashName, ""); err != nil {
		t.Fatal(err)
	}

	ws1 := "ws1"
	if err := testGetWorkspaceForKey(b0, ws1+"/"+hashName, ws1); err != nil {
		t.Fatal(err)
	}

	if err := testGetWorkspaceForKey(b1, workspaceKeyPrefix+"="+ws1+"/"+hashName, ws1); err != nil {
		t.Fatal(err)
	}

	ws2 := "ws2"
	if err := testGetWorkspaceForKey(b1, workspaceKeyPrefix+"="+ws2+"/"+hashName, ws2); err != nil {
		t.Fatal(err)
	}

	defaultWorkspaceKeyPrefix := "workspace"
	ws3 := "ws3"
	if err := testGetWorkspaceForKey(b2, defaultWorkspaceKeyPrefix+"="+ws3+"/"+hashName, ws3); err != nil {
		t.Fatal(err)
	}

	backend.TestBackendStates(t, b1)
	backend.TestBackendStates(t, b2)
	backend.TestBackendStates(t, b0)
}

func testGetWorkspaceForKey(b *Backend, key string, expected string) error {
	if actual := b.keyEnv(key); actual != expected {
		return fmt.Errorf("incorrect workspace for key[%q]. Expected[%q]: Actual[%q]", key, expected, actual)
	}
	return nil
}

func checkStateList(b backend.Backend, expected []string) error {
	states, err := b.Workspaces()
	if err != nil {
		return err
	}

	if !reflect.DeepEqual(states, expected) {
		return fmt.Errorf("incorrect states listed: %q", states)
	}
	return nil
}

// create the dynamoDB table, and wait until we can query it.
func createDynamoDBTable(t *testing.T, dynClient *dynamodb.DynamoDB, tableName string, dbtype string) {
	var createInput *dynamodb.CreateTableInput
	if dbtype == "lock" {
		createInput = &dynamodb.CreateTableInput{
			AttributeDefinitions: []*dynamodb.AttributeDefinition{
				{
					AttributeName: aws.String("LockID"),
					AttributeType: aws.String("S"),
				},
			},
			KeySchema: []*dynamodb.KeySchemaElement{
				{
					AttributeName: aws.String("LockID"),
					KeyType:       aws.String("HASH"),
				},
			},
			ProvisionedThroughput: &dynamodb.ProvisionedThroughput{
				ReadCapacityUnits:  aws.Int64(5),
				WriteCapacityUnits: aws.Int64(5),
			},
			TableName: aws.String(tableName),
		}
	}

	if dbtype == "state" {
		createInput = &dynamodb.CreateTableInput{
			AttributeDefinitions: []*dynamodb.AttributeDefinition{
				{
					AttributeName: aws.String("StateID"),
					AttributeType: aws.String("S"),
				},
				{
					AttributeName: aws.String("SegmentID"),
					AttributeType: aws.String("S"),
				},
			},
			KeySchema: []*dynamodb.KeySchemaElement{
				{
					AttributeName: aws.String("StateID"),
					KeyType:       aws.String("HASH"),
				},
				{
					AttributeName: aws.String("SegmentID"),
					KeyType:       aws.String("RANGE"),
				},
			},
			ProvisionedThroughput: &dynamodb.ProvisionedThroughput{
				ReadCapacityUnits:  aws.Int64(5),
				WriteCapacityUnits: aws.Int64(5),
			},
			TableName: aws.String(tableName),
		}
	}

	fmt.Println("Creating dynamodb table", createInput)

	_, err := dynClient.CreateTable(createInput)
	if err != nil {
		t.Fatal(err)
	}

	// now wait until it's ACTIVE
	start := time.Now()
	time.Sleep(time.Second)

	describeInput := &dynamodb.DescribeTableInput{
		TableName: aws.String(tableName),
	}

	for {
		resp, err := dynClient.DescribeTable(describeInput)
		if err != nil {
			t.Fatal(err)
		}

		if *resp.Table.TableStatus == "ACTIVE" {
			return
		}

		if time.Since(start) > time.Minute {
			t.Fatalf("timed out creating DynamoDB table %s", tableName)
		}

		time.Sleep(3 * time.Second)
	}

}

func deleteDynamoDBTable(t *testing.T, dynClient *dynamodb.DynamoDB, tableName string) {
	params := &dynamodb.DeleteTableInput{
		TableName: aws.String(tableName),
	}
	fmt.Println("Deleting dynamodb table", params)
	_, err := dynClient.DeleteTable(params)
	if err != nil {
		t.Logf("WARNING: Failed to delete the test DynamoDB table %q. It has been left in your AWS account and may incur charges. (error was %s)", tableName, err)
	}
}
