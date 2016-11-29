# Scaleway's API

[![GoDoc](https://godoc.org/github.com/scaleway/scaleway-cli/pkg/api?status.svg)](https://godoc.org/github.com/scaleway/scaleway-cli/pkg/api)

This package contains facilities to play with the Scaleway API, it includes the following features:

- dedicated configuration file containing credentials to deal with the API
- caching to resolve UUIDs without contacting the API

## Links

- [API documentation](https://developer.scaleway.com)
- [Official Python SDK](https://github.com/scaleway/python-scaleway)
- Projects using this SDK
  - https://github.com/scaleway/devhub
  - https://github.com/scaleway/docker-machine-driver-scaleway
  - https://github.com/scaleway-community/scaleway-ubuntu-coreos/blob/master/overlay/usr/local/update-firewall/scw-api/cache.go
  - https://github.com/pulcy/quark
  - https://github.com/hex-sh/terraform-provider-scaleway
  - https://github.com/tscolari/bosh-scaleway-cpi
- Other **golang** clients
  - https://github.com/lalyos/onlabs
  - https://github.com/meatballhat/packer-builder-onlinelabs
  - https://github.com/nlamirault/go-scaleway
  - https://github.com/golang/build/blob/master/cmd/scaleway/scaleway.go
