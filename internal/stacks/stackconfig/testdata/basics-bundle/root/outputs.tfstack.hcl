output "greeting" {
  type  = string
  value = stack.nested.greeting
}

output "sound" {
  type  = string
  value = local.sound
}
