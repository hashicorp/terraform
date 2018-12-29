# triton-go

`go-triton` is an idiomatic library exposing a client SDK for Go applications using the Joyent Triton API. 

## Usage

Triton uses [HTTP Signature][4] to sign the Date header in each HTTP request made to the Triton API. Currently, requests can be signed using either a private key file loaded from disk (using an [`authentication.PrivateKeySigner`][5]), or using a key stored with the local SSH Agent (using an [`SSHAgentSigner`][6].

To construct a Signer, use the `New*` range of methods in the `authentication` package. In the case of `authentication.NewSSHAgentSigner`, the parameters are the fingerprint of the key with which to sign, and the account name (normally stored in the `SDC_ACCOUNT` environment variable). For example:

```
const fingerprint := "a4:c6:f3:75:80:27:e0:03:a9:98:79:ef:c5:0a:06:11"
sshKeySigner, err := authentication.NewSSHAgentSigner(fingerprint, "AccountName")
if err != nil {
	log.Fatalf("NewSSHAgentSigner: %s", err)
}
```

An appropriate key fingerprint can be generated using `ssh-keygen`:

```
ssh-keygen -Emd5 -lf ~/.ssh/id_rsa.pub | cut -d " " -f 2 | sed 's/MD5://'
```

To construct a Client, use the `NewClient` function, passing in the endpoint, account name and constructed signer:

```go
client, err := triton.NewClient("https://us-sw-1.api.joyent.com/", "AccountName",	sshKeySigner)
if err != nil {
	log.Fatalf("NewClient: %s", err)
}
```

Having constructed a `triton.Client`, use the methods available to access functionality by functional grouping. For example, for access to operations on SSH keys, use the `Keys()` method to obtain a client which has access to the `CreateKey`, `ListKeys` and `DeleteKey` operations. For access to operations on Machines, use the `Machines()` method to obtain a client which has access to the `RenameMachine`, `GetMachineMetadata`, `GetMachineTag`, and other operations.

Operation methods take their formal parameters via a struct named `OperationInput` - for example when creating an SSH key, the `CreateKeyInput` struct is used with the `func CreateKey(*CreateKeyInput) (*Key, error)` method. This allows specification of named parameters:

```
client := state.Client().Keys()

key, err := client.CreateKey(&CreateKeyInput{
	Name: "tempKey",
	Key:  "ssh-rsa .....",
})
if err != nil {
	panic(err)
}

// Key contains the return value.
```

## Error Handling

If an error is returned by the HTTP API, the `error` returned from the function will contain an instance of `triton.TritonError` in the chain. Error wrapping is performed using the [errwrap][7] library from HashiCorp.

## Completeness

The following list is updated as new functionality is added. The complete list of operations is taken from the [CloudAPI documentation](https://apidocs.joyent.com/cloudapi).

- Accounts
	- [x] GetAccount
	- [x] UpdateAccount
- Keys
	- [x] ListKeys
	- [x] GetKey
	- [x] CreateKey
	- [x] DeleteKey
- Users
	- [ ] ListUsers
	- [ ] GetUser
	- [ ] CreateUser
	- [ ] UpdateUser
	- [ ] ChangeUserPassword
	- [ ] DeleteUser
- Roles
	- [x] ListRoles
	- [x] GetRole
	- [x] CreateRole
	- [x] UpdateRole
	- [x] DeleteRole
- Role Tags
	- [ ] SetRoleTags
- Policies
	- [ ] ListPolicies
	- [ ] GetPolicy
	- [ ] CreatePolicy
	- [ ] UpdatePolicy
	- [ ] DeletePolicy
- User SSH Keys
	- [x] ListUserKeys
	- [x] GetUserKey
	- [x] CreateUserKey
	- [x] DeleteUserKey
- Config
	- [x] GetConfig
	- [x] UpdateConfig
- Datacenters
	- [x] ListDatacenters
	- [x] GetDatacenter
- Services
	- [x] ListServices
- Images
	- [x] ListImages
	- [x] GetImage
	- [x] DeleteImage
	- [x] ExportImage
	- [x] CreateImageFromMachine
	- [x] UpdateImage
- Packages
	- [x] ListPackages
	- [x] GetPackage
- Instances
	- [ ] ListMachines
	- [x] GetMachine
	- [x] CreateMachine
	- [ ] StopMachine
	- [ ] StartMachine
	- [ ] RebootMachine
	- [x] ResizeMachine
	- [x] RenameMachine
	- [x] EnableMachineFirewall
	- [x] DisableMachineFirewall
	- [ ] CreateMachineSnapshot
	- [ ] StartMachineFromSnapshot
	- [ ] ListMachineSnapshots
	- [ ] GetMachineSnapshot
	- [ ] DeleteMachineSnapshot
	- [x] UpdateMachineMetadata
	- [ ] ListMachineMetadata
	- [ ] GetMachineMetadata
	- [ ] DeleteMachineMetadata
	- [ ] DeleteAllMachineMetadata
	- [x] AddMachineTags
	- [x] ReplaceMachineTags
	- [x] ListMachineTags
	- [x] GetMachineTag
	- [x] DeleteMachineTag
	- [x] DeleteMachineTags
	- [x] DeleteMachine
	- [ ] MachineAudit
- Analytics
	- [ ] DescribeAnalytics
	- [ ] ListInstrumentations
	- [ ] GetInstrumentation
	- [ ] GetInstrumentationValue
	- [ ] GetInstrumentationHeatmap
	- [ ] GetInstrumentationHeatmapDetails
	- [ ] CreateInstrumentation
	- [ ] DeleteInstrumentation
- Firewall Rules
	- [x] ListFirewallRules
	- [x] GetFirewallRule
	- [x] CreateFirewallRule
	- [x] UpdateFirewallRule
	- [x] EnableFirewallRule
	- [x] DisableFirewallRule
	- [x] DeleteFirewallRule
	- [ ] ListMachineFirewallRules
	- [x] ListFirewallRuleMachines
- Fabrics
	- [x] ListFabricVLANs
	- [x] CreateFabricVLAN
	- [x] GetFabricVLAN
	- [x] UpdateFabricVLAN
	- [x] DeleteFabricVLAN
	- [x] ListFabricNetworks
	- [x] CreateFabricNetwork
	- [x] GetFabricNetwork
	- [x] DeleteFabricNetwork
- Networks
	- [x] ListNetworks
	- [x] GetNetwork
- Nics
	- [ ] ListNics
	- [ ] GetNic
	- [x] AddNic
	- [x] RemoveNic

## Running Acceptance Tests

Acceptance Tests run directly against the Triton API, so you will need either a local installation or Triton or an account with Joyent in order to run them. The tests create real resources (and thus cost real money!)

In order to run acceptance tests, the following environment variables must be set:

- `TRITON_TEST` - must be set to any value in order to indicate desire to create resources
- `SDC_URL` - the base endpoint for the Triton API
- `SDC_ACCOUNT` - the account name for the Triton API
- `SDC_KEY_ID` - the fingerprint of the SSH key identifying the key

Additionally, you may set `SDC_KEY_MATERIAL` to the contents of an unencrypted private key. If this is set, the PrivateKeySigner (see above) will be used - if not the SSHAgentSigner will be used.

### Example Run

The verbose output has been removed for brevity here.

```
$ HTTP_PROXY=http://localhost:8888 \
	TRITON_TEST=1 \
	SDC_URL=https://us-sw-1.api.joyent.com \
	SDC_ACCOUNT=AccountName \
	SDC_KEY_ID=a4:c6:f3:75:80:27:e0:03:a9:98:79:ef:c5:0a:06:11 \
	go test -v -run "TestAccKey"
=== RUN   TestAccKey_Create
--- PASS: TestAccKey_Create (12.46s)
=== RUN   TestAccKey_Get
--- PASS: TestAccKey_Get (4.30s)
=== RUN   TestAccKey_Delete
--- PASS: TestAccKey_Delete (15.08s)
PASS
ok  	github.com/jen20/triton-go	31.861s
```

[4]: https://github.com/joyent/node-http-signature/blob/master/http_signing.md 
[5]: https://godoc.org/github.com/joyent/go-triton/authentication
[6]: https://godoc.org/github.com/joyent/go-triton/authentication
[7]: https://github.com/hashicorp/go-errwrap
