terraform {
  required_providers {
    cloud = {
      source = "tf.example.com/awesomecorp/happycloud"
    }
    null = {
      version = "2.0.1"
    }
  }
}

module "nested" {
  source = "./grandchild"
}
