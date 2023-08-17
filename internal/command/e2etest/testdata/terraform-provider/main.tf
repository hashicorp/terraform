provider "mnptu" {

}

data "mnptu_remote_state" "test" {
  backend = "local"
  config = {
    path = "test.tfstate"
  }
}
