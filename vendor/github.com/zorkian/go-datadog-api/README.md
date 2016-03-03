[![GoDoc](http://img.shields.io/badge/godoc-reference-blue.svg)](http://godoc.org/github.com/zorkian/go-datadog-api)
[![Build
status](https://travis-ci.org/zorkian/go-datadog-api.svg)](https://travis-ci.org/zorkian/go-datadog-api)

# Datadog API in Go

Hi!

This is a Go wrapper for the Datadog API. You should use this library if you need to interact
with the Datadog system. You can post metrics with it if you want, but this library is probably
mostly used for automating dashboards/alerting and retrieving data (events, etc).

The source API documentation is here: <http://docs.datadoghq.com/api/>

## USAGE

To use this project, include it in your code like:

``` go
    import "github.com/zorkian/go-datadog-api"
```

Then, you can work with it:

``` go
    client := datadog.NewClient("api key", "application key")
    
    dash, err := client.GetDashboard(10880)
    if err != nil {
        log.Fatalf("fatal: %s\n", err)
    }
    log.Printf("dashboard %d: %s\n", dash.Id, dash.Title)
```

That's all; it's pretty easy to use. Check out the Godoc link for the
available API methods and, if you can't find the one you need,
let us know (or patches welcome)!

## DOCUMENTATION

Please see: <http://godoc.org/github.com/zorkian/go-datadog-api>

## BUGS/PROBLEMS/CONTRIBUTING

There are certainly some, but presently no known major bugs. If you do
find something that doesn't work as expected, please file an issue on
Github:

<https://github.com/zorkian/go-datadog-api/issues>

Thanks in advance! And, as always, patches welcome!

## DEVELOPMENT

* Get dependencies with `make updatedeps`.
* Run tests tests with `make test`.
* Integration tests can be run with `make testacc`.

The acceptance tests require _DATADOG_API_KEY_ and _DATADOG_APP_KEY_ to be available
in your environment variables.

*Warning: the integrations tests will create and remove real resources in your Datadog
account*

## COPYRIGHT AND LICENSE

Please see the LICENSE file for the included license information.

Copyright 2013 by authors and contributors.
