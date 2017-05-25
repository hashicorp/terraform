Terraform SCVMM provider:

    Microsoft SCVMM is a server application which can be used to manage resources. The Terraform SCVMM provider uses the Go Library and the masterzen library for creating a new connection with winrm and Powershell of SCVMM server. 
    To use the resources, Windows PowerShell and Virtual Machine Manager commands are used.

Resources:

Virtual Machine:

- Create VM: Creates a VM using a template and sets the ID for Terraform
- Read VM: Gets information about a VM. This includes the VM name, ID, and RAM virtual disk
- Start/Stop VM: Starts or stops the VM
- Delete VM: Deletes the VM and set the ID to null

Virtual Disk Drive:

- Create Virtual Disk Drive : Creates a virtual disk drive of the specified size for the specific VM
- Read Virtual Disk Drive: Reads the information about the virtual disk
- Delete Virtual Disk Drive: Deletes the virtual disk drive

Checkpoint:

- Create checkpoint : Creates a checkpoint for the specified VM
- Restore checkpoint : Reverts to a specified checkpoint
- Delete checkpoint: Deletes a checkpoint

The main.tf file contains micro-services for calling providers and resources. We need to specify the required details for resource creation in this file.
