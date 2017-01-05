# EdgeGrid for GoLang

[![Build Status](https://travis-ci.org/akamai-open/AkamaiOPEN-edgegrid-golang.svg?branch=master)](https://travis-ci.org/akamai-open/AkamaiOPEN-edgegrid-golang)
[![Coverage Status](https://coveralls.io/repos/github/njuettner/edgegrid/badge.svg?branch=master)](https://coveralls.io/github/njuettner/edgegrid?branch=master)
[![GoDoc](https://godoc.org/github.com/akamai-open/AkamaiOPEN-edgegrid-golang?status.svg)](https://godoc.org/github.com/akamai-open/AkamaiOPEN-edgegrid-golang)
[![Go Report Card](https://goreportcard.com/badge/github.com/akamai-open/AkamaiOPEN-edgegrid-golang)](https://goreportcard.com/report/github.com/akamai-open/AkamaiOPEN-edgegrid-golang)
[![License](http://img.shields.io/:license-apache-blue.svg)](https://github.com/akamai-open/AkamaiOPEN-edgegrid-golang/blob/master/LICENSE)

This library implements an Authentication handler for [net/http](https://golang.org/pkg/net/http/)
that provides the [Akamai {OPEN} Edgegrid Authentication](https://developer.akamai.com/introduction/Client_Auth.html) 
scheme. For more information visit the [Akamai {OPEN} Developer Community](https://developer.akamai.com).

GET Example:

```go
  package main

  import (
    "fmt"
    "github.com/akamai-open/AkamaiOPEN-edgegrid-golang"
    "io/ioutil"
    "net/http"
  )

  func main() {
    client := http.Client{}

    config := edgegrid.InitConfig("~/.edgerc", "default")

    // Retrieve all locations for diagnostic tools
    req, _ := http.NewRequest("GET", fmt.Sprintf("https://%s/diagnostic-tools/v1/locations", config.Host), nil)
    req = edgegrid.AddRequestHeader(config, req)
    resp, _ := client.Do(req)

    defer resp.Body.Close()
    byt, _ := ioutil.ReadAll(resp.Body)
    fmt.Println(string(byt))
  }
```

Parameter Example:

```go
  package main

  import (
    "fmt"
    "github.com/akamai-open/AkamaiOPEN-edgegrid-golang"
    "io/ioutil"
    "net/http"
  )

  func main() {
    client := http.Client{}

    config := edgegrid.InitConfig("~/.edgerc", "default")

    // Retrieve dig information for specified location
    req, _ := http.NewRequest("GET", fmt.Sprintf("https://%sdiagnostic-tools/v1/dig", config.Host), nil)

    q := req.URL.Query()
    q.Add("hostname", "developer.akamai.com")
    q.Add("queryType", "A")
    q.Add("location", "Auckland, New Zealand")

    req.URL.RawQuery = q.Encode()
    req = edgegrid.AddRequestHeader(config, req)
    resp, _ := client.Do(req)

    defer resp.Body.Close()
    byt, _ := ioutil.ReadAll(resp.Body)
    fmt.Println(string(byt))
  }
```

POST Example:

```go
  package main

  import (
    "fmt"
    "github.com/akamai-open/AkamaiOPEN-edgegrid-golang"
    "io/ioutil"
    "net/http"
  )

  func main() {
    client := http.Client{}

    config := edgegrid.InitConfig("~/.edgerc", "default")
    
    // Acknowledge a map
    req, _ := http.NewRequest("POST", fmt.Sprintf("https://%s/siteshield/v1/maps/1/acknowledge", config.Host), nil)
    req = edgegrid.AddRequestHeader(config, req)
    resp, _ := client.Do(req)

    defer resp.Body.Close()
    byt, _ := ioutil.ReadAll(resp.Body)
    fmt.Println(string(byt))
  }
```

PUT Example:

```go
  package main

  import (
    "fmt"
    "github.com/akamai-open/AkamaiOPEN-edgegrid-golang"
    "io/ioutil"
    "net/http"
  )

  func main() {
    client := http.Client{}

    config := edgegrid.InitConfig("~/.edgerc", "default")
    body := []byte("{\n  \"name\": \"Simple List\",\n  \"type\": \"IP\",\n  \"unique-id\": \"345_BOTLIST\",\n  \"list\": [\n    \"192.168.0.1\",\n    \"192.168.0.2\",\n  ],\n  \"sync-point\": 0\n}")
    
    // Update a Network List
    req, _ := http.NewRequest("PUT", fmt.Sprintf("https://%s/network-list/v1/network_lists/unique-id?extended=extended", config.Host), bytes.NewBuffer(body))
    req = edgegrid.AddRequestHeader(config, req)
    resp, _ := client.Do(req)

    defer resp.Body.Close()
    byt, _ := ioutil.ReadAll(resp.Body)
    fmt.Println(string(byt))
  }
```

Alternatively, your program can read it from config struct.

```go
  package main

  import (
    "fmt"
    "github.com/akamai-open/AkamaiOPEN-edgegrid-golang"
    "io/ioutil"
    "net/http"
  )

  func main() {
    client := http.Client{}
    config := edgegrid.Config{
      Host : "xxxxxx.luna.akamaiapis.net",
      ClientToken:  "xxxx-xxxxxxxxxxx-xxxxxxxxxxx",
      ClientSecret: "xxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
      AccessToken:  "xxxx-xxxxxxxxxxx-xxxxxxxxxxx",
      MaxBody:      1024,
      HeaderToSign: []string{
        "X-Test1",
        "X-Test2",
        "X-Test3",
      },
      Debug:        false,
    }
    
    // Retrieve all locations for diagnostic tools
    req, _ := http.NewRequest("GET", fmt.Sprintf("https://%s/diagnostic-tools/v1/locations", config.Host), nil)
    req = edgegrid.AddRequestHeader(config, req)
    resp, _ := client.Do(req)

    defer resp.Body.Close()
    byt, _ := ioutil.ReadAll(resp.Body)
    fmt.Println(string(byt))
  }
```

## Installation

```bash
  $ go get github.com/akamai-open/AkamaiOPEN-edgegrid-golang
```

## Contribute

1. Fork [the repository](https://github.com/njuettner/edgegrid) to start making your changes to the **master** branch
2. Send a pull request.

## Author

[Nick Juettner](mailto:hello@juni.io) - Software Engineer @ [Zalando SE](https://tech.zalando.com/)

