gocommon
========

Common Go library for Joyent's Triton and Manta.

[![wercker status](https://app.wercker.com/status/2f63bf7f68bfdd46b979abad19c0bee0/s/master "wercker status")](https://app.wercker.com/project/byKey/2f63bf7f68bfdd46b979abad19c0bee0)

## Installation

Use `go-get` to install gocommon.
```
go get github.com/joyent/gocommon
```

## Documentation

Auto-generated documentation can be found on godoc.

- [github.com/joyent/gocommon](http://godoc.org/github.com/joyent/gocommon)
- [github.com/joyent/gocommon/client](http://godoc.org/github.com/joyent/client)
- [github.com/joyent/gocommon/errors](http://godoc.org/github.com/joyent/gocommon/errors)
- [github.com/joyent/gocommon/http](http://godoc.org/github.com/joyent/gocommon/http)
- [github.com/joyent/gocommon/jpc](http://godoc.org/github.com/joyent/gocommon/jpc)
- [github.com/joyent/gocommon/testing](http://godoc.org/github.com/joyent/gocommon/testing)


## Contributing

Report bugs and request features using [GitHub Issues](https://github.com/joyent/gocommon/issues), or contribute code via a [GitHub Pull Request](https://github.com/joyent/gocommon/pulls). Changes will be code reviewed before merging. In the near future, automated tests will be run, but in the meantime please `go fmt`, `go lint`, and test all contributions.


## Developing

This library assumes a Go development environment setup based on [How to Write Go Code](https://golang.org/doc/code.html). Your GOPATH environment variable should be pointed at your workspace directory.

You can now use `go get github.com/joyent/gocommon` to install the repository to the correct location, but if you are intending on contributing back a change you may want to consider cloning the repository via git yourself. This way you can have a single source tree for all Joyent Go projects with each repo having two remotes -- your own fork on GitHub and the upstream origin.

For example if your GOPATH is `~/src/joyent/go` and you're working on multiple repos then that directory tree might look like:

```
~/src/joyent/go/
|_ pkg/
|_ src/
   |_ github.com
      |_ joyent
         |_ gocommon
         |_ gomanta
         |_ gosdc
         |_ gosign
```

### Recommended Setup

```
$ mkdir -p ${GOPATH}/src/github.com/joyent
$ cd ${GOPATH}/src/github.com/joyent
$ git clone git@github.com:<yourname>/gocommon.git

# fetch dependencies
$ git clone git@github.com:<yourname>/gosign.git
$ go get -v -t ./...

# add upstream remote
$ cd gocommon
$ git remote add upstream git@github.com:joyent/gocommon.git
$ git remote -v
origin  git@github.com:<yourname>/gocommon.git (fetch)
origin  git@github.com:<yourname>/gocommon.git (push)
upstream        git@github.com:joyent/gocommon.git (fetch)
upstream        git@github.com:joyent/gocommon.git (push)
```

### Run Tests

The library needs values for the `SDC_URL`, `MANTA_URL`, `MANTA_KEY_ID` and `SDC_KEY_ID` environment variables even though the tests are run locally. You can generate a temporary key and use its fingerprint for tests without adding the key to your Triton Cloud account.

```
# create a temporary key
ssh-keygen -b 2048 -C "Testing Key" -f /tmp/id_rsa -t rsa -P ""

# set up environment
# note: leave the -E md5 argument off on older ssh-keygen
export KEY_ID=$(ssh-keygen -E md5 -lf /tmp/id_rsa | awk -F' ' '{print $2}' | cut -d':' -f2-)
export SDC_KEY_ID=${KEY_ID}
export MANTA_KEY_ID=${KEY_ID}
export SDC_URL=https://us-east-1.api.joyent.com
export MANTA_URL=https://us-east.manta.joyent.com

cd ${GOPATH}/src/github.com/joyent/gocommon
go test ./...
```

### Build the Library

```
cd ${GOPATH}/src/github.com/joyent/gocommon
go build ./...
```
