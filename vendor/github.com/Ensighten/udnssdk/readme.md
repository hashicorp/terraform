# udnssdk - An UltraDNS SDK for Go

This is a golang SDK for the UltraDNS REST API. It's not feature complete, and currently is only known to be used for Terraform's `ultradns` provider.

Full API docs are available at [godoc](https://godoc.org/github.com/Ensighten/udnssdk)

## Example

```go
package main

import (
	"fmt"
	"log"

	"github.com/Ensighten/udnssdk"
)

func main() {
	client := udnssdk.NewClient("username", "password", udnssdk.DefaultTestBaseURL)
	if client == nil {
		log.Fatalf("Failed to create client")
	}

	fmt.Printf("---- Query RRSets\n")
	rrsetkey := RRSetKey{
		Zone: "domain.com",
		Type: "ANY",
		Name: "",
	}
	rrsets, err := client.RRSets.Select(rrsetkey)
	if err != nil {
		log.Fatalf(err)
	}
	fmt.Printf("%+v\n", rrsets)

	fmt.Printf("---- Create RRSet\n")
	rrsetkey = RRSetKey{
		Zone: "domain.com",
		Type: "A",
		Name: "test",
	}
	rrset := udnssdk.RRSet{
		OwnerName: r.Name,
		RRType:    r.Type,
		TTL:       300,
		RData:     []string{"127.0.0.1"},
	}
	resp, err := client.RRSets.Create(rrsetkey, rrset)
	if err != nil {
		log.Fatalf(err)
	}
	fmt.Printf("Response: %+v\n", resp)

	fmt.Printf("---- Update RRSet\n")
	rrset = udnssdk.RRSet{
		OwnerName: r.Name,
		RRType:    r.Type,
		TTL:       300,
		RData:     []string{"127.0.0.2"},
	}
	resp, err := client.RRSets.Update(rrsetkey, rrset)
	if err != nil {
		log.Fatalf(err)
	}
	fmt.Printf("Response: %+v\n", resp)

	fmt.Printf("---- Delete RRSet\n")
	resp, err := client.RRSets.Delete(rrsetkey)
	if err != nil {
		log.Fatalf(err)
	}
	fmt.Printf("Response: %+v\n", resp)
}
```

## Thanks

* Originally started as a modified version of [weppos/go-dnsimple](https://github.com/weppos/go-dnsimple)
* Designed to add UltraDNS support to [terraform](http://terraform.io)
* And for other languages, be sure to check out [UltraDNS's various SDKs](https://github.com/ultradns)
