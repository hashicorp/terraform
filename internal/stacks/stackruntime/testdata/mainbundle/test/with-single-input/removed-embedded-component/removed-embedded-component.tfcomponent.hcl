
required_providers {
  testing = {
    source  = "hashicorp/testing"
    version = "0.1.0"
  }
}

stack "a" {
  source = "../removed-component"
}
