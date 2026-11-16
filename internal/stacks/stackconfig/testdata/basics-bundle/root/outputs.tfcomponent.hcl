output "greeting" {
  type  = string
  value = stack.nested.greeting
}

output "sound" {
  type  = string
  value = local.sound
}

output "password" {
  type      = string
  value     = "not really"
  sensitive = true
  ephemeral = true
}
