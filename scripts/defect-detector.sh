#!/usr/bin/env bash
# Copyright IBM Corp. 2014, 2026
# SPDX-License-Identifier: BUSL-1.1


set -euo pipefail

echo "==> Checking (tfdiags.Diagnostics).Append usage..."

# Use the analyzer binary built from the current branch to check for findings in the currently checked-out code.
# Output written to the specified location (output_file) is used and cleaned up by calling code.
collect_findings() {
  local analyzer_bin="$1"
  local output_file="$2"
  local analyzer_output

  set +e
  analyzer_output="$("${analyzer_bin}" ./... 2>&1)"
  local analyzer_status=$?
  set -e

  if [[ ${analyzer_status} -ne 0 ]] && ! grep -q "ignored return value from tfdiags.Diagnostics.Append" <<<"${analyzer_output}"; then
    echo >&2 "==> defect-detector failed unexpectedly:"
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

  base_output="${tmp_dir}/base-findings.txt"
  head_output="${tmp_dir}/head-findings.txt"
  analyzer_bin="${tmp_dir}/defectdetector"
  current_head="$(git rev-parse HEAD)"

  cleanup() {
    git checkout --detach "${current_head}" >/dev/null 2>&1 || true
    rm -rf "${tmp_dir}"
  }
  trap cleanup EXIT

  echo "==> Building analyzer binary from current branch..."
  go build -o "${analyzer_bin}" ./tools/defect-detector/main

  echo "==> Comparing findings against ${base_branch}..."
  git fetch --no-tags --depth=1 origin "${GITHUB_BASE_REF}"

  git checkout --detach "${base_branch}"
  collect_findings "${analyzer_bin}" "${base_output}"

  git checkout --detach "${current_head}"
  collect_findings "${analyzer_bin}" "${head_output}"

  # Compare findings between base and head branches, if there is new content present only in head
  # then the check is failed and details printed to output.
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
if ! go run ./tools/defect-detector/main ./...; then
  echo "==> Found places where (tfdiags.Diagnostics).Append return value is ignored. Please fix the above issues and try again."
  exit 1
fi