terraform {
  required_providers {
    test = {
      source = "mycorp/test"
    }
  }
}

provider "TEST" {

}

resource test_resource "test" {
  // this resource is (implicitly) provided by "mycorp/test"
}

resource test_resource "TEST" {
  // this resource is (explicitly) provided by "hashicorp/test"
  provider = TEST
}
