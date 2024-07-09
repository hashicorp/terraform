
action "example" {
  variable "a" {
  }

  step "b" {
    receiver = resource.thingy.example
    action   = do_it

    arguments {
      in = var.a.out
    }
  }

  step "c" {
    receiver = resource.thingy.example
    action   = do_it_again

    arguments {
      in = step.b.out
    }
  }

  output "d" {
    value = step.c
  }
}

resource "thingy" "example" {
  lifecycle {
    when_creating {
      step "foo" {
        action = example
      }

      step "bar" {
        receiver = self
        action   = do_it

        arguments {
          in = step.foo.out
        }
      }
    }

    when_updating {
      step "foo" {
        action = example

        arguments {
          a = self.id
        }
      }
    }

    when_destroying {
      step "foo" {
        action = example
      }
    }
  }
}
