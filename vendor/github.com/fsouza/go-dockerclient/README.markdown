# go-dockerclient

[![Travis Build Status](https://travis-ci.org/fsouza/go-dockerclient.svg?branch=master)](https://travis-ci.org/fsouza/go-dockerclient)
[![AppVeyor Build Status](https://ci.appveyor.com/api/projects/status/4m374pti06ubg2l7?svg=true)](https://ci.appveyor.com/project/fsouza/go-dockerclient)
[![GoDoc](https://img.shields.io/badge/api-Godoc-blue.svg?style=flat-square)](https://godoc.org/github.com/fsouza/go-dockerclient)

This package presents a client for the Docker remote API. It also provides
support for the extensions in the [Swarm API](https://docs.docker.com/swarm/swarm-api/).

This package also provides support for docker's network API, which is a simple
passthrough to the libnetwork remote API.  Note that docker's network API is
only available in docker 1.8 and above, and only enabled in docker if
DOCKER_EXPERIMENTAL is defined during the docker build process.

For more details, check the [remote API
documentation](http://docs.docker.com/engine/reference/api/docker_remote_api/).

## Example

```go
package main

import (
	"fmt"

	"github.com/fsouza/go-dockerclient"
)

func main() {
	endpoint := "unix:///var/run/docker.sock"
	client, err := docker.NewClient(endpoint)
	if err != nil {
		panic(err)
	}
	imgs, err := client.ListImages(docker.ListImagesOptions{All: false})
	if err != nil {
		panic(err)
	}
	for _, img := range imgs {
		fmt.Println("ID: ", img.ID)
		fmt.Println("RepoTags: ", img.RepoTags)
		fmt.Println("Created: ", img.Created)
		fmt.Println("Size: ", img.Size)
		fmt.Println("VirtualSize: ", img.VirtualSize)
		fmt.Println("ParentId: ", img.ParentID)
	}
}
```

## Using with TLS

In order to instantiate the client for a TLS-enabled daemon, you should use
NewTLSClient, passing the endpoint and path for key and certificates as
parameters.

```go
package main

import (
	"fmt"

	"github.com/fsouza/go-dockerclient"
)

func main() {
	endpoint := "tcp://[ip]:[port]"
	path := os.Getenv("DOCKER_CERT_PATH")
	ca := fmt.Sprintf("%s/ca.pem", path)
	cert := fmt.Sprintf("%s/cert.pem", path)
	key := fmt.Sprintf("%s/key.pem", path)
	client, _ := docker.NewTLSClient(endpoint, cert, key, ca)
	// use client
}
```

If using [docker-machine](https://docs.docker.com/machine/), or another
application that exports environment variables `DOCKER_HOST`,
`DOCKER_TLS_VERIFY`, `DOCKER_CERT_PATH`, you can use NewClientFromEnv.


```go
package main

import (
	"fmt"

	"github.com/fsouza/go-dockerclient"
)

func main() {
	client, _ := docker.NewClientFromEnv()
	// use client
}
```

See the documentation for more details.

## Developing

All development commands can be seen in the [Makefile](Makefile).

Commited code must pass:

* [golint](https://github.com/golang/lint) (with some exceptions, see the Makefile).
* [go vet](https://golang.org/cmd/vet/)
* [gofmt](https://golang.org/cmd/gofmt)
* [go test](https://golang.org/cmd/go/#hdr-Test_packages)

Running `make test` will check all of these. If your editor does not
automatically call ``gofmt -s``, `make fmt` will format all go files in this
repository.

## Vendoring

go-dockerclient uses [dep](https://github.com/golang/dep/) for vendoring. If
you're using dep, you should be able to pick go-dockerclient releases and get
the proper dependencies.

With other vendoring tools, users might need to specify go-dockerclient's
dependencies manually.

## Using with Docker 1.9 and Go 1.4

There's a tag for using go-dockerclient with Docker 1.9 (which requires
compiling go-dockerclient with Go 1.4), the tag name is ``docker-1.9/go-1.4``.

The instructions below can be used to get a version of go-dockerclient that compiles with Go 1.4:

```
% git clone -b docker-1.9/go-1.4 https://github.com/fsouza/go-dockerclient.git $GOPATH/src/github.com/fsouza/go-dockerclient
% git clone -b v1.9.1 https://github.com/docker/docker.git $GOPATH/src/github.com/docker/docker
% go get github.com/fsouza/go-dockerclient
```
