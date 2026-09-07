terraform {
  cloud {
    organization = "sarahfrench"
    workspaces {
      name = "test-cloud-backend"
    }
  }
}
