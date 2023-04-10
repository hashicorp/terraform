# Building from Source

Pre-built binaries are available for download for a variety of supported platforms through the [HashiCorp Releases website](https://releases.hashicorp.com/terraform/). 

However, if you'd like to build Terraform yourself, you can do so using the Go build toolchain and the options specified in this document.

## Prerequisites

1. Ensure you've installed the Go language version specified in [`.go-version`](https://github.com/hashicorp/terraform/blob/main/.go-version).
2. Clone this repository to a location of your choice.

## Terraform Build Options

Terraform accepts certain options passed using `ldflags` at build time which control the behavior of the resulting binary.

### Dev Version Reporting

Terraform will include a `-dev` flag when reporting its own version (ex: 1.5.0-dev) unless `version.dev` is set to `no`:

```
go build -ldflags "-w -s -X 'github.com/hashicorp/terraform/version.dev=no'" -o bin/ .
```

### Experimental Features

Experimental features of Terraform will be disabled unless `main.experimentsAllowed` is set to `yes`:

```
go build -ldflags "-w -s -X 'main.experimentsAllowed=yes'" -o bin/ .
```

## Go Options

The Terraform release process uses the Go toolchain defaults for the current operating system and processor architecture.


