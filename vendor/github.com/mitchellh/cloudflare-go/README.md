[![GoDoc](https://godoc.org/github.com/cloudflare/cloudflare-go?status.svg)](https://godoc.org/github.com/cloudflare/cloudflare-go)

# cloudflare

A Go library for interacting with [CloudFlare's API v4](https://api.cloudflare.com/).

# Installation

You need a working Go environment.

```
go get github.com/cloudflare/cloudflare-go
```

# Getting Started

```
package main

import (
	"fmt"

	"github.com/cloudflare/cloudflare-go"
)

var api *cloudflare.API

func main() {
	// Construct a new API object
	api = cloudflare.New(os.Getenv("CF_API_KEY"), os.Getenv("CF_API_EMAIL"))

	// Fetch the list of zones on the account
	zones, err := api.ListZones()
	if err != nil {
		fmt.Println(err)
	}
	// Print the zone names
	for _, z := range zones {
		fmt.Println(z.Name)
	}
}
```

An example application, [flarectl](cmd/flarectl), is in this repository.
