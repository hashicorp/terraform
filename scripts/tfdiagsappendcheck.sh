#!/usr/bin/env bash
# Copyright IBM Corp. 2014, 2026
# SPDX-License-Identifier: BUSL-1.1


set -euo pipefail

echo "==> Checking (tfdiags.Diagnostics).Append usage..."

collect_findings() {
  local output_file="$1"
  local analyzer_output

  set +e
  analyzer_output="$(go run ./tools/tfdiagsappendcheck/main ./... 2>&1)"
  local analyzer_status=$?
  set -e

  if [[ ${analyzer_status} -ne 0 ]] && ! grep -q "ignored return value from tfdiags.Diagnostics.Append" <<<"${analyzer_output}"; then
    echo >&2 "==> tfdiagsappendcheck failed unexpectedly:"
    echo >&2 "${analyzer_output}"
    exit ${analyzer_status}
  fi

  grep -F "ignored return value from tfdiags.Diagnostics.Append" <<<"${analyzer_output}" | sort -u >"${output_file}" || true
}

# In pull request checks we compare findings in the base branch with findings
# in the PR branch, and fail only for newly introduced findings.
if [[ -n "${GITHUB_BASE_REF:-}" ]]; then
  base_branch="origin/${GITHUB_BASE_REF}"
  tmp_dir="$(mktemp -d)"
  trap 'rm -rf "${tmp_dir}"' EXIT

  base_output="${tmp_dir}/base-findings.txt"
  head_output="${tmp_dir}/head-findings.txt"
  current_head="$(git rev-parse HEAD)"

  echo "==> Comparing findings against ${base_branch}..."
  git fetch --no-tags --depth=1 origin "${GITHUB_BASE_REF}"

  git checkout --detach "${base_branch}"
  collect_findings "${base_output}"

  git checkout --detach "${current_head}"
  collect_findings "${head_output}"

  if new_findings="$(comm -13 "${base_output}" "${head_output}")" && [[ -n "${new_findings}" ]]; then
    echo >&2 "==> Found newly introduced places where (tfdiags.Diagnostics).Append return value is ignored:"
    echo >&2 "${new_findings}"
    exit 1
  fi

  echo "==> No newly introduced tfdiags.Diagnostics.Append findings relative to ${base_branch}."
  exit 0
fi


# Script is not running in a pull request context.
# Run the analyzer on the entire codebase.
if ! go run ./tools/tfdiagsappendcheck/main ./...; then
  echo "==> Found places where (tfdiags.Diagnostics).Append return value is ignored. Please fix the above issues and try again."
  exit 1
fi