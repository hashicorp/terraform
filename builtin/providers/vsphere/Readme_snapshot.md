This is extenstion to existing vsphere provider with added resources for snapshot
with funtionalities as:
1.Create Snapshot
2.Delete Snapshot
3.Revert Snapshot


Steps to modify existing provider
I. Copy below files to $GOPATH/src/github.com/hashicorp/terraform/builtin/providers/vsphere/
1.provider.go
2.resource_vsphere_snapshot.go
3.resource_vsphere_revert_snapshot.go
4.resource_vsphere_snapshot_test.go
5.resource_vsphere_revert_snapshot_test.go


2. To build the terraform run 
goto $GOPATH/src/github.com/hashicorp/terraform/
$ make dev

This will update the existing vsphere provider

The sample_create_snap.tf and sample_revert_snap.tf file contains the microservices of how to call the providers and resources.
We need to specify required details for resource creation in this file