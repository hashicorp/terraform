terraform {
  required_version = ">200.0.0"

  bad {
    block = "false"
  }

  required_providers {
    bang = {
      oops = "boom"
    }
  }
}

nope {
  boom {}
}
