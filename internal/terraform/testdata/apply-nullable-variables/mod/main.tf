// optional, and this can take null as an input
variable "nullable_null_default" {
  // This is implied now as the default, and probably should be implied even
  // when nullable=false is the default, so we're leaving this unset for the test.
  // nullable = true

  default = null
}

// assigning null can still override the default.
variable "nullable_non_null_default" {
  nullable = true
  default = "ok"
}

// required, and assigning null is valid.
variable "nullable_no_default" {
  nullable = true
}


// this combination is invalid
//variable "non_nullable_null_default" {
//  nullable = false
//  default = null
//}


// assigning null will take the default
variable "non_nullable_default" {
  nullable = false
  default = "ok"
}

// required, but null is not a valid value
variable "non_nullable_no_default" {
  nullable = false
}

output "nullable_null_default" {
  value = var.nullable_null_default
}

output "nullable_non_null_default" {
  value = var.nullable_non_null_default
}

output "nullable_no_default" {
  value = var.nullable_no_default
}

output "non_nullable_default" {
  value = var.non_nullable_default
}

output "non_nullable_no_default" {
  value = var.non_nullable_no_default
}

