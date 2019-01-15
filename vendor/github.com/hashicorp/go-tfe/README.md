Terraform Enterprise Go Client
==============================

[![Build Status](https://travis-ci.org/hashicorp/go-tfe.svg?branch=master)](https://travis-ci.org/hashicorp/go-tfe)
[![GitHub license](https://img.shields.io/github/license/hashicorp/go-tfe.svg)](https://github.com/hashicorp/go-tfe/blob/master/LICENSE)
[![GoDoc](https://godoc.org/github.com/hashicorp/go-tfe?status.svg)](https://godoc.org/github.com/hashicorp/go-tfe)
[![Go Report Card](https://goreportcard.com/badge/github.com/hashicorp/go-tfe)](https://goreportcard.com/report/github.com/hashicorp/go-tfe)
[![GitHub issues](https://img.shields.io/github/issues/hashicorp/go-tfe.svg)](https://github.com/hashicorp/go-tfe/issues)

This is an API client for [Terraform Enterprise](https://www.hashicorp.com/products/terraform).

## NOTE

The Terraform Enterprise API endpoints are in beta and are subject to change!
So that means this API client is also in beta and is also subject to change. We
will indicate any breaking changes by releasing new versions. Until the release
of v1.0, any minor version changes will indicate possible breaking changes. Patch
version changes will be used for both bugfixes and non-breaking changes.

## Coverage

Currently the following endpoints are supported:

- [x] [Accounts](https://www.terraform.io/docs/enterprise/api/account.html)
- [x] [Configuration Versions](https://www.terraform.io/docs/enterprise/api/configuration-versions.html)
- [x] [OAuth Clients](https://www.terraform.io/docs/enterprise/api/oauth-clients.html)
- [x] [OAuth Tokens](https://www.terraform.io/docs/enterprise/api/oauth-tokens.html)
- [x] [Organizations](https://www.terraform.io/docs/enterprise/api/organizations.html)
- [x] [Organization Tokens](https://www.terraform.io/docs/enterprise/api/organization-tokens.html)
- [x] [Policies](https://www.terraform.io/docs/enterprise/api/policies.html)
- [x] [Policy Sets](https://www.terraform.io/docs/enterprise/api/policy-sets.html)
- [x] [Policy Checks](https://www.terraform.io/docs/enterprise/api/policy-checks.html)
- [ ] [Registry Modules](https://www.terraform.io/docs/enterprise/api/modules.html)
- [x] [Runs](https://www.terraform.io/docs/enterprise/api/run.html)
- [x] [SSH Keys](https://www.terraform.io/docs/enterprise/api/ssh-keys.html)
- [x] [State Versions](https://www.terraform.io/docs/enterprise/api/state-versions.html)
- [x] [Team Access](https://www.terraform.io/docs/enterprise/api/team-access.html)
- [x] [Team Memberships](https://www.terraform.io/docs/enterprise/api/team-members.html)
- [x] [Team Tokens](https://www.terraform.io/docs/enterprise/api/team-tokens.html)
- [x] [Teams](https://www.terraform.io/docs/enterprise/api/teams.html)
- [x] [Variables](https://www.terraform.io/docs/enterprise/api/variables.html)
- [x] [Workspaces](https://www.terraform.io/docs/enterprise/api/workspaces.html)
- [ ] [Admin](https://www.terraform.io/docs/enterprise/api/admin/index.html)

## Installation

Installation can be done with a normal `go get`:

```
go get -u github.com/hashicorp/go-tfe
```

## Documentation

For complete usage of the API client, see the full [package docs](https://godoc.org/github.com/hashicorp/go-tfe).

## Usage

```go
import tfe "github.com/hashicorp/go-tfe"
```

Construct a new TFE client, then use the various endpoints on the client to
access different parts of the Terraform Enterprise API. For example, to list
all organizations:

```go
config := &tfe.Config{
	Token: "insert-your-token-here",
}

client, err := tfe.NewClient(config)
if err != nil {
	log.Fatal(err)
}

orgs, err := client.Organizations.List(context.Background(), OrganizationListOptions{})
if err != nil {
	log.Fatal(err)
}
```

## Examples

The [examples](https://github.com/hashicorp/go-tfe/tree/master/examples) directory
contains a couple of examples. One of which is listed here as well:

```go
package main

import (
	"log"

	tfe "github.com/hashicorp/go-tfe"
)

func main() {
	config := &tfe.Config{
		Token: "insert-your-token-here",
	}

	client, err := tfe.NewClient(config)
	if err != nil {
		log.Fatal(err)
	}

	// Create a context
	ctx := context.Background()

	// Create a new organization
	options := tfe.OrganizationCreateOptions{
		Name:  tfe.String("example"),
		Email: tfe.String("info@example.com"),
	}

	org, err := client.Organizations.Create(ctx, options)
	if err != nil {
		log.Fatal(err)
	}

	// Delete an organization
	err = client.Organizations.Delete(ctx, org.Name)
	if err != nil {
		log.Fatal(err)
	}
}
```

## Running tests

Tests are run against an actual backend so they require a valid backend address
and token. In addition it also needs a Github token for running the OAuth Client
tests:

```sh
$ export TFE_ADDRESS=https://tfe.local
$ export TFE_TOKEN=xxxxxxxxxxxxxxxxxxx
$ export GITHUB_TOKEN=xxxxxxxxxxxxxxxx
```

In order for the tests relating to queuing and capacity to pass, FRQ should be
enabled with a limit of 2 concurrent runs per organization.

As running the tests takes about ~10 minutes, make sure to add a timeout to your
command (as the default timeout is 10m):

```sh
$ go test ./... -timeout=15m
```

## Issues and Contributing

If you find an issue with this package, please report an issue. If you'd like,
we welcome any contributions. Fork this repository and submit a pull request.
