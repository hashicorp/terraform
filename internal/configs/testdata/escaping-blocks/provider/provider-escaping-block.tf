
provider "foo" {
  alias = "bar"

  normal = "yes"

  _ {
    # This "escaping block" is an escape hatch for when a provider
    # declares argument names that collide with meta-argument
    # names. The examples below are not really realistic because they
    # are long-standing names that predate the need for escaping,
    # but we're using them as a proxy for new meta-arguments we might
    # add in future language editions which might collide with
    # names defined in pre-existing providers.

    # alias is set both as a meta-argument above _and_
    # as a provider-type-specific argument
    alias = "not actually alias"

    # version is only set in here, not as a meta-argument
    version = "not actually version"
  }
}
