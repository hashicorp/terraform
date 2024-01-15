# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: BUSL-1.1

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