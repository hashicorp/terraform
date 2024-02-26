// This in-repo component has an invalid root module
component "in_repo_invalid" {
  source = "./invalid"
}

// This in-repo component has a valid root module and an invalid child in-repo
// module
component "in_repo_invalid_child" {
  source = "./invalid_child"
}

// This in-repo component has a remote source child module with a valid root
// module and an invalid child in-repo module
component "in_repo_invalid_nested_remote" {
  source = "./invalid_nested_remote"
}

// This remote source component has an invalid root module
component "remote_invalid" {
  source = "https://testing.invalid/invalid.tar.gz"
}

// This remote source component has an invalid child in-repo module
component "remote_invalid_child" {
  source = "https://testing.invalid/invalid_child.tar.gz"
}

// This remote source component has two invalid grandchildren which are both
// in-repo modules and share the same relative source
component "remote_invalid_grandchildren" {
  source = "https://testing.invalid/invalid_grandchildren.tar.gz"
}
