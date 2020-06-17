# Gox - Simple Go Cross Compilation

Gox is a simple, no-frills tool for Go cross compilation that behaves a
lot like standard `go build`. Gox will parallelize builds for multiple
platforms. Gox will also build the cross-compilation toolchain for you.

## Installation

To install Gox, please use `go get`. We tag versions so feel free to
checkout that tag and compile.

```
$ go get github.com/mitchellh/gox
...
$ gox -h
...
```

## Usage

If you know how to use `go build`, then you know how to use Gox. For
example, to build the current package, specify no parameters and just
call `gox`. Gox will parallelize based on the number of CPUs you have
by default and build for every platform by default:

```
$ gox
Number of parallel builds: 4

-->      darwin/386: github.com/mitchellh/gox
-->    darwin/amd64: github.com/mitchellh/gox
-->       linux/386: github.com/mitchellh/gox
-->     linux/amd64: github.com/mitchellh/gox
-->       linux/arm: github.com/mitchellh/gox
-->     freebsd/386: github.com/mitchellh/gox
-->   freebsd/amd64: github.com/mitchellh/gox
-->     openbsd/386: github.com/mitchellh/gox
-->   openbsd/amd64: github.com/mitchellh/gox
-->     windows/386: github.com/mitchellh/gox
-->   windows/amd64: github.com/mitchellh/gox
-->     freebsd/arm: github.com/mitchellh/gox
-->      netbsd/386: github.com/mitchellh/gox
-->    netbsd/amd64: github.com/mitchellh/gox
-->      netbsd/arm: github.com/mitchellh/gox
-->       plan9/386: github.com/mitchellh/gox
```

Or, if you want to build a package and sub-packages:

```
$ gox ./...
...
```

Or, if you want to build multiple distinct packages:

```
$ gox github.com/mitchellh/gox github.com/hashicorp/serf
...
```

Or if you want to just build for linux:

```
$ gox -os="linux"
...
```

Or maybe you just want to build for 64-bit linux:

```
$ gox -osarch="linux/amd64"
...
```

And more! Just run `gox -h` for help and additional information.

## Versus Other Cross-Compile Tools

A big thanks to these other options for existing. They each paved the
way in many aspects to make Go cross-compilation approachable.

* [Dave Cheney's golang-crosscompile](https://github.com/davecheney/golang-crosscompile) -
  Gox compiles for multiple platforms and can therefore easily run on
  any platform Go supports, whereas Dave's scripts require a shell. Gox
  will also parallelize builds. Dave's scripts build sequentially. Gox has
  much easier to use OS/Arch filtering built in.

* [goxc](https://github.com/laher/goxc) -
  A very richly featured tool that can even do things such as build system
  packages, upload binaries, generate download webpages, etc. Gox is a
  super slim alternative that only cross-compiles binaries. Gox builds packages in parallel, whereas
  goxc doesn't. Gox doesn't enforce a specific output structure for built
  binaries.

