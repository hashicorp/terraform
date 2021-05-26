---
layout: "language"
page_title: "Configuration Refactoring - Configuration Language"
---

# Configuration Refactoring

When maintaining a module or family of modules over a long period, you might
wish to change the names and locations of existing resource instances without
Terraform proposing to destroy the existing object and create a new one at
the new address.

The Terraform language includes some mechanisms, described below, which can
allow module authors the freedom to do certain kinds of refactoring without
the need to carefully coordinate with existing users of a module that might
already have real infrastructure objects bound to the old addresses.

## Refactoring Concepts

Before thinking about refactoring it's worth understanding some concepts and
mechanisms within Terraform which are in most situations more like an
implementation detail, but that which become relevant when you need to
arrange for a seamless upgrade to a newer version of your Terraform module
after refactoring.

In the early life of a Terraform module it will typically contain only
declarations of resources, possibly factored out into multiple modules.
Terraform uses [State](../state/) to automatically track the relationships
between resource instances declared in your configuration and the real
infrastructure objects they represent.

When applying a configuration for the first time, Terraform will consult the
(currently empty) state and notice that no remote objects are bound to any
of your resource instances and so it will propose to create new ones.

When you create Terraform plans on subsequent runs, Terraform will notice that
remote objects already exist and will consult the remote system to make sure
its record (in the Terraform State) is up to date. It will then compare that
updated state with the values declared in the current configuration, and if it
detects any differences (with the help of the relevant provider plugins) it'll
then propose to take one or more actions to change the remote objects so that
they will match the configuration.

This mechanism of comparing configuration to state is what drives Terraform's
normal planning behavior. Terraform configuration is declarative, and so
the configuration defines a _desired state_, and it's Terraform's job to
propose a set of actions to move from the current state into that desired
state.

Crucial to Terraform's planning mechanism is the idea of
_resource instance addresses_, which allow Terraform to correlate objects
tracked in the state with objects declared in your configuration, and so
Terraform can understand the difference between creating or destroying an
object vs. updating or replacing an object. The resource instance address
is also what Terraform uses to describe its proposed actions to you when
presenting a plan, which could be a short address like `aws_instance.example`
or, for more complex configurations, a more lengthy address like
`module.example[1].aws_instance.example["west"]`. In all cases, these addresses
uniquely identify a particular resource instance and Terraform generally
assumes that you'll keep them constant while you change other aspects of
your configuration.

_Refactoring_ creates a special situation within Terraform where you are, in
effect, intentionally _changing_ the address of a resource instance. This could
be as simple as changing the resource name, like renaming `aws_instance.a`
to `aws_instance.b`, or it could be a more indirect difference such as moving
your resource block into a child module, or switching from a single-instance
resource to one that uses `for_each`. That might mean that the address
will change from `module.a.aws_instance.example` to
`module.b.aws_instance.example`, or from `aws_instance.example` to
`aws_instance.example["a"]`, or any combination of these.

By default, Terraform understands any change of address like this to mean that
you want to destroy the object associated with the old address and create an
entirely new object associated with the new address. When refactoring though,
we typically want to retain the existing remote object and just change it to
be bound to a different resource instance address within Terraform.

To achieve that, it's not sufficient to compare the current configuration to
the current state: we need to give Terraform some additional information
about how the configuration has changed over time, so that it can plan to
automatically migrate existing objects to their new addresses, and thus
avoid any disruption from creating and destroying objects.

The remainder of this page describes some configuration language features which
allow you to give Terraform some additional historical context about your
configuration. If you add these additional annotations as part of your
refactoring work then Terraform will propose to make matching changes to the
Terraform State as part of the next plan you create.

## Declaring that a Resource Instance has Moved

There are various changes you can make during refactoring that will change
the resource instance address that Terraform uses to identify an object:

* Changing the name in a resource declaration, such as changing
  `resource "aws_instance" "a"` to `resource "aws_instance" "b"`.
* Adding the [`count`](../meta-arguments/count.html) or the
  [`for_each`](../meta-arguments/for_each.html) meta-argument to a resource
  which didn't previously have it, or vice-versa. In that case, you might
  want to retain the previous single instance `aws_instance.example` as
  the new `aws_instance.example["a"]`, possibly while declaring entirely new
  objects with different instance keys alongside.
* Moving a resource block that was previously in your root module into a new
  child module, or vice-versa. In that case, `aws_instance.a` might
  become `module.m.aws_instance.a`.
* Moving a resource block from one child module to another, such as if you
  are breaking a larger module into smaller components. In that case,
  `module.network.aws_vpc.main` might become `module.vpc.aws_vpc.this`.

In all of these cases, and in various combinations of these, we can say that
one or more of the existing resource instances has "moved" from one address
to another.

To tell Terraform that you've moved a resource instance in one of these ways,
you can use a top-level `moved` block alongside the normal configuration block
that declares the new address of the object:

```hcl
resource "aws_instance" "b" {
  # ...
}

moved {
  from = aws_instance.a
  to   = aws_instance.b
}
```

Think of these `moved` blocks _not_ as a direct command to Terraform to take an
action, but instead as a historical record of the refactoring you've done in
your module over time.

In many situations Terraform will not take any special action in response
to a `moved` block, but there is one special situation where it's important:
if Terraform sees that the state contains an object bound to
`aws_instance.a` rather than `aws_instance.b`, the `moved` block will serve as
a hint to Terraform that it should plan to move this object from the old
address to the new address, rather than the default behavior of proposing to
destroy the `aws_instance.a` object and create a new `aws_instance.b` object.

If Terraform detects that the prior state associated with `aws_instance.a`
otherwise matches the current configuration associated with `aws_instance.b`
then Terraform will note in the plan that the object has moved but will not
propose any changes to it:

```
  # aws_instance.a has moved to aws_instance.b
    resource "aws_instance" "b" {
      # (no configuration changes)
    }
```

If you change the configuration for this resource at the same time as moving
it then Terraform will instead propose to update the existing object, while
also including an acknowledgement of the move:

```
  # aws_instance.b must be updated
  # (aws_instance.a has moved to aws_instance.b)
  ~ resource "aws_instance" "b" {
      ~ tags = {
          ~ Name = "a" -> "b"
        }
      # (other arguments unchanged)
    }
```

In both cases, if you apply the proposed plan then Terraform will move the
existing object to its new address in the state before taking any of the other
actions proposed for it, and future plans against the same configuration and
state will not mention the change of address again.

## Specifying Old and New Addresses

The `from` and `to` arguments in a `moved` block use a slightly different
syntax than other resource instance references in a module, because the author
of a module is allowed to move resources into and out of child modules that
belong to the same package of modules. Normally a module contains references
only to objects declared within the same module, but `moved` blocks are an
exception to that rule.

You can use a normal resource instance address like `aws_instance.a` or
`aws_instance.b[0]` to refer to an instance of a resource in the current
module, much the same as you would for other resource references in your module.

To refer to an instance address in a child (or descendent) module, you can
add a `module.NAME` prefix for each level of module nesting relative to the
current module. For example, if your module contains a `module "example"`
block then it might be valid to write `module.example.aws_instance.a`, if
that module happens to include a `resource "aws_instance" "a"` block.

This ability to refer directly to a resource declared in a descendent module
creates the risk of tighter coupling between caller and called module than
the Terraform language normally permits, and so Terraform will only allow
references to child modules that belong to the same package of modules as the
caller, which means module calls where the `source` argument begins with either
`./` or `../` to indicate that the caller and called module are versioned and
distributed together as a single unit. In that case, you can coordinate your
changes to both modules so that the source and destination modules will always
describe the move consistently.

When describing resource instances moved between modules, we recommend
documenting the move in the closest common ancestor module to both of the
instances. For example, if you are documenting a move from
`module.a.module.b.aws_instance.c` to `module.a.module.x.aws_instance.c`
then we'd recommend to document it in the source code for `module.a`, rather
than in the root module, so that all configurations sharing that module will
benefit from that additional record.

## Moving Whole Modules and Resources

Terraform tracks individual objects by their resource instance addresses, but
as a convenience in some common refactoring situations you can also record
that you've renamed a whole resource or an entire module with a single
`moved` block:

```hcl
resource "aws_instance" "new" {
  count = 3

  # ...
}

moved {
  from = aws_instance.old
  to   = aws_instance.new

  # This block is a shorthand for all of the following separate instance moves,
  # because aws_instance.new has count = 3 .
  #   aws_instance.old[0] -> aws_instance.new[0]
  #   aws_instance.old[1] -> aws_instance.new[1]
  #   aws_instance.old[2] -> aws_instance.new[2]
}
```

```hcl
module "new" {
  source = "./modules/new"

  # ...
}

moved {
  from = module.old
  to   = module.new

  # This block documents that every one of the resource instances declared
  # in this module and its descendent modules has moved to a different
  # module prefix, without the caller needing to be aware of or individually
  # document all of those instances, and without any changes to the called
  # module itself.
}
```

A `moved` block is valid only if the `from` and `to` addresses refer to the
same kind of object. For example, you can't document a move from a whole module
address to a single resource instance address.

A reference to an entire module is an exception to the usual rule that you
can only specify addresses in modules that are distributed together. Because
the `module.old` and `module.new` addresses in the above example don't make
any assumptions about what resource instances are declared in `./modules/new`,
Terraform itself can automatically determine all of the resource instances
under that prefix and then move corresponding objects in the state where
needed.

## Removing Old `moved` Blocks

Since `moved` blocks capture historical configuration changes and cease to be
used by Terraform once they've been incorporated into the state, if you wish
you can remove `moved` blocks from your module at a later date.

Before doing so you must be confident that there are no configurations using
your module whose states might still be using the old names, because those
callers might try to upgrade directly from their current version to your
latest version and thus never see the `moved` blocks, and so Terraform would
plan to destroy an object at the old address rather than just move it.

Conversely, there's no _requirement_ to remove old `moved` blocks from your
module, and so if you have a widely-used module with callers you don't work
with directly -- for example, if you maintain an open source module with
a large number of different users -- we recommend retaining the `moved` blocks
in your module indefinitely. Terraform will entirely ignore any `moved`
block for which the `from` address is not bound to any object in the state.
Removing a `moved` block in a later version of a shared module is a breaking
change to that module.

In order to reduce the risk of ambiguigous situations, Terraform will _not_
allow a situation where the `from` address refers to an instance that is
currently declared in the same configuration, and so retaining historical
`moved` blocks will also block you from declaring new resource instances with
addresses that were previously moved from. We recommend using a different
address for your new resource instances where possible. If that isn't possible
then you can remove a `moved` block to un-reserve the address, but all of
the caveats related to callers upgrading will still apply.

## Chained and Nested Moves

For modules that are maintained over a long period of time, you're increasingly
likely to encounter situations where old and new moves will interact, and
where a caller of the module could be at any point in a series of different
moves made in successive module versions.

Terraform will accept and honor situations where the `to` address of one
`moved` block matches the `from` address of another, both within the same
module and between modules:

```hcl
resource "aws_vpc" "c" {

}

moved {
  from = aws_vpc.a
  to   = aws_vpc.b
}

moved {
  from = aws_vpc.b
  to   = aws_vpc.c
}
```

This situation is called _chained moves_, and if necessary Terraform will
notice that an existing state still has an `aws_vpc.a` object and _not_ either
`aws_vpc.b` or `aws_vpc.c` and thus plan to move the `aws_vpc.a` object
directly to `aws_vpc.c` in order to honor both of the recorded moves.

Another possible interaction is when a parent module records a move of a
child module that also has a move recorded within it, creating a possible
sequence like this:

* `module.a.aws_instance.this`
* `module.b.aws_instance.this`
* `module.b.aws_instance.that`

This situation is called _nested moves_, and again Terraform will detect and
handle all three of the possible prior states in such a situation. For example,
if the state includes an object for `module.a.aws_instance.this` and _doesn't_
include an object for either `module.b.aws_instance.this` or
`module.b.aws_instance.that` then Terraform will plan to move
the `module.a.aws_instance.this` object directly to
`module.b.aws_instance.that`.

Chained moves and nested moves can also interact together, and so we can
talk more generally about _move sequences_, which are a sequence of addresses
that an object should move along until it becomes associated with a resource
instance still declared in the configuration.

If a move sequence ends at an address that _doesn't_ match a resource instance
currently declared in the configuration then Terraform will propose to
destroy any object found at any address in the sequence.

In very rare situations it might be the case that multiple addresses in a
move sequence will all have objects recorded in the state. This situation
is reachable only if the `moved` blocks in the current configuration are _not_
an accurate record of earlier configuration changes. If Terraform detects that
case then it will resolve it by retaining the object that is closest to the end
of the move sequence and proposing to destroy the others.

## Deprecated Resource Types

Although `moved` blocks are primarily for documenting changes you've made
yourself as a module author, they can also serve a secondary purpose as part
of the deprecation cycle for a resource type defined in a provider.

Terraform will not normally allow you to declare that you've moved between
two addresses that have different resource type names, because each resource
type has its own separate schema and thus the representation of a bound object
in the Terraform state would not be compatible between the two resource types.

However, as a special case a provider can declare that a particular resource
type name is a deprecated alias for another resource type name. In doing so,
the provider developer promises that the two resource types have
sufficiently-compatible schemas that it will be possible to successfully
migrate from the old type to the new type, and so Terraform will then make
an exception to the usual rule and allow you to declare a move between the
old and new type name:

```hcl
resource "happycloud_replacement" "example" {
  # ...
}

moved {
  from = happycloud_deprecated.example
  to   = happycloud_replacement.example
}
```

A provider plugin will typically emit a warning when you use a deprecated
resource type. You can silence that warning by changing the existing `resource`
block to use the new name _and_ adding a `moved` block like the one shown above
to record that your intent was to move the existing object to the new address.

When creating the first plan after creating this change, Terraform will report
in the usual way that the resource instance has moved, possibly also including
any updates to that object proposed by the provider based on your configuration.
You can apply that plan to incorporate the change of address into your
Terraform state.
