[![Build Status](https://travis-ci.org/ewilde/go-runscope.svg?branch=master)](https://travis-ci.org/ewilde/go-runscope)

# go-runscope
go-runscope is a [go](https://golang.org/) client library for the
[runscope api](https://www.runscope.com/docs/api)

## Installation

```
go get github.com/ewilde/go-runscope
```

## Usage
```go
package main

import (
    "fmt"
    "github.com/ewilde/go-runscope"
)

func createBucket() {
    var accessToken = "{your token}"  // See https://www.runscope.com/applications
    var teamUUID = "{your team uuid}" // See https://www.runscope.com/teams
    var client = runscope.NewClient(runscope.APIURL, accessToken)
    var bucket = &runscope.Bucket{
        Name: "My first bucket",
        Team: &runscope.Team{
            ID: teamUUID,
        },
    }

    bucket, err := client.CreateBucket(bucket)
    if err != nil {
        log.Printf("[ERROR] error creating bucket: %s", err)
    }
}
```

### All Resources and Actions
Complete examples can be found in the [examples folder](examples) or
in the unit tests

#### Bucket
```go
Client.CreateBucket(bucket *Bucket) (*Bucket, error)
...
    var bucket = &runscope.Bucket{
        Name: "My first bucket",
        Team: &runscope.Team{
            ID: teamUUID,
        },
    }
	bucket, err := client.CreateBucket(&{Bucket{Name: "test", Team}})
```


```go
Client.ReadBucket(key string) (*Bucket, error)
...
    bucket, err := client.ReadBucket("htqee6p4dhvc")
    if err != nil {
        log.Printf("[ERROR] error creating bucket: %s", err)
    }

    fmt.Printf("Bucket read successfully: %s", bucket.String())
```


```go
Client.DeleteBucket(key string)
...
    err := client.DeleteBucket("htqee6p4dhvc")
    if err != nil {
        log.Printf("[ERROR] error creating bucket: %s", err)
    }
```

#### Environment
```go
Client.CreateSharedEnvironment(environment *Environment, bucket *Bucket) (*Environment, error)
...
environment := &runscope.Environment{
		Name: "tf_environment",
		InitialVariables: map[string]string{
			"VarA" : "ValB",
			"VarB" : "ValB",
		},
		Integrations: []*runscope.Integration {
			{
				ID:              "27e48b0d-ba8e-4fe0-bcaa-dd9de08dc47d",
				IntegrationType: "pagerduty",
			},
			{
				ID:              "574f4560-0f50-41da-a2f7-bdce419ad378",
				IntegrationType: "slack",
			},
		},
	}

environment, err := client.CreateSharedEnvironment(environment, createBucket())
if err != nil {
    log.Printf("[ERROR] error creating environment: %s", err)
}
```

```go
Client.ReadSharedEnvironment(environment *Environment, bucket *Bucket) (*Environment, error)

Client.ReadTestEnvironment(environment *Environment, test *Test) (*Environment, error)

Client.UpdateSharedEnvironment(environment *Environment, bucket *Bucket) (*Environment, error)

Client.UpdateTestEnvironment(environment *Environment, test *Test) (*Environment, error)
```
#### Test
```go
Client.CreateTest(test *Test) (*Test, error) (*Environment, error)
...
    test := &Test{ Name: "tf_test", Description: "This is a tf new test", Bucket: bucket }
	test, err = client.CreateTest(newTest)
	defer client.DeleteTest(newTest)

	if err != nil {
		t.Error(err)
	}

Client.ReadTest(test *Test) (*Test, error)

Client.UpdateTest(test *Test) (*Test, error)

Client.DeleteTest(test *Test) error
```
#### Test step
```go
Client.CreateTestStep(testStep *TestStep, bucketKey string, testID string) (*TestStep, error)
...
    step := NewTestStep()
    step.StepType = "request"
    step.URL = "http://example.com"
    step.Method = "GET"
    step.Assertions = [] Assertion {{
        Source: "response_status",
        Comparison : "equal_number",
        Value: 200,
    }}

    step, err = client.CreateTestStep(step, bucket.Key, test.ID)
    if err != nil {
        t.Error(err)
    }

Client.ReadTestStep(testStep *TestStep, bucketKey string, testID string) (*TestStep, error)

Client.UpdateTestStep(testStep *TestStep, bucketKey string, testID string) (*TestStep, error)

Client.DeleteTestStep(testStep *TestStep, bucketKey string, testID string) error
```
#### Schedule
```go
Client.CreateSchedule(schedule *Schedule, bucketKey string, testID string) (*Schedule, error)
...
    schedule := NewSchedule()
    schedule.Note = "Daily schedule"
    schedule.Interval = "1d"
    schedule.EnvironmentID = environment.ID

    schedule, err = client.CreateSchedule(schedule, bucket.Key, test.ID)
    if err != nil {
        t.Error(err)
    }

Client.ReadSchedule(schedule *Schedule, bucketKey string, testID string) (*Schedule, error)

Client.UpdateSchedule(schedule *Schedule, bucketKey string, testID string) (*Schedule, error)

Client.DeleteSchedule(schedule *Schedule, bucketKey string, testID string) error
```
## Developing
### Running the tests
By default the tests requiring access to the runscope api (most of them)
will be skipped. To run the integration tests please set the following
environment variables

```bash
RUNSCOPE_ACC=true
RUNSCOPE_ACCESS_TOKEN={your access token}
RUNSCOPE_TEAM_UUID={your team uuid}
```
Access tokens can be created using the [applications](https://www.runscope.com/applications)
section of your runscope account.

Your team url can be found by taking the uuid from https://www.runscope.com/teams

## Contributing

1. Fork it ( https://github.com/ewilde/go-runscope/fork )
2. Create your feature branch (`git checkout -b my-new-feature`)
3. Make sure that `make build` passes with test running
3. Commit your changes (`git commit -am 'Add some feature'`)
4. Push to the branch (`git push origin my-new-feature`)
5. Create a new Pull Request

