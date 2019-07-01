provider "terraform" {

}

data "terraform_remote_state" "test" {
  backend = "local"
  config = {
    path = "nothing.tfstate"
  }
}
