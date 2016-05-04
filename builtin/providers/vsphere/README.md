# Terraform vSphere Provider Dev Docs

This document is in place for developer documentation.  User documentation is located [HERE](https://www.terraform.io/docs/providers/vsphere/) on Terraform's website.

Thank-you [@tkak](https://github.com/tkak) and [Rakuten, Inc.](https://github.com/rakutentech) for their original contribution of the source base used for this provider!

## Introductory Documentation

Both [README.md](../../../README.md) and [BUILDING.md](../../../BUILDING.md) should be read first!

## Base API Dependency ~ [govmomi](https://github.com/vmware/govmomi) 

This provider utilizes [govmomi](https://github.com/vmware/govmomi) Go Library for communicating to  VMware vSphere APIs (ESXi and/or vCenter).
Because of the dependency this provider is compatible with VMware systems that are supported by govmomi. Much thanks to the dev team that maintains govmomi, and
even more thanks to their guidance with the development of this provider.  We have had many issues answered by the govmomi team!

#### vSphere CLI ~ [govc](https://github.com/vmware/govmomi/blob/master/govc/README.md)

One of the great tools that govmomi contains is [govc](https://github.com/vmware/govmomi/blob/master/govc/README.md). It is a command line tool for using the govmomi API.  Not only is it a tool to use, but also it's 
[source base](https://github.com/vmware/govmomi/blob/master/govc/) is a great resource of examples on how to exercise the API.

## Required privileges for running Terraform as non-administrative user
Most of the organizations are concerned about administrative privileges. In order to use Terraform provider as non priviledged user, we can define a new Role within a vCenter and assign it appropriate privileges:
Navigate to Administration -> Access Control -> Roles
Click on "+" icon (Create role action), give it appropraite name and select following privileges:
 * Datastore
   - Allocate space
   - Browse datastore
   - Low level file operations
   - Remove file
   - Update virtual machine files
   - Update virtual machine metadata
 
 * Folder (all)
   - Create folder
   - Delete folder
   - Move folder
   - Rename folder
 
 * Network
   - Assign network

 * Resource
   - Apply recommendation
   - Assign virtual machine to resource pool

 * Virtual Machine
   - Configuration (all) - for now
   - Guest Operations (all) - for now
   - Interaction (all)
   - Inventory (all)
   - Provisioning (all)

These settings were tested with [vSphere 6.0](https://pubs.vmware.com/vsphere-60/index.jsp?topic=%2Fcom.vmware.vsphere.security.doc%2FGUID-18071E9A-EED1-4968-8D51-E0B4F526FDA3.html) and [vSphere 5.5](https://pubs.vmware.com/vsphere-55/index.jsp?topic=%2Fcom.vmware.vsphere.security.doc%2FGUID-18071E9A-EED1-4968-8D51-E0B4F526FDA3.html). For additional information on roles and permissions, please refer to official VMware documentation.

This section is a work in progress and additional contributions are more than welcome.
 
