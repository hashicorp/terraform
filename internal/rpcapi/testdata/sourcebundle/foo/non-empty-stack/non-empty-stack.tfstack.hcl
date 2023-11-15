
component "single" {
  source = "./empty-module"
}

component "for_each" {
  source   = "./empty-module"
  for_each = {}
}

stack "single" {
  source = "./child"
}

stack "for_each" {
  source   = "./child"
  for_each = {}
}
