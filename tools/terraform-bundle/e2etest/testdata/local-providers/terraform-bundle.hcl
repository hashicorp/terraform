terraform {
  version = "0.13.0"
}

providers {
  // this provider is installed in .plugins
  mycloud = {
    versions = ["0.1"]
    source   = "example.com/myorg/mycloud"
  }
}
