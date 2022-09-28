variable "bad_string" {
  type = "string" # ERROR: Invalid quoted type constraints
}

variable "bad_map" {
  type = "map" # ERROR: Invalid quoted type constraints
}

variable "bad_list" {
  type = "list" # ERROR: Invalid quoted type constraints
}
