#!/bin/bash

# This script rewrites [GH-nnnn]-style references in the CHANGELOG.md file to
# be Markdown links to the given github issues.
#
# This is run during releases so that the issue references in all of the
# released items are presented as clickable links, but we can just use the
# easy [GH-nnnn] shorthand for quickly adding items to the "Unrelease" section
# while merging things between releases.

set -e

SCRIPT_DIR=$(dirname "$(readlink -f "$0")")

cd "$SCRIPT_DIR/.."
sed -ri 's/\[GH-([0-9]+)\]/\(\[#\1\]\(https:\/\/github.com\/hashicorp\/terraform\/issues\/\1\)\)/' CHANGELOG.md
