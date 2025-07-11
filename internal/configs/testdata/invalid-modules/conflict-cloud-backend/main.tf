terraform {
  backend "foo" {}

  cloud {
    organization = "sarahfrench"
    workspaces {
      name = "test-cloud-backend"
    }
  }
}
