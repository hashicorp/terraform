### Autoscale a Linux VM Scale Set ###

The following template deploys a Linux VM Scale Set integrated with Azure autoscale.

The template deploys a Linux VMSS with a desired count of VMs in the scale set. Once the VM Scale Sets is deployed, user can deploy an application inside each of the VMs (either by directly logging into the VMs or via a [`remote-exec` provisioner](https://www.terraform.io/docs/provisioners/remote-exec.html)).

The Autoscale rules are configured as follows:
- sample for CPU (\\Processor\\PercentProcessorTime) in each VM every 1 Minute
- if the Percent Processor Time is greater than 50% for 5 Minutes, then the scale out action (add more VM instances, one at a time) is triggered
- once the scale out action is completed, the cool down period is 1 Minute