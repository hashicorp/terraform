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
        // golang.org/x/crypto/openpgp is deprecated/unmaintained with no fixed
        // version. The built terraform binary does not use this package
        // (confirmed via `go mod why golang.org/x/crypto/openpgp`).
        // It is only used in the copywrite tool.
        "GO-2026-5932",
        // These vulnerabilities all point to the same issue.
        // https://test.osv.dev/vulnerability/GO-2026-4762
        "GHSA-p77j-4mvh-x3m3",
        "GO-2026-4762",
        "CVE-2026-33186",
      ]
    }
  }
}
