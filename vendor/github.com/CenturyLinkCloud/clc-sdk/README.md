CLC SDK (for go!) [![Build Status](https://travis-ci.org/CenturyLinkCloud/clc-sdk.svg?branch=master)](https://travis-ci.org/CenturyLinkCloud/clc-sdk) [![Coverage Status](https://coveralls.io/repos/mikebeyer/clc-sdk/badge.svg?branch=master&service=github)](https://coveralls.io/github/mikebeyer/clc-sdk?branch=master)
======

Installation
---------------------

```sh
$ go get github.com/CenturyLinkCloud/clc-sdk
$ make deps
$ make test
```


Configuration
-------
The SDK supports the following helpers for creating your configuration 


Reading from the environment

```go
config, _ := api.EnvConfig()
```

Reading from a file


```go
config, _ := api.FileConfig("./config.json")

```

Direct configuration

```go
config, _ := api.NewConfig(un, pwd)
// defaults:
config.Alias = "" // resolved on Authentication
config.UserAgent = "CenturyLinkCloud/clc-sdk"
config.BaseURI = "https://api.ctl.io/v2"

```

Enable http wire tracing with env var `DEBUG=on`.

Additionally, callers of the SDK should set `config.UserAgent` to identify to platform appropriately.


Examples
-------
To create a new server

```go
client := clc.New(api.EnvConfig())

server := server.Server{
		Name:           "server",
		CPU:            1,
		MemoryGB:       1,
		GroupID:        "GROUP-ID",
		SourceServerID: "UBUNTU-14-64-TEMPLATE",
		Type:           "standard",
	}

resp, _ := client.Server.Create(server)
```

Check status of a server build

```go
resp, _ := client.Server.Create(server)

status, _ := client.Status.Get(resp.GetStatusID())
```

Async polling for complection

```go
resp, _ := client.Server.Create(server)

poll := make(chan *status.Response, 1)
service.Status.Poll(resp.GetStatusID(), poll)

status := <- poll
```

License
-------
This project is licensed under the [Apache License v2.0](http://www.apache.org/licenses/LICENSE-2.0.html).
