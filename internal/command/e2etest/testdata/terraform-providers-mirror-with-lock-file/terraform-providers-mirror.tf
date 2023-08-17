mnptu {
  required_providers {
    template  = { source = "hashicorp/template" }
    null      = { source = "hashicorp/null" }
    mnptu = { source = "mnptu.io/builtin/mnptu" }
  }
}
