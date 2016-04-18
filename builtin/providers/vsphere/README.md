# Terraform vSphere Provider Dev Docs

This document is in place for developer documentation.  User documentation is located [HERE](https://www.terraform.io/docs/providers/vsphere/) on Terraform's website.

Thank-you [@tkak](https://github.com/tkak) and [Rakuten, Inc.](https://github.com/rakutentech) for there original contribution of the source base used for this provider!

## Introductory Documentation

Both [README.md](../../../README.md) and [BUILDING.md](../../../BUILDING.md) should be read first!

## Base API Dependency ~ [govmomi](https://github.com/vmware/govmomi) 

This provider utilizes [govmomi](https://github.com/vmware/govmomi) Go Library for communicating to  VMware vSphere APIs (ESXi and/or vCenter).
Because of the dependency this provider is compatible with VMware systems that are supported by govmomi. Much thanks to the dev team that maintains govmomi, and
even more thanks to there guidance with the development of this provider.  We have had many issues answered by the govomi team!

#### vSphere CLI ~ [govc](https://github.com/vmware/govmomi/blob/master/govc/README.md)

One of the great tools that govmomi contains is [govc](https://github.com/vmware/govmomi/blob/master/govc/README.md). It is a command line tool for using the govmomi API.  Not only is it a tool to use, but also it's 
[source base](https://github.com/vmware/govmomi/blob/master/govc/) is a great resource of examples on how to exercise the API.
