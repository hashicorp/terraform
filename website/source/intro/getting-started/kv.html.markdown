---
layout: "intro"
page_title: "Key/Value Data"
sidebar_current: "gettingstarted-kv"
---

# Key/Value Data

In addition to providing service discovery and integrated health checking,
Terraform provides an easy to use Key/Value store. This can be used to hold
dynamic configuration, assist in service coordination, build leader election,
and anything else a developer can think to build. The
[HTTP API](/docs/agent/http.html) fully documents the features of the K/V store.

This page assumes you have at least one Terraform agent already running.

## Simple Usage

To demonstrate how simple it is to get started, we will manipulate a few keys
in the K/V store.

Querying the agent we started in a prior page, we can first verify that
there are no existing keys in the k/v store:

```
$ curl -v http://localhost:8500/v1/kv/?recurse
* About to connect() to localhost port 8500 (#0)
*   Trying 127.0.0.1... connected
> GET /v1/kv/?recurse HTTP/1.1
> User-Agent: curl/7.22.0 (x86_64-pc-linux-gnu) libcurl/7.22.0 OpenSSL/1.0.1 zlib/1.2.3.4 libidn/1.23 librtmp/2.3
> Host: localhost:8500
> Accept: */*
>
< HTTP/1.1 404 Not Found
< X-Terraform-Index: 1
< Date: Fri, 11 Apr 2014 02:10:28 GMT
< Content-Length: 0
< Content-Type: text/plain; charset=utf-8
<
* Connection #0 to host localhost left intact
* Closing connection #0
```

Since there are no keys, we get a 404 response back.
Now, we can put a few example keys:

```
$ curl -X PUT -d 'test' http://localhost:8500/v1/kv/web/key1
true
$ curl -X PUT -d 'test' http://localhost:8500/v1/kv/web/key2?flags=42
true
$ curl -X PUT -d 'test'  http://localhost:8500/v1/kv/web/sub/key3
true
$ curl http://localhost:8500/v1/kv/?recurse
[{"CreateIndex":97,"ModifyIndex":97,"Key":"web/key1","Flags":0,"Value":"dGVzdA=="},
 {"CreateIndex":98,"ModifyIndex":98,"Key":"web/key2","Flags":42,"Value":"dGVzdA=="},
 {"CreateIndex":99,"ModifyIndex":99,"Key":"web/sub/key3","Flags":0,"Value":"dGVzdA=="}]
```

Here we have created 3 keys, each with the value of "test". Note that the
`Value` field returned is base64 encoded to allow non-UTF8
characters. For the "web/key2" key, we set a `flag` value of 42. All keys
support setting a 64bit integer flag value. This is opaque to Terraform but can
be used by clients for any purpose.

After setting the values, we then issued a GET request to retrieve multiple
keys using the `?recurse` parameter.

You can also fetch a single key just as easily:

```
$ curl http://localhost:8500/v1/kv/web/key1
[{"CreateIndex":97,"ModifyIndex":97,"Key":"web/key1","Flags":0,"Value":"dGVzdA=="}]
```

Deleting keys is simple as well. We can delete a single key by specifying the
full path, or we can recursively delete all keys under a root using "?recurse":

```
$ curl -X DELETE http://localhost:8500/v1/kv/web/sub?recurse
$ curl http://localhost:8500/v1/kv/web?recurse
[{"CreateIndex":97,"ModifyIndex":97,"Key":"web/key1","Flags":0,"Value":"dGVzdA=="},
 {"CreateIndex":98,"ModifyIndex":98,"Key":"web/key2","Flags":42,"Value":"dGVzdA=="}]
```

A key can be updated by setting a new value by issuing the same PUT request.
Additionally, Terraform provides a Check-And-Set operation, enabling atomic
key updates. This is done by providing the `?cas=` paramter with the last
`ModifyIndex` value from the GET request. For example, suppose we wanted
to update "web/key1":

```
$ curl -X PUT -d 'newval' http://localhost:8500/v1/kv/web/key1?cas=97
true
$ curl -X PUT -d 'newval' http://localhost:8500/v1/kv/web/key1?cas=97
false
```

In this case, the first CAS update succeeds because the last modify time is 97.
However the second operation fails because the `ModifyIndex` is no longer 97.

We can also make use of the `ModifyIndex` to wait for a key's value to change.
For example, suppose we wanted to wait for key2 to be modified:

```
$ curl "http://localhost:8500/v1/kv/web/key2?index=101&wait=5s"
[{"CreateIndex":98,"ModifyIndex":101,"Key":"web/key2","Flags":42,"Value":"dGVzdA=="}]
```

By providing "?index=" we are asking to wait until the key has a `ModifyIndex` greater
than 101. However the "?wait=5s" parameter restricts the query to at most 5 seconds,
returning the current, unchanged value. This can be used to efficiently wait for
key modifications. Additionally, this same technique can be used to wait for a list
of keys, waiting only until any of the keys has a newer modification time.

This is only a few example of what the API supports. For full documentation, please
reference the [HTTP API](/docs/agent/http.html).

