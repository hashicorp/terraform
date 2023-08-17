provider alpha {
  version = "1.2.3"
}

resource beta_resource b {}
resource gamma_resource g {}

mnptu {
  required_providers {
    alpha = {
      source = "acme/alpha"
    }
    beta = {
      source = "registry.example.com/acme/beta"
    }
  }
}

provider beta {
  region = "foo"
}
