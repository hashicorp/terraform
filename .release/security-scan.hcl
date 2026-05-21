# Copyright IBM Corp. 2014, 2026
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
  
  triage {
    suppress {
      vulnerabilities = [
        // These vulnerabilities all point to the same issue.
        // https://test.osv.dev/vulnerability/GO-2026-4762
        "GHSA-p77j-4mvh-x3m3",
        "GO-2026-4762",
        "CVE-2026-33186",
      ]
    }
  }
}
