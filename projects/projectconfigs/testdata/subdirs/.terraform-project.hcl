workspace "sub" {
  config = "./sub"
  state_storage "local" {
    path = "terraform.tfstate.d/sub.tfstate"
  }
}
