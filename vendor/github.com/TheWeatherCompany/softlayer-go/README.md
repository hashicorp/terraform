softlayer-go [![Build Status](https://travis-ci.org/TheWeatherCompany/softlayer-go.svg?branch=master)](https://travis-ci.org/TheWeatherCompany/softlayer-go#) [![Join the chat at https://gitter.im/TheWeatherCompany/softlayer-go](https://badges.gitter.im/TheWeatherCompany/softlayer-go.svg)](https://gitter.im/TheWeatherCompany/softlayer-go?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge&utm_content=badge)
============

An *incomplete* SoftLayer (SL) client API written in Go language.

## Getting Started
------------------

The best way to get started would be to look at the [integration](integration) tests for creating a [virtual guest](https://github.com/TheWeatherCompany/softlayer-go/blob/master/integration/virtual_guest_lifecycle/virtual_guest_lifecycle_test.go) and the [test helpers](test_helpers). Here is a snippet of what is needed.

```go
//Add necessary imports, e.g., os, slclient, datatypes
// "os"
// slclient "github.com/TheWeatherCompany/softlayer-go/client"
// datatypes "github.com/TheWeatherCompany/softlayer-go/data_types"

//Access SoftLayer username and API key from environment variable or hardcode here
username := os.Getenv("SL_USERNAME")
apiKey := os.Getenv("SL_API_KEY")
	
//Create a softLayer-go client
client := slclient.NewSoftLayerClient(username, apiKey)

//Create a template for the virtual guest (changing properties as needed)
virtualGuestTemplate := datatypes.SoftLayer_Virtual_Guest_Template{
  Hostname:  "some-hostname",
	Domain:    "some-domain.com",
	StartCpus: 1,
	MaxMemory: 1024,
	Datacenter: datatypes.Datacenter{
		Name: "ams01",
	},
	SshKeys:                      []SshKey{},  //or get the necessary keys and add here
	HourlyBillingFlag:            true,
	LocalDiskFlag:                true,
	OperatingSystemReferenceCode: "UBUNTU_LATEST",
}
	
//Get the SoftLayer virtual guest service
virtualGuestService, err := client.GetSoftLayer_Virtual_Guest_Service()
if err != nil {
  return err
}
	
//Create the virtual guest with the service
virtualGuest, err := virtualGuestService.CreateObject(virtualGuestTemplate)
if err != nil {
	return err
}
	
//Use the virtualGuest or other services...
```

### Overview Presentations (*)
--------------------------

TBD

### Cloning and Building
------------------------

Clone this repo and build it. Using the following commands on a Linux or Mac OS X system:

```
$ mkdir -p softlayer-go/src/github.com/TheWeatherCompany
$ export GOPATH=$(pwd)/softlayer-go:$GOPATH
$ cd softlayer-go/src/github.com/TheWeatherCompany
$ git clone https://github.com/TheWeatherCompany/softlayer-go.git
$ cd softlayer-go
$ export SL_USERNAME=your-username@your-org.com
$ export SL_API_KEY=your-softlayer-api-key
$ godep restore
$ ./bin/build
$ ./bin/test-unit
$ ./bin/test-integration
```

NOTE: you may need to install [godep](https://github.com/tools/godep) on your system, if you have not already. You can with this one line command: `$ go get github.com/tools/godep`

NOTE2: if you get any dependency errors, then use `go get path/to/dependency` to get it, e.g., `go get github.com/onsi/ginkgo` and `go get github.com/onsi/gomega`. You also need to do `godep save ./...` in order for any new or updated depencies to be reflected into the `Godeps` directory.

The executable output should now be located in: `out/slgo`. It does not do anything currently, expect printing a version number. In time this may change. For now, this project is intended to be a set of useful and reusable Golang libraries to access SoftLayer.

### Running Tests
-----------------

The [SoftLayer](http://www.softlayer.com) (SL) Golang client and associated tests and binary distribution depend on you having a real SL account. Get one for free for one month [here](http://www.softlayer.com/info/free-cloud). From your SL account you can get an API key. Using your account name and API key you will need to set two environment variables: `SL_USERNAME` and `SL_API_KEY`. You can do so as follows:

```
$ export SL_USERNAME=your-username@your-org.com
$ export SL_API_KEY=your-softlayer-api-key
```

You should run the tests to make sure all is well, do this with: `$ ./bin/test-unit` and `$ ./bin/test-integration` in your cloned repository. Please note that the `$ ./bin/test-integration` will spin up real SoftLayer virtual guests (VMs) and associated resources and will also delete them. This integration test may take up to 30 minutes (usually shorter)

The output should of `$ ./bin/test-unit` be similar to:

```
➜  softlayer-go git:(master) bin/test-unit

 Cleaning build artifacts...

 Formatting packages...

 Unit Testing packages:
[1457666427] SoftLayer Client Suite - 14/14 specs - 4 nodes •••••••••••••• SUCCESS! 40.325961ms
[1457666427] Services Suite - 327/327 specs - 4 nodes ••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••• SUCCESS! 288.0041ms
[1457666427] Common Suite - 4/4 specs - 4 nodes •••• SUCCESS! 13.394673ms

Ginkgo ran 3 suites in 5.387042692s
Test Suite Passed

 Vetting packages for potential issues...

SWEET SUITE SUCCESS
```

## Developing (*)
-----------------

1. Check for existing stories on our [public Tracker](https://www.pivotaltracker.com/n/projects/1344876)
2. Select an unstarted story and work on code for it
3. If the story you want to work on is not there then open an issue and ask for a new story to be created
4. Run `go get golang.org/x/tools/cmd/vet`
5. Run `go get github.com/xxx ...` to install test dependencies (as you see errors)
6. Write a [Ginkgo](https://github.com/onsi/ginkgo) test
7. Run `bin/test` and watch the test fail
8. Make the test pass
9. Submit a pull request

## Contributing
---------------

* We gratefully acknowledge and thank the [current contributors](https://github.com/TheWeatherCompany/softlayer-go/graphs/contributors)
* We welcome any and all contributions as Pull Requests (PR)
* We also welcome issues and bug report and new feature request. We will address as time permits
* Follow the steps above in Developing to get your system setup correctly
* Please make sure your PR is passing Travis before submitting
* Feel free to email me or the current collaborators if you have additional questions about contributions
* Before submitting your first PR, please read and follow steps in [CONTRIBUTING.md](CONTRIBUTING.md)

### Managing dependencies
-------------------------

* All dependencies managed via [Godep](https://github.com/tools/godep). See [Godeps/_workspace](https://github.com/TheWeatherCompany/softlayer-go/tree/master/Godeps/_workspace) directory on master

#### Short `godep` Guide
* If you ever import a new package `foo/bar` (after you `go get foo/bar`, so that foo/bar is in `$GOPATH`), you can type `godep save ./...` to add it to the `Godeps` directory.
* To restore dependencies from the `Godeps` directory, simply use `godep restore`. `restore` is the opposite of `save`.
* If you ever remove a dependency or a link becomes deprecated, the easiest way is probably to remove your entire `Godeps` directory and run `godep save ./...` again, after making sure all your dependencies are in your `$GOPATH`. Don't manually edit `Godeps.json`!
* To update an existing dependency, you can use `godep update foo/bar` or `godep update foo/...` (where `...` is a wildcard)
* The godep project [readme](https://github.com/tools/godep/README.md) is a pretty good resource: [https://github.com/tools/godep](https://github.com/tools/godep)

### Current conventions
-----------------------

* Basic Go conventions
* Strict TDD for any code added or changed
* Go fakes when needing to mock objects

(*) these items are in the works, we will remove the * once they are available

**NOTE**: this client is created to support the [bosh-softlayer-cpi](https://github.com/TheWeatherCompany/bosh-softlayer-cpi) project and only implements the portion of the SL APIs needed to complete the implementation of the BOSH CPI. You are welcome to use it in your own projects and as you do if you find areas we have not yet implemented but that you need, please submit [Pull Requests](https://help.github.com/articles/using-pull-requests/) or engage with us in discussions.
