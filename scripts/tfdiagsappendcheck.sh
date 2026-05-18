#!/usr/bin/env bash
# Copyright IBM Corp. 2014, 2026
# SPDX-License-Identifier: BUSL-1.1


echo "==> Checking (tfdiags.Diagnostics).Append usage..."

if ! go run ./tools/tfdiagsappendcheck/main ./...; then
  echo "==> Found places where (tfdiags.Diagnostics).Append return value is ignored. Please fix the above issues and try again."
  exit 1
fi