
module "example" {
  new   = "b_override new"
  newer = "b_override newer"

  providers = {
    test = test.b_override
  }
}
