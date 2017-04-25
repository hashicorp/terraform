# Terraform vSphere NSX Provider

This document is in place for developer documentation.  User documentation is located [HERE](https://www.terraform.io/docs/providers/nsx/) on Terraform's website.

A Terraform provider for VMware vSphere NSX.  The NSX provider is used to interact with resources supported by VMware NSX.
The provider needs to be configured with the proper credentials before it can be used.

## Introductory Documentation

Both [README.md](../../../README.md) and [BUILDING.md](../../../BUILDING.md) should be read first!

## Base API Dependency ~ [gonsx](https://github.com/sky-uk/gonsx)

This provider utilizes [gonsx](https://github.com/sky-uk/gonsx) Go Library for communicating to  VMware vSphere NSX APIs (ESXi and/or vCenter).
Because of the dependency this provider is compatible with VMware systems that are supported by gonsx. You you want to contributed additional functionality into gonsx API bindings
please feel free to send the pull requests.


## Resources Implemented
| Feature                 | Create | Read  | Update  | Delete |
|-------------------------|--------|-------|---------|--------|
| Security Tag            |   Y    |   Y   |    N    |   Y    |
| Service                 |   Y    |   Y   |    N    |   Y    |


### Limitations

This is currently a proof of concept and only has a very limited number of
supported resources.  These resources also have a very limited number
of attributes.

This section is a work in progress and additional contributions are more than welcome.



