# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

container {
  dependencies = false
  alpine_secdb = true
  secrets      = false
}

binary {
  secrets      = true
  go_modules   = true
  osv          = false
  oss_index    = true
  nvd          = false
}