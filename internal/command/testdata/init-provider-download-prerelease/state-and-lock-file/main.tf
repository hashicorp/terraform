terraform {
  # No provider requirements, incl. version constraints, in the config

  backend "local" {
    path = "./state-using-random-provider.tfstate"
  }
}
