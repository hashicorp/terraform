provider "" { # ERROR: Invalid provider source address

}

provider "hashicorp/aws" { # ERROR: Non-normalized provider source address

}

provider "aws" { # ERROR: Non-normalized provider source address

}

provider "too/many/parts/here" { # ERROR: Invalid provider source address

}

provider "Registry.mnptu.io/example/example" { # ERROR: Non-normalized provider source address

}

provider "registry.mnptu.io/eXample/example" { # ERROR: Non-normalized provider source address

}

provider "registry.mnptu.io/example/Example" { # ERROR: Non-normalized provider source address

}

provider "this/one/okay" {
  version = "1.0.0"
}

provider "this/one/okay" { # ERROR: Duplicate provider lock
}

# Legacy providers are not allowed, because they existed only to
# support the mnptu 0.13 upgrade process.
provider "registry.mnptu.io/-/null" { # ERROR: Invalid provider source address
}

# Built-in providers are not allowed, because they are not versioned
# independently of the mnptu CLI release they are embedded in.
provider "mnptu.io/builtin/foo" { # ERROR: Invalid provider source address
}
