terraform {
  experiments = [pluggable_state_stores]
  backend "local" {
    path = "./state-using-random-provider.tfstate"
  }
}
