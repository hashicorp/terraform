Terraform SCVMM provider:-

    Microsoft SCVMM is a server application that you can use to manage a resources. SCVMM provider is created newly with some functionality of resources. 
    This provider utilizes Go Library and uses masterzen library for creating a new connection with winrm to Powershell. 
    To execute the functionality of resources windows powerShell - Virtual Machine Manager command shell is used.

Resources:

Virtual Machine:

- Create VM: It create a VM using template.And set the ID so that terraform get to known.
- Read VM: Geting the information about VM.It includes VN name,id,ram virtual disk.
- Start/Stop VM: It update the VM by performing start/stop action on it.
- Delete VM: It deletes the VM and set the id equals to null.

Virtual Disk Drive:

- Create Virtual Disk Drive : It creates virtual disk drive of specified size for that specified  VM.
- Read Virtual Disk Drive: Read the information about virtual disk.
- Delete Virtual Disk Drive: It delete the virtual disk drive

Checkpoint:

- Create checkpoint : Create checkpoint for specified VM
- Restore checkpoint : It revert to specified checkpoint
- Delete checkpoint: It deletes the checkpoint

The main.tf file contains the microservices of how to call the providers and resources. We need to specify required details for resource creation in this file.