---
layout: "vsphere"
page_title: "VMware vSphere: vsphere_user_security_setup"
sidebar_current: "docs-vsphere-resource-user-security-setup"
description: |-
  Setup up a vSphere user to use the vSphere Terraform provider.
-----------------------------------------------------------------------------------------------------------------------------------------------------

## Required privileges for running Terraform as non-administrative user
Most of the organizations are concerned about administrative privileges. In order to use Terraform provider as non administrative user, we can define a new Role within a vCenter and assign it appropriate privileges.

In the vCenter UI navigate to the following:

Navigate to Administration -> Access Control -> Roles

Click on "+" icon (Create role action), give it appropriate name and select following privileges:
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
