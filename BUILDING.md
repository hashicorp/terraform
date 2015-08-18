# Building Terraform

This document contains details about the process for building binaries for
Terraform. 

## Versioning

As a pre-1.0 project, we use the MINOR and PATCH versions as follows:

 * a `MINOR` version increment indicates a release that may contain backwards
   incompatible changes
 * a `PATCH` version increment indicates a release that may contain bugfixes as
   well as additive (backwards compatible) features and enhancements

## Process

If only need to build binaries for the platform you're running (Windows, Linux,
Mac OS X etc..), you can follow the instructions in the README for [Developing
Terraform][1].

The guide below outlines the steps HashiCorp takes to build the official release 
binaries for Terraform. This process will generate a set of binaries for each supported
platform, using the [gox](https://github.com/mitchellh/gox) tool.

A Vagrant virtual machine is used to provide a consistent envirornment with
the pre-requisite tools in place. The specifics of this VM are defined in the 
[Vagrantfile](Vagrantfile).


```sh
# clone the repository if needed
git clone https://github.com/hashicorp/terraform.git
cd terraform

# Spin up a fresh build VM
vagrant destroy -f
vagrant up
vagrant ssh

# The Vagrantfile installs Go and configures the $GOPATH at /opt/gopath
# The current "terraform" directory is then sync'd into the gopath
cd /opt/gopath/src/github.com/hashicorp/terraform/

# Fetch dependencies
make updatedeps

# Verify unit tests pass
make test

# Build the release
# This generates binaries for each platform and places them in the pkg folder
make release
```

After running these commands, you should have 

# Prep release commit
export VERSION="vX.Y.Z"
# Edit CHANGELOG.md, adding current date to unreleased version header
# Edit version.go, setting VersionPrelease to empty string

# Snapshot dependency information
go get github.com/tools/godep
godep save ./...
cp Godeps/Godeps.json deps/$(echo $VERSION | sed 's/\./-/g').json

# Make and tag release commit (skipping Godeps dir)
git add CHANGELOG.md terraform/version.go deps/
git commit -a -m "${VERSION}"
git tag -m "${VERSION}" "${VERSION}"

# Build the release
make release

# Add Godeps for the archive
git add Godeps

# Make an archive with vendored dependencies
stashName=$(git stash create)
git archive -o terraform-$VERSION-src.tar.gz $stashName
git reset --hard ${VERSION}

# Zip and push release to bintray
export BINTRAY_API_KEY="..."
./scripts/dist "X.Y.Z" # no `v` prefix here

# -- "Point of no return" --
# -- Process can be aborted safely at any point before this --

# Push the release commit and tag
git push origin master
git push origin vX.Y.Z

# Click "publish" on the release from the Bintray Web UI
# Upload terraform-$VERSION-src.tar.gz as a file to the GitHub release.

# -- Release is complete! --

# Start release branch (to be used for reproducible builds and docs updates)
git checkout -b release/$VERSION
git push origin release/$VERSION

# Clean up master
git checkout master
# Set VersionPrerelease to "dev"
# Add new CHANGELOG section for next release
git add -A
git commit -m "release: clean up after ${VERSION}"
```


[1]: https://github.com/hashicorp/terraform#developing-terraform
