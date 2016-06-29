# Microsoft Azure SDK for Go

This project provides various Go packages to perform operations
on Microsoft Azure REST APIs.

[![GoDoc](https://godoc.org/github.com/Azure/azure-sdk-for-go?status.svg)](https://godoc.org/github.com/Azure/azure-sdk-for-go) [![Build Status](https://travis-ci.org/Azure/azure-sdk-for-go.svg?branch=master)](https://travis-ci.org/Azure/azure-sdk-for-go)

See list of implemented API clients [here](http://godoc.org/github.com/Azure/azure-sdk-for-go).

> **NOTE:** This repository is under heavy ongoing development and
is likely to break over time. We currently do not have any releases
yet. If you are planning to use the repository, please consider vendoring
the packages in your project and update them when a stable tag is out.

# Installation

    go get -d github.com/Azure/azure-sdk-for-go/management

# Usage

Read Godoc of the repository at: http://godoc.org/github.com/Azure/azure-sdk-for-go/

The client currently supports authentication to the Service Management
API with certificates or Azure `.publishSettings` file. You can 
download the `.publishSettings` file for your subscriptions
[here](https://manage.windowsazure.com/publishsettings).

### Example: Creating a Linux Virtual Machine

```go
package main

import (
	"encoding/base64"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/management"
	"github.com/Azure/azure-sdk-for-go/management/hostedservice"
	"github.com/Azure/azure-sdk-for-go/management/virtualmachine"
	"github.com/Azure/azure-sdk-for-go/management/vmutils"
)

func main() {
	dnsName := "test-vm-from-go"
	storageAccount := "mystorageaccount"
	location := "West US"
	vmSize := "Small"
	vmImage := "b39f27a8b8c64d52b05eac6a62ebad85__Ubuntu-14_04-LTS-amd64-server-20140724-en-us-30GB"
	userName := "testuser"
	userPassword := "Test123"

	client, err := management.ClientFromPublishSettingsFile("path/to/downloaded.publishsettings", "")
	if err != nil {
		panic(err)
	}

	// create hosted service
	if err := hostedservice.NewClient(client).CreateHostedService(hostedservice.CreateHostedServiceParameters{
		ServiceName: dnsName,
		Location:    location,
		Label:       base64.StdEncoding.EncodeToString([]byte(dnsName))}); err != nil {
		panic(err)
	}

	// create virtual machine
	role := vmutils.NewVMConfiguration(dnsName, vmSize)
	vmutils.ConfigureDeploymentFromPlatformImage(
		&role,
		vmImage,
		fmt.Sprintf("http://%s.blob.core.windows.net/sdktest/%s.vhd", storageAccount, dnsName),
		"")
	vmutils.ConfigureForLinux(&role, dnsName, userName, userPassword)
	vmutils.ConfigureWithPublicSSH(&role)

	operationID, err := virtualmachine.NewClient(client).
		CreateDeployment(role, dnsName, virtualmachine.CreateDeploymentOptions{})
	if err != nil {
		panic(err)
	}
	if err := client.WaitForOperation(operationID, nil); err != nil {
		panic(err)
	}
}
```

# License

This project is published under [Apache 2.0 License](LICENSE).

-----
This project has adopted the [Microsoft Open Source Code of Conduct](https://opensource.microsoft.com/codeofconduct/). For more information see the [Code of Conduct FAQ](https://opensource.microsoft.com/codeofconduct/faq/) or contact [opencode@microsoft.com](mailto:opencode@microsoft.com) with any additional questions or comments.
