#!/bin/bash

# mockgen is particularly sensitive about what mode we run it in
export GOFLAGS=""
export GO111MODULE=on

mockgen -destination mock.go github.com/hashicorp/terraform/internal/tfplugin5 ProviderClient,ProvisionerClient,Provisioner_ProvisionResourceClient,Provisioner_ProvisionResourceServer
