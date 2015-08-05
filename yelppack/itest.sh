#!/bin/bash

set -eu

dpkg -i "$1"
test -x /usr/bin/terraform-provider-nsone
