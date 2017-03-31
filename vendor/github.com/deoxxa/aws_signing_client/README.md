# AWS Signing Client

This package provides simple `http.Client` creation that wraps all outgoing HTTP requests with Amazon AWS signatures using the [AWS SDK for Go](https://github.com/aws/aws-sdk-go).

## Requirements

In order to use signing graciously provided by [@nicolai86](https://github.com/nikolai86) in the AWS SDK for Go, you must be using a version that has been updated since the merge of [pull request #735](https://github.com/aws/aws-sdk-go/pull/735) for the SDK (tagged release v1.2.0).

## Usage

You can provide your own `*http.Client` to have any existing fields persist or your `RoundTripper` wrapped:

```go
import (
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/signer/v4"
	"github.com/sha1sum/aws_signing_client"
)

var credentials *credentials.Credentials
// ... set credentials ...
var signer = v4.NewSigner(credentials)

var myClient *http.Client
// ...

// *v4.Signer, *http.Client, AWS service abbreviation, AWS region
var awsClient = aws_signing_client.New(signer, myClient, "es", "us-east-1")
```

... or you can simply have the default client with default client and transport created for you:

```go
import (
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/signer/v4"
	"github.com/sha1sum/aws_signing_client"
)

var credentials *credentials.Credentials
// ... set credentials ...
var signer = v4.NewSigner(credentials)

// *v4.Signer, *http.Client, AWS service abbreviation, AWS region
var awsClient = aws_signing_client.New(signer, nil, "es", "us-east-1")
```