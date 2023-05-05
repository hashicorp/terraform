#!/usr/bin/env bash
# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

set -uo pipefail

if [[ $arch == 'arm' || $arch == 'arm64' ]]
then
    export DIR=$(mktemp -d)
    unzip -d $DIR "${e2e_cache_path}/terraform-e2etest_${os}_${arch}.zip"
    unzip -d $DIR "./terraform_${version}_${os}_${arch}.zip"
    sudo chmod +x $DIR/e2etest
    docker run --platform=linux/arm64 -v $DIR:/src -w /src arm64v8/alpine ./e2etest -test.v
else
    unzip "${e2e_cache_path}/terraform-e2etest_${os}_${arch}.zip"
    unzip "./terraform_${version}_${os}_${arch}.zip"
    TF_ACC=1 ./e2etest -test.v
fi