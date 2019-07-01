
module "example" {
  source = "./example2-a_override"

  foo = "a_override foo"
  new = "a_override new"
}
