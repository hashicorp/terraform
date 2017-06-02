package runscope

import (
	"fmt"
	"time"
	"encoding/json"
)

// Test represents the details for a runscope test. See https://www.runscope.com/docs/api/tests
type Test struct {
	ID                   string         `json:"id,omitempty"`
	Bucket               *Bucket        `json:"-"`
	Name                 string         `json:"name,omitempty"`
	Description          string         `json:"description,omitempty"`
	CreatedAt            *time.Time     `json:"created_at,omitempty"`
	CreatedBy            *Contact       `json:"created_by,omitempty"`
	DefaultEnvironmentID string         `json:"default_environment_id,omitempty"`
	ExportedAt           *time.Time     `json:"exported_at,omitempty"`
	Environments         []*Environment `json:"environments"`
	LastRun              *TestRun       `json:"last_run"`
	Steps                []*TestStep    `json:"steps"`
}

// TestRun represents the details of the last time the test ran
type TestRun struct {
	RemoteAgentUUID     string     `json:"remote_agent_uuid,omitempty"`
	FinishedAt          *time.Time `json:"finished_at,omitempty"`
	ErrorCount          int        `json:"error_count,omitempty"`
	MessageSuccess      int        `json:"message_success,omitempty"`
	TestUUID            string     `json:"test_uuid,omitempty"`
	ID                  string     `json:"id,omitempty"`
	ExtractorSuccess    int        `json:"extractor_success,omitempty"`
	UUID                string     `json:"uuid,omitempty"`
	EnvironmentUUID     string     `json:"environment_uuid,omitempty"`
	EnvironmentName     string     `json:"environment_name,omitempty"`
	Source              string     `json:"source,omitempty"`
	RemoteAgentName     string     `json:"remote_agent_name,omitempty"`
	RemoteAgent         string     `json:"remote_agent,omitempty"`
	Status              string     `json:"status,omitempty"`
	BucketKey           string     `json:"bucket_key,omitempty"`
	RemoteAgentVersion  string     `json:"remote_agent_version,omitempty"`
	SubstitutionSuccess int        `json:"substitution_success,omitempty"`
	MessageCount        int        `json:"message_count,omitempty"`
	ScriptCount         int        `json:"script_count,omitempty"`
	SubstitutionCount   int        `json:"substitution_count,omitempty"`
	ScriptSuccess       int        `json:"script_success,omitempty"`
	AssertionCount      int        `json:"assertion_count,omitempty"`
	AssertionSuccess    int        `json:"assertion_success,omitempty"`
	CreatedAt           *time.Time `json:"created_at,omitempty"`
	Messages            []string   `json:"messages,omitempty"`
	ExtractorCount      int        `json:"extractor_count,omitempty"`
	TemplateUUIDs       []string   `json:"template_uuids,omitempty"`
	Region              string     `json:"region,omitempty"`
}

// TestStep represents each step that makes up part of the test. See https://www.runscope.com/docs/api/steps
type TestStep struct {
	URL        string                 `json:"url,omitempty"`
	Variables  []*Variable            `json:"variables,omitempty"`
	Args       map[string]interface{} `json:"args,omitempty"`
	StepType   string                 `json:"step_type,omitempty"`
	Auth       map[string]string      `json:"auth,omitempty"`
	ID         string                 `json:"id,omitempty"`
	Body       string                 `json:"body,omitempty"`
	Note       string                 `json:"note,omitempty"`
	Headers    map[string][]string    `json:"headers,omitempty"`
	RequestID  string                 `json:"request_id,omitempty"`
	Assertions []*Assertion           `json:"assertions,omitempty"`
	Scripts    []*Script               `json:"scripts,omitempty"`
	Method     string                 `json:"method,omitempty"`
}

// Variable allow you to extract data from request, subtest, and Ghost Inspector steps for use in subsequent steps in the test. Similar to Assertions, each variable is defined by a name, source, and property. See https://www.runscope.com/docs/api/steps#variables
type Variable struct {
	Name     string `json:"name,omitempty"`
	Property string `json:"property,omitempty"`
	Source   string `json:"source,omitempty"`
}

// Assertion allow you to specify success criteria for a given request, Ghost Inspector, subtest, or condition step. Each assertion is defined by a source, property, comparison, and value. See https://www.runscope.com/docs/api/steps#assertions
type Assertion struct {
	Comparison string      `json:"comparison,omitempty"`
	Value      interface{} `json:"value,omitempty"`
	Source     string      `json:"source,omitempty"`
	Property   string      `json:"property,omitempty"`
}

// Script not sure how this is used, currently not documented, but looks like a javascript string that gets evaluated? See See https://www.runscope.com/docs/api/steps
type Script struct {
	Value string `json:"value"`
}
/*
"request_id": "2dbfb5d2-3b5a-499c-9550-b06f9a475feb",
"assertions": [
{
"comparison": "equal_number",
"value": 200,
"source": "response_status"
}
],
"scripts": [],
"before_scripts": [],
"data": "",
"method": "GET"
*/

// Contact details
type Contact struct {
	Email string     `json:"email,omitempty"`
        ID    string     `json:"id"`
        Name  string     `json:"name,omitempty"`
}

// NewTest creates a new test struct
func NewTest() *Test {
	return &Test { Bucket: &Bucket{}}
}

// CreateTest creates a new runscope test. See https://www.runscope.com/docs/api/tests#create
func (client *Client) CreateTest(test *Test) (*Test, error) {
	newResource, error := client.createResource(test, "test", test.Name,
		fmt.Sprintf("/buckets/%s/tests", test.Bucket.Key))
	if error != nil {
		return nil, error
	}

	newTest, error := getTestFromResponse(newResource.Data)
	if error != nil {
		return nil, error
	}

	newTest.Bucket = test.Bucket
	return newTest, nil
}

// ReadTest list details about an existing test. See https://www.runscope.com/docs/api/tests#detail
func (client *Client) ReadTest(test *Test) (*Test, error) {
	resource, error := client.readResource("test", test.ID, fmt.Sprintf("/buckets/%s/tests/%s", test.Bucket.Key, test.ID))
	if error != nil {
		return nil, error
	}

	readTest, error := getTestFromResponse(resource.Data)
	if error != nil {
		return nil, error
	}

	readTest.Bucket = test.Bucket
	return readTest, nil
}

// UpdateTest update an existing test. See https://www.runscope.com/docs/api/tests#modifying
func (client *Client) UpdateTest(test *Test) (*Test, error) {
	resource, error := client.updateResource(test, "test", test.ID, fmt.Sprintf("/buckets/%s/tests/%s", test.Bucket.Key, test.ID))
	if error != nil {
		return nil, error
	}

	readTest, error := getTestFromResponse(resource.Data)
	if error != nil {
		return nil, error
	}

	readTest.Bucket = test.Bucket
	return readTest, nil
}

// DeleteTest delete an existing test. See https://www.runscope.com/docs/api/tests#delete
func (client *Client) DeleteTest(test *Test) error {
	return client.deleteResource("test", test.ID, fmt.Sprintf("/buckets/%s/tests/%s", test.Bucket.Key, test.ID))
}

func (test *Test) String() string {
	value, err := json.Marshal(test)
	if err != nil {
		return ""
	}

	return string(value)
}

func getTestFromResponse(response interface{}) (*Test, error) {
	test := new(Test)
	err := decode(test, response)
	return test, err
}
