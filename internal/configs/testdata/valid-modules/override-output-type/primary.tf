output "fully_overridden" {
  value = "hello"
  type  = string
}

output "no_override" {
  value = "hello"
  type  = string
}

output "type_added_by_override" {
  value = "hello"
}
