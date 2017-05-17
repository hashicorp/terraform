// We test this in a child module, since the root module state exists
// very early on, even before any resources are created in it, but that is not
// true for child modules.

module "child" {
  source = "./child"
}
