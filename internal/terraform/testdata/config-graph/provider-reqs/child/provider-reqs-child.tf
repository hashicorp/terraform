terraform {
  required_providers {
    cloud = {
      source = "tf.example.com/awesomecorp/happycloud"
    }
    null = {
      # This should merge with the null provider constraint in the root module
      version = "2.0.1"
    }
  }
}

module "nested" {
  source = "./grandchild"
}
