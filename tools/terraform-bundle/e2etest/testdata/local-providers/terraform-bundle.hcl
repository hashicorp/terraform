terraform {
  version = "0.12.0"
}

providers {
  // this provider is installed in .plugins
  mycloud = {
    versions = ["0.1"]
    source   = "example.com/mycorp/mycloud"
  }
}
