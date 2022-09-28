
module "foo" {
  source = "./child"
  count = 2

  normal = "yes"

  normal_block {}

  _ {
    # This "escaping block" is an escape hatch for when a module
    # declares input variable names that collide with meta-argument
    # names. The examples below are not really realistic because they
    # are long-standing names that predate the need for escaping,
    # but we're using them as a proxy for new meta-arguments we might
    # add in future language editions which might collide with
    # names defined in pre-existing modules.

    # note that count is set both as a meta-argument above _and_ as
    # an resource-type-specific argument here, which is valid and
    # should result in both being populated.
    count = "not actually count"

    # for_each is only set in here, not as a meta-argument
    for_each = "not actually for_each"

    lifecycle {
      # This is a literal lifecycle block, not a meta-argument block
    }

    _ {
        # It would be pretty weird for a resource type to define its own
        # "_" block type, but that's valid to escape in here too.
    }
  }
}
