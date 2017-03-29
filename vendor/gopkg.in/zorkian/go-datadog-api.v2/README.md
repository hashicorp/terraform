[![GoDoc](http://img.shields.io/badge/godoc-reference-blue.svg)](https://godoc.org/gopkg.in/zorkian/go-datadog-api.v2)
[![Build
status](https://travis-ci.org/zorkian/go-datadog-api.svg)](https://travis-ci.org/zorkian/go-datadog-api)

# Datadog API in Go

**This is the v2.0 version of the API, and has breaking changes. Use the main or v1.0 branch if you need
legacy code to be supported.**

A Go wrapper for the Datadog API. Use this library if you need to interact
with the Datadog system. You can post metrics with it if you want, but this library is probably
mostly used for automating dashboards/alerting and retrieving data (events, etc).

The source API documentation is here: <http://docs.datadoghq.com/api/>

## Installation
To use the default branch, include it in your code like:
```go
    import "github.com/zorkian/go-datadog-api"
```

Or, if you need to control which version to use, import using [gopkg.in](http://labix.org/gopkg.in). Like so:
```go
    import "gopkg.in/zorkian/go-datadog-api.v2"
```

Using go get:
```bash
go get gopkg.in/zorkian/go-datadog-api.v2
```

## USAGE
This library uses pointers to be able to verify if values are set or not (vs the default value for the type). Like
 protobuf there are helpers to enhance the API. You can decide to not use them, but you'll have to be careful handling
 nil pointers.

Using the client:
```go
    client := datadog.NewClient("api key", "application key")

    dash, err := client.GetDashboard(datadog.Int(10880))
    if err != nil {
        log.Fatalf("fatal: %s\n", err)
    }
    
    log.Printf("dashboard %d: %s\n", dash.GetId(), dash.GetTitle())
```

An example using datadog.String(), which allocates a pointer for you:
```go
	m := datadog.Monitor{
		Name: datadog.String("Monitor other things"),
		Creator: &datadog.Creator{
			Name: datadog.String("Joe Creator"),
		},
	}
```

An example using the SetXx, HasXx, GetXx and GetXxOk accessors:
```go
	m := datadog.Monitor{}
	m.SetName("Monitor all the things")
	m.SetMessage("Electromagnetic energy loss")

	// Use HasMessage(), to verify we have interest in the message.
	// Using GetMessage() always safe as it returns the actual or, if never set, default value for that type.
	if m.HasMessage() {
		fmt.Printf("Found message %s\n", m.GetMessage())
	}

	// Alternatively, use GetMessageOk(), it returns a tuple with the (default) value and a boolean expressing
	// if it was set at all:
	if v, ok := m.GetMessageOk(); ok {
		fmt.Printf("Found message %s\n", v)
	}
```

Check out the Godoc link for the available API methods and, if you can't find the one you need,
let us know (or patches welcome)!

## DOCUMENTATION

Please see: <https://godoc.org/gopkg.in/zorkian/go-datadog-api.v2>

## BUGS/PROBLEMS/CONTRIBUTING

There are certainly some, but presently no known major bugs. If you do
find something that doesn't work as expected, please file an issue on
Github:

<https://github.com/zorkian/go-datadog-api/issues>

Thanks in advance! And, as always, patches welcome!

## DEVELOPMENT
### Running tests
* Run tests tests with `make test`.
* Integration tests can be run with `make testacc`. Run specific integration tests with `make testacc TESTARGS='-run=TestCreateAndDeleteMonitor'`

The acceptance tests require _DATADOG_API_KEY_ and _DATADOG_APP_KEY_ to be available
in your environment variables.

*Warning: the integrations tests will create and remove real resources in your Datadog account.*

### Regenerating code
Accessors `HasXx`, `GetXx`, `GetOkXx` and `SetXx` are generated for each struct field type type that contains pointers.
When structs are updated a contributor has to regenerate these using `go generate` and commit these changes.
Optionally there is a make target for the generation:

```bash
make generate
```

## COPYRIGHT AND LICENSE

Please see the LICENSE file for the included license information.

Copyright 2017 by authors and contributors.
