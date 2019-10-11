# aws-sdk-go-base

An opinionated [AWS Go SDK](https://github.com/aws/aws-sdk-go) library for consistent authentication configuration between projects and additional helper functions. This library was originally started in [HashiCorp Terraform](https://github.com/hashicorp/terraform), migrated with the [Terraform AWS Provider](https://github.com/terraform-providers/terraform-provider-aws) during the Terraform 0.10 Core and Provider split, and now is offered as a separate library to allow easier dependency management in the Terraform ecosystem.

**NOTE:** This library is not currently designed or intended for usage outside the [Terraform S3 Backend](https://www.terraform.io/docs/backends/types/s3.html) and the [Terraform AWS Provider](https://www.terraform.io/docs/providers/aws/index.html).

## Requirements

- [Go](https://golang.org/doc/install) 1.12

## Development

Testing this project can be done through Go standard library functionality or if [Make](https://www.gnu.org/software/make/) is available:

```sh
$ go test -v ./...
# Optionally if Make is available; both run the same testing
$ make test
```

Code quality assurance uses [golangci-lint](https://github.com/golangci/golangci-lint):

```sh
$ golangci-lint run ./...
# Optionally if Make is available; both run the same linting
$ make lint
```
