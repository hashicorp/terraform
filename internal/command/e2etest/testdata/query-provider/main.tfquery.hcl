list "simple_resource" "test" {
  provider = simple
  include_resource = true
  config {
    value = "dynamic_value"
  }
}