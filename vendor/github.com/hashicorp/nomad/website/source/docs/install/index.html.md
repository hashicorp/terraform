---
layout: "docs"
page_title: "Install Nomad"
sidebar_current: "docs-install"
description: |-
  Learn how to install Nomad.
---

# Install Nomad

Installing Nomad is simple. There are two approaches to installing Nomad:
downloading a precompiled binary for your system, or installing from source.

Downloading a precompiled binary is easiest, and we provide downloads over
TLS along with SHA256 sums to verify the binary.

## Precompiled Binaries

To install the precompiled binary,
[download](/downloads.html) the appropriate package for your system.
Nomad is currently packaged as a zip file. We do not have any near term
plans to provide system packages.

Once the zip is downloaded, unzip it into any directory. The
`nomad` binary inside is all that is necessary to run Nomad (or
`nomad.exe` for Windows). Any additional files, if any, aren't
required to run Nomad.

Copy the binary to anywhere on your system. If you intend to access it
from the command-line, make sure to place it somewhere on your `PATH`.

## Compiling from Source

To compile from source, you will need [Go](https://golang.org) installed and
configured properly (including a `GOPATH` environment variable set), as well
as a copy of [`git`](https://www.git-scm.com/) in your `PATH`.

  1. Clone the Nomad repository into your `GOPATH`: `mkdir -p $GOPATH/src/github.com/hashicorp && cd $GOPATH/src/github.com/hashicorp && git clone https://github.com/hashicorp/nomad.git && cd nomad`

  1. Run `make bootstrap`. This will download and compile libraries and tools needed
     to compile Nomad.

  1. Run `make dev`. This will build Nomad for your current system and put
     the binary in `./bin/` (relative to the git checkout).  The `make dev`
     target is just a shortcut that builds `nomad` for only your local build
     environment (no cross-compiled targets).  If you would like to
     cross-compile Nomad for different platforms, just run `make`.

  1. Run `make install`.  This will install `./bin/nomad` into
     `/usr/local/bin/nomad`.

## Verifying the Installation

To verify Nomad is properly installed, execute the `nomad` binary on
your system. You should see help output. If you are executing it from
the command line, make sure it is on your `PATH` or you may get an error
about `nomad` not being found.
