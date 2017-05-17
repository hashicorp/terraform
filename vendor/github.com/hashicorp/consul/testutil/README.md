Consul Testing Utilities
========================

This package provides some generic helpers to facilitate testing in Consul.

TestServer
==========

TestServer is a harness for managing Consul agents and initializing them with
test data. Using it, you can form test clusters, create services, add health
checks, manipulate the K/V store, etc. This test harness is completely decoupled
from Consul's core and API client, meaning it can be easily imported and used in
external unit tests for various applications. It works by invoking the Consul
CLI, which means it is a requirement to have Consul installed in the `$PATH`.

Following is an example usage:

```go
package my_program

import (
	"testing"

	"github.com/hashicorp/consul/consul/structs"
	"github.com/hashicorp/consul/testutil"
)

func TestMain(t *testing.T) {
	// Create a test Consul server
	srv1 := testutil.NewTestServer(t)
	defer srv1.Stop()

	// Create a secondary server, passing in configuration
	// to avoid bootstrapping as we are forming a cluster.
	srv2 := testutil.NewTestServerConfig(t, func(c *testutil.TestServerConfig) {
		c.Bootstrap = false
	})
	defer srv2.Stop()

	// Join the servers together
	srv1.JoinLAN(srv2.LANAddr)

	// Create a test key/value pair
	srv1.SetKV("foo", []byte("bar"))

	// Create lots of test key/value pairs
	srv1.PopulateKV(map[string][]byte{
		"bar": []byte("123"),
		"baz": []byte("456"),
	})

	// Create a service
	srv1.AddService("redis", structs.HealthPassing, []string{"master"})

	// Create a service check
	srv1.AddCheck("service:redis", "redis", structs.HealthPassing)

	// Create a node check
	srv1.AddCheck("mem", "", structs.HealthCritical)

	// The HTTPAddr field contains the address of the Consul
	// API on the new test server instance.
	println(srv1.HTTPAddr)
}
```
