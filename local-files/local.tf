terraform {
  required_version = ">=0.15, !=0.16"
  required_providers {
    local = {
      source="registry.terraform.io/hashicorp/local"
      version="~>2.1.0"
    }
  }
}

# Create local file
resource "local_file" "mypets" {
  filename        = "${path.module}/mypets.txt"
  content         = "${path.root}\n${path.module}\n${terraform.workspace}"
  file_permission = "0664"
}
