# Introducing the Azure Resource Manager packages for Go

## How Did We Get Here?

Azure is growing rapidly, regularly adding new services and features. While rapid growth
is good for users, it is hard on SDKs. Each new service and each new feature requires someone to
learn the details and add the needed code to the SDK. As a result, the
[Azure SDK for Go](https://github.com/Azure/azure-sdk-for-go)
has lagged behind Azure. It is missing
entire services and has not kept current with features. There is simply too much change to maintain
a hand-written SDK.

For this reason, the
[Azure SDK for Go](https://github.com/Azure/azure-sdk-for-go),
with the release of the Azure Resource Manager (ARM)
packages, is transitioning to a generated-code model. Other Azure SDKs, notably the
[Azure SDK for .NET](https://github.com/Azure/azure-sdk-for-net), have successfully adopted a
generated-code strategy. Recently, Microsoft published the
[Autorest](https://github.com/Azure/autorest) tool used to create these SDKs. While the code is not
yet public (mostly because work remains), we have been adding support for Go. The ARM packages are
the first set generated using this new toolchain.

There are a couple of items to note. First, since both the tooling and the underlying support
packages are new, the code is not yet "production ready." Treat these packages as of
***beta*** quality.
That's not to say we don't believe in the code, but we want to see what others think and how well
they work in a variety of environments before settling down into an official, first release. If you
find problems or have suggestions, please submit a pull request to document what you find. However,
since the code is generated, we'll use your pull request to guide changes we make to the underlying
generator versus merging the pull request itself.

The second item of note is that, to keep the generated code clean and reliable, it depends on
another new package [go-autorest](https://github.com/Azure/go-autorest).
Though part of the SDK, we separated the code to better control versioning and maintain agility.
Since 
[go-autorest](https://github.com/Azure/go-autorest)
is hand-crafted, we will take pull requests in the same manner as for our other repositories.

We intend to rapidly improve these packages until they are "production ready."
So, try them out and give us your thoughts.

## What Have We Done?

Creating new frameworks is hard and often leads to "cliffs": The code is easy to use until some
special case or tweak arises and then, well, then you're stuck. Often times small differences in
requirements can lead to forking the code and investing a lot of time. Cliffs occur even more 
frequently in generated code. We wanted to avoid them and believe the new model does. Our initial
goals were:

* Easy-to-use out of the box. It should be "clone and go" for straight-forward use.
* Easy composition to handle the majority of complex cases.
* Easy to integrate with existing frameworks, fit nicely with channels, supporting fan-out /
fan-in set ups.

These are best shown in a series of examples, all of which are included in the
[arm/examples](https://github.com/Azure/azure-sdk-for-go/blob/master/arm/examples/)
sub-folder.

## First a Sidenote: Authentication and the Azure Resource Manager

Before using the Azure Resource Manager packages, you need to understand how it authenticates and
authorizes requests.
Unlike the earlier Azure service APIs, the Azure Resource Manager does *not* use certificates.
Instead, it relies on [OAuth2](http://oauth.net). While OAuth2 provides many advantages over
certificates, programmatic use, such as for scripts on headless servers, requires understanding and
creating one or more *Service Principals.*
There are several good blog posts, such as
[Automating Azure on your CI server using a Service Principal](http://blog.davidebbo.com/2014/12/azure-service-principal.html)
and
[Microsoft Azure REST API + OAuth 2.0](https://ahmetalpbalkan.com/blog/azure-rest-api-with-oauth2/),
that describe what this means.
For details on creating and authorizing Service Principals, see the MSDN articles
[Azure API Management REST API Authentication](https://msdn.microsoft.com/en-us/library/azure/5b13010a-d202-4af5-aabf-7ebc26800b3d)
and
[Create a new Azure Service Principal using the Azure portal](https://azure.microsoft.com/en-us/documentation/articles/resource-group-create-service-principal-portal/).
Dushyant Gill, a Senior Program Manager for Azure Active Directory, has written an extensive blog
post,
[Developer's Guide to Auth with Azure Resource Manager API](http://www.dushyantgill.com/blog/2015/05/23/developers-guide-to-auth-with-azure-resource-manager-api/),
that is also quite helpful.

## A Simple Example: Checking availability of name within Azure Storage

Each ARM provider, such as
[Azure Storage](http://azure.microsoft.com/en-us/documentation/services/storage/)
or
[Azure Compute](https://azure.microsoft.com/en-us/documentation/services/virtual-machines/),
has its own package. Start by importing
the packages for the providers you need. Next, most packages divide their APIs across multiple
clients to avoid name collision and improve usability. For example, the
[Azure Storage](http://azure.microsoft.com/en-us/documentation/services/storage/)
package has
two clients:
[storage.StorageAccountsClient](https://godoc.org/github.com/Azure/azure-sdk-for-go/arm/storage#StorageAccountsClient)
and
[storage.UsageOperationsClient](https://godoc.org/github.com/Azure/azure-sdk-for-go/arm/storage#UsageOperationsClient).
To check if a name is available, use the
[storage.StorageAccountsClient](https://godoc.org/github.com/Azure/azure-sdk-for-go/arm/storage#StorageAccountsClient):

```go
package main

import(
  "fmt"
  "log"

  "github.com/Azure/azure-sdk-for-go/arm/examples/helpers"
  "github.com/Azure/azure-sdk-for-go/arm/storage"
  "github.com/Azure/go-autorest/autorest"
  "github.com/Azure/go-autorest/autorest/azure"
  "github.com/Azure/go-autorest/autorest/to"
)

func checkName(name string) {
  c, err := helpers.LoadCredentials()
  if err != nil {
    log.Fatalf("Error: %v", err)
  }

  ac := storage.NewAccountsClient(c["subscriptionID"])

  spt, err := helpers.NewServicePrincipalTokenFromCredentials(c, azure.PublicCloud.ResourceManagerEndpoint)
  if err != nil {
    log.Fatalf("Error: %v", err)
  }
  ac.Authorizer = spt

  ac.Sender = autorest.CreateSender(
    autorest.WithLogging(log.New(os.Stdout, "sdk-example: ", log.LstdFlags)))

  cna, err := ac.CheckNameAvailability(
    storage.AccountCheckNameAvailabilityParameters{
      Name: to.StringPtr(name),
      Type: to.StringPtr("Microsoft.Storage/storageAccounts")})

  if err != nil {
    log.Fatalf("Error: %v", err)
  } else {
    if to.Bool(cna.NameAvailable) {
      fmt.Printf("The name '%s' is available\n", name)
    } else {
      fmt.Printf("The name '%s' is unavailable because %s\n", name, cna.Message)
    }
  }
}
```

Each ARM client composes with [autorest.Client](https://godoc.org/github.com/Azure/go-autorest/autorest#Client).
[autorest.Client](https://godoc.org/github.com/Azure/go-autorest/autorest#Client)
enables altering the behavior of the API calls by leveraging the decorator pattern of
[go-autorest](https://github.com/Azure/go-autorest). For example, in the code above, the
[azure.ServicePrincipalToken](https://godoc.org/github.com/Azure/go-autorest/autorest/azure#ServicePrincipalToken)
includes a
[WithAuthorization](https://godoc.org/github.com/Azure/go-autorest/autorest#Client.WithAuthorization)
[autorest.PrepareDecorator](https://godoc.org/github.com/Azure/go-autorest/autorest#PrepareDecorator)
that applies the OAuth2 authorization token to the request. It will, as needed, refresh the token
using the supplied credentials.

Providing a decorated
[autorest.Sender](https://godoc.org/github.com/Azure/go-autorest/autorest#Sender) or populating
the [autorest.Client](https://godoc.org/github.com/Azure/go-autorest/autorest#Client)
with a custom
[autorest.PrepareDecorator](https://godoc.org/github.com/Azure/go-autorest/autorest#PrepareDecorator)
or
[autorest.RespondDecorator](https://godoc.org/github.com/Azure/go-autorest/autorest#RespondDecorator)
enables more control. See the included example file
[check.go](https://github.com/Azure/azure-sdk-for-go/blob/master/arm/examples/check.go)
for more details. Through these you can modify the outgoing request, inspect the incoming response,
or even go so far as to provide a
[circuit breaker](https://msdn.microsoft.com/en-us/library/dn589784.aspx)
to protect your service from unexpected latencies.

Lastly, all Azure ARM API calls return an instance of the
[autorest.Error](https://godoc.org/github.com/Azure/go-autorest/autorest#Error) interface.
Not only does the interface give anonymous access to the original
[error](http://golang.org/ref/spec#Errors),
but provides the package type (e.g.,
[storage.StorageAccountsClient](https://godoc.org/github.com/Azure/azure-sdk-for-go/arm/storage#StorageAccountsClient)),
the failing method (e.g.,
[CheckNameAvailability](https://godoc.org/github.com/Azure/azure-sdk-for-go/arm/storage#StorageAccountsClient.CheckNameAvailability)),
and a detailed error message.

## Something a Bit More Complex: Creating a new Azure Storage account

Redundancy, both local and across regions, and service load affect service responsiveness. Some
API calls will return before having completed the request. An Azure ARM API call indicates the
request is incomplete (versus the request failed for some reason) by returning HTTP status code
'202 Accepted.' The
[autorest.Client](https://godoc.org/github.com/Azure/go-autorest/autorest#Client)
composed into
all of the Azure ARM clients, provides support for basic request polling. The default is to
poll until a specified duration has passed (with polling frequency determined by the
HTTP [Retry-After](http://www.w3.org/Protocols/rfc2616/rfc2616-sec14.html#sec14.37)
header in the response). By changing the
[autorest.Client](https://godoc.org/github.com/Azure/go-autorest/autorest#Client)
settings, you can poll for a fixed number of attempts or elect to not poll at all.

Whether you elect to poll or not, all Azure ARM client responses compose with an instance of
[autorest.Response](https://godoc.org/github.com/Azure/go-autorest/autorest#Response).
At present,
[autorest.Response](https://godoc.org/github.com/Azure/go-autorest/autorest#Response)
only composes over the standard
[http.Response](https://golang.org/pkg/net/http/#Response)
object (that may change as we implement more features). When your code receives an error from an
Azure ARM API call, you may find it useful to inspect the HTTP status code contained in the returned
[autorest.Response](https://godoc.org/github.com/Azure/go-autorest/autorest#Response).
If, for example, it is an HTTP 202, then you can use the
[GetPollingLocation](https://godoc.org/github.com/Azure/go-autorest/autorest#Response.GetPollingLocation)
response method to extract the URL at which to continue polling. Similarly, the
[GetPollingDelay](https://godoc.org/github.com/Azure/go-autorest/autorest#Response.GetPollingDelay)
response method returns, as a
[time.Duration](http://golang.org/pkg/time/#Duration),
the service suggested minimum polling delay.

Creating a new Azure storage account is a straight-forward way to see these concepts.

```go

package main

import(
  "fmt"

  "github.com/Azure/azure-sdk-for-go/arm/examples/helpers"
  "github.com/Azure/azure-sdk-for-go/arm/storage"
  "github.com/Azure/go-autorest/autorest"
  "github.com/Azure/go-autorest/autorest/azure"
  "github.com/Azure/go-autorest/autorest/to"
)

func create_account(resourceGroup, name string) {
  c, err := helpers.LoadCredentials()
  if err != nil {
    log.Fatalf("Error: %v", err)
  }

  ac := storage.NewAccountsClient(c["subscriptionID"])

  spt, err := helpers.NewServicePrincipalTokenFromCredentials(c, azure.AzureResourceManagerScope)
  if err != nil {
    log.Fatalf("Error: %v", err)
  }
  ac.Authorizer = spt
  ac.PollingMode = autorest.PollUntilAttempts
  ac.PollingAttempts = 5

  cp := storage.AccountCreateParameters{}
  cp.Location = to.StringPtr("westus")
  cp.Properties = &storage.AccountPropertiesCreateParameters{AccountType: storage.StandardLRS}

  sa, err := ac.Create(resourceGroup, name, cp)
  if err != nil {
    if sa.Response.StatusCode != http.StatusAccepted {
      fmt.Printf("Creation of %s.%s failed with err -- %v\n", resourceGroup, name, err)
      return
    } else {
      fmt.Printf("Create initiated for %s.%s -- poll %s to check status\n",
        resourceGroup,
        name,
        sa.GetPollingLocation())
      return
    }
  }

  fmt.Printf("Successfully created %s.%s\n\n", resourceGroup, name)
}
```

The above example modifies the
[autorest.Client](https://godoc.org/github.com/Azure/go-autorest/autorest#Client)
portion of the
[storage.StorageAccountsClient](https://godoc.org/github.com/Azure/azure-sdk-for-go/arm/storage#StorageAccountsClient)
to poll for a fixed number of attempts versus polling for a set duration (which is the default).
If an error occurs creating the storage account, the code inspects the HTTP status code and
prints the URL the
[Azure Storage](http://azure.microsoft.com/en-us/documentation/services/storage/)
service returned for polling.
More details, including deleting the created account, are in the example code file
[create.go](https://github.com/Azure/azure-sdk-for-go/blob/master/arm/examples/create.go).

## Making Asynchronous Requests

One of Go's many strong points is how natural it makes sending and managing asynchronous requests
by means of goroutines. We wanted the ARM packages to fit naturally in the variety of asynchronous
patterns used in Go code, but also be straight-forward for simple use cases. We accomplished both
by adopting a pattern for all APIs. Each package API includes (at least) four methods
(more if the API returns a paged result set). For example, for an API call named `Foo` the package
defines:

- `FooPreparer`: This method accepts the arguments for the API and returns a prepared
`http.Request`.
- `FooSender`: This method sends the prepared `http.Request`. It handles the possible status codes
and will, unless the disabled in the [autorest.Client](https://godoc.org/github.com/Azure/go-
autorest/autorest#Client), handling polling.
- `FooResponder`: This method accepts and handles the `http.Response` returned by the sender
and unmarshals the JSON, if any, into the result.
- `Foo`: This method accepts the arguments for the API and returns the result. It is a wrapper
around the `FooPreparer`, `FooSender`, and `FooResponder`.

By using the preparer, sender, and responder methods, package users can spread request and
response handling across goroutines as needed. Further, adding a cancel channel to the
`http.Response` (most easily through a
[PrepareDecorator](https://godoc.org/github.com/Azure/go-autorest/autorest#PrepareDecorator)),
enables canceling sent requests (see the documentation on
[http.Request](https://golang.org/pkg/net/http/#Request)) for details.

## Paged Result Sets

Some API calls return partial results. Typically, when they do, the result structure will include
a `Value` array and a `NextLink` URL. The `NextLink` URL is used to retrieve the next page or
block of results.

The packages add two methods to make working with and retrieving paged results natural. First,
on paged result structures, the packages include a preparer method that returns an `http.Request`
for the next set of results. For a result set returned in a structure named `FooResults`, the
package will include a method named `FooResultsPreparer`. If the `NextLink` is `nil` or empty, the
method returns `nil`.

The corresponding API (which typically includes "List" in the name) has a method to ease retrieving
the next result set given a result set. For example, for an API named `FooList`, the package will
include `FooListNextResults` that accepts the results of the last call and returns the next set.

## Summing Up

The new Azure Resource Manager packages for the Azure SDK for Go are a big step toward keeping the
SDK current with Azure's rapid growth.
As mentioned, we intend to rapidly stabilize these packages for production use.
We'll also add more examples, including some highlighting the
[Azure Resource Manager Templates](https://msdn.microsoft.com/en-us/library/azure/dn790568.aspx)
and the other providers.

So, give the packages a try, explore the various ARM providers, and let us know what you think. 

We look forward to hearing from you!


## Installing the Azure Resource Manager Packages

Install the packages you require as you would any other Go package:

```bash
go get github.com/Azure/azure-sdk-for-go/arm/authorization
go get github.com/Azure/azure-sdk-for-go/arm/compute
go get github.com/Azure/azure-sdk-for-go/arm/dns
go get github.com/Azure/azure-sdk-for-go/arm/features
go get github.com/Azure/azure-sdk-for-go/arm/logic
go get github.com/Azure/azure-sdk-for-go/arm/network
go get github.com/Azure/azure-sdk-for-go/arm/redis
go get github.com/Azure/azure-sdk-for-go/arm/resources
go get github.com/Azure/azure-sdk-for-go/arm/scheduler
go get github.com/Azure/azure-sdk-for-go/arm/search
go get github.com/Azure/azure-sdk-for-go/arm/storage
go get github.com/Azure/azure-sdk-for-go/arm/subscriptions
go get github.com/Azure/azure-sdk-for-go/arm/web
```

## License

See the Azure SDK for Go LICENSE file.
