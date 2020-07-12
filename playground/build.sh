#!/bin/bash

set -eu

BASE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$BASE"

rm -rf dist/*
mkdir -p dist
cp -r static/* dist/

GOOS=js GOARCH=wasm go build -o dist/terraform.wasm
