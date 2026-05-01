terraform {
  backend "local" {
    path = "./state-using-random-provider.tfstate"
  }
}
