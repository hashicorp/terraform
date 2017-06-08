# Terraform Brocade Virtual Traffic Manager Provider

This document is in place for developer documentation.  User documentation is located [HERE](https://www.terraform.io/docs/providers/brocadevtm/) on Terraform's website.

A Terraform provider for the Brocade vTM.  The Brocade vTM provider is used to interact with resources supported by the Brocade Virtual Traffic Manager (vTM).
The provider needs to be configured with the proper credentials before it can be used.

## Introductory Documentation

Both [README.md](../../../README.md) and [BUILDING.md](../../../BUILDING.md) should be read first!

## Base API Dependency ~ [go-brocade-vtm](https://github.com/sky-uk/go-brocade-vtm)

This provider utilizes [go-brocade-vtm](https://github.com/sky-uk/go-brocade-vtm) Go Library for communicating to the Brocade Virtual Traffic Manager REST API.
Because of the dependency this provider is compatible with Brocade systems that are supported by go-brocade-vtm. If you want to contributed additional functionality into gobrocade-vtm API bindings
please feel free to send the pull requests.


## Resources Implemented
| Feature                 | Create | Read  | Update  | Delete |
|-------------------------|--------|-------|---------|--------|
| Monitor                 |   Y    |   Y   |    N    |   Y    |
| Pools                   |   N    |   N   |    N    |   N    |
| Traffic IP              |   N    |   N   |    N    |   N    |
| Virtual Server          |   N    |   N   |    N    |   N    |


### Limitations

This is currently a proof of concept and only has a very limited number of
supported resources.  These resources also have a very limited number
of attributes.

This section is a work in progress and additional contributions are more than welcome.
