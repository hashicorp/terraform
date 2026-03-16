required_providers {
  testing = {
    source  = "hashicorp/testing"
    version = "0.1.0"
  }
}

removed {
  from = stack.multiple
  source = "../"
}
