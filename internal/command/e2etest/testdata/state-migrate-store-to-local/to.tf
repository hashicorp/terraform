terraform {
  backend "local" {
    workspace_dir = "new-location"
    path = "new-location/default/terraform.tfstate"
  }
}

variable "name" {
  default = "world"
}

resource "terraform_data" "my-data" {
  input = "hello ${var.name}"
}
