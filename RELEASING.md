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
# Verify tests pass
make test

# Prep release commit
export VERSION="vX.Y.Z"
# Edit CHANGELOG, adding current date to unreleased version header
# Edit version.go, setting VersionPrelease to empty string

# Snapshot dependency information
godep save
mv Godeps/Godeps.json deps/$(echo $VERSION | sed 's/\./-/g').json
rm -rf Godeps

# Make and tag release commit
git commit -a -m "${VERSION}"
git tag -m "${VERSION}" "${VERSION}"

# Build release in Vagrant machine
vagrant destroy -f; vagrant up # Build a fresh VM for a clean build
vagrant ssh
cd /opt/gopath/src/github.com/hashicorp/terraform/
make release

# Zip and push release to bintray
export BINTRAY_API_KEY="..."
./scripts/dist "X.Y.Z" # no `v` prefix here

# -- "Point of no return" --
# -- Process can be aborted safely at any point before this --

# Push the release commit and tag
git push origin master
git push origin vX.Y.Z

# Click "publish" on the release from the Bintray Web UI

# -- Release is complete! --

# Make a follow-on commit to master restoring VersionPrerelease to "dev" and
setting up a new CHANGELOG section.
```
