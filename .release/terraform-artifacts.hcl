# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: BUSL-1.1

schema = 1
artifacts {
  zip = [
    "terraform_${version}_darwin_amd64.zip",
    "terraform_${version}_darwin_arm64.zip",
    "terraform_${version}_freebsd_386.zip",
    "terraform_${version}_freebsd_amd64.zip",
    "terraform_${version}_freebsd_arm.zip",
    "terraform_${version}_linux_386.zip",
    "terraform_${version}_linux_amd64.zip",
    "terraform_${version}_linux_arm.zip",
    "terraform_${version}_linux_arm64.zip",
    "terraform_${version}_openbsd_386.zip",
    "terraform_${version}_openbsd_amd64.zip",
    "terraform_${version}_solaris_amd64.zip",
    "terraform_${version}_windows_386.zip",
    "terraform_${version}_windows_amd64.zip",
  ]
  rpm = [
    "terraform-${version_linux}-1.aarch64.rpm",
    "terraform-${version_linux}-1.armv7hl.rpm",
    "terraform-${version_linux}-1.i386.rpm",
    "terraform-${version_linux}-1.x86_64.rpm",
  ]
  deb = [
    "terraform_${version_linux}-1_amd64.deb",
    "terraform_${version_linux}-1_arm64.deb",
    "terraform_${version_linux}-1_armhf.deb",
    "terraform_${version_linux}-1_i386.deb",
  ]
  container = [
    "terraform_default_linux_386_${version}_${commit_sha}.docker.tar",
    "terraform_default_linux_amd64_${version}_${commit_sha}.docker.tar",
    "terraform_default_linux_arm64_${version}_${commit_sha}.docker.tar",
    "terraform_default_linux_arm_${version}_${commit_sha}.docker.tar",
  ]
}
