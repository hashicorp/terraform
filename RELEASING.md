# Releasing Terraform

This document contains details about the Terraform release process.

## Schedule

Terraform currently has no fixed release schedule, the HashiCorp maintainers
can usually give a feel for roughly when the next release is planned.

## Versioning

As a pre-1.0 project, we use the MINOR and PATCH versions as follows:

 * a `MINOR` version increment indicates a release that may contain backwards
   incompatible changes
 * a `PATCH` version increment indicates a release that may contain bugfixes as
   well as additive (backwards compatible) features and enhancements

## Process

For maintainer documentation purposes, here is the current release process:

```sh
# Spin up a fresh build VM
vagrant destroy -f
vagrant up
vagrant ssh
cd /opt/gopath/src/github.com/hashicorp/terraform/

# Fetch dependencies
make updatedeps

# Verify unit tests pass
make test

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

# Make an archive with vendored dependencies
stashName=$(git stash)
git archive -o terraform-$VERSION-src.tar.gz $stashName

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
