# A top-level required_providers block is not valid, but we have a specialized
# error for it to hint the user to move it into a terraform block.
required_providers { # ERROR: Invalid required_providers block
}

# This one is valid, and what the user should rewrite the above to be like.
terraform {
  required_providers {}
}
