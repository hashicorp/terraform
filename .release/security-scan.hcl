# Copyright IBM Corp. 2014, 2025
# SPDX-License-Identifier: BUSL-1.1

container {
  dependencies = false
  alpine_secdb = true
  secrets      = false
}

binary {
  secrets      = true
  go_modules   = true
  osv          = true
  nvd          = false
}