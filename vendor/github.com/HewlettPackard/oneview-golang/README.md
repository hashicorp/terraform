# HPE OneView golang bindings

[![Build Status](https://travis-ci.org/HewlettPackard/oneview-golang.svg?branch=master)](https://travis-ci.org/HewlettPackard/oneview-golang)

HPE OneView allows you to treat your physical infrastructure as code, and now
you can integrate your favorite tools based in golang with HPE OneView.

## Build requirements
We use docker to build and test, run this project on a system that has docker. 
If you don't use docker, you will need to install and setup go-lang locally as
well as any other make requirements.  You can use `USE_CONTAINER=false` environment
setting for make to avoid using docker. Otherwise make sure to have these tools:
- docker client and daemon
- gnu make tools

## Testing your changes

### From a container
```
make test
```

### Without docker
* Install golang 1.5 or better
* Install go packages listed in .travis.yml
```
USE_CONTAINER=false make test
```

## Contributing

Want to hack on oneview-golang? Please start with the [Contributing Guide](https://github.com/HewlettPackard/docker-machine-oneview/blob/master/CONTRIBUTING.md).

This code base is primarily consumed and used by the docker-machine-oneview project.  We will follow contribution standards set by this project.

## Hacking Guide

Use the [hacking guide](HACKING.md) to run local acceptance testing and debugging local contributions.
Currently test cases are not supported portion of our CI/CD approval process but might be made available from this test suite in the future.   Contributions to the test suite is appreciated.

## License
This project is licensed under the Apache License, Version 2.0.  See LICENSE for full license text.