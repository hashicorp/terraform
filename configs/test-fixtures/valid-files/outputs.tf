
output "foo" {
  value = "hello"
}

output "bar" {
  value = local.bar
}

output "baz" {
  value     = "ssshhhhhhh"
  sensitive = true
}

output "cheeze_pizza" {
  description = "Nothing special"
  value       = "🍕"
}

output "π" {
  value = 3.14159265359
  depends_on = [
    pizza.cheese,
  ]
}
