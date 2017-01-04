---
layout: "docs"
page_title: "Interpolation Syntax"
sidebar_current: "docs-config-interpolation"
description: |-
  Embedded within strings in Terraform, whether you're using the Terraform syntax or JSON syntax, you can interpolate other values into strings. These interpolations are wrapped in `${}`, such as `${var.foo}`.
---

# Interpolation Syntax

Embedded within strings in Terraform, whether you're using the
Terraform syntax or JSON syntax, you can interpolate other values. These
interpolations are wrapped in `${}`, such as `${var.foo}`.

The interpolation syntax is powerful and allows you to reference
variables, attributes of resources, call functions, etc.

You can perform [simple math](#math) in interpolations, allowing
you to write expressions such as `${count.index + 1}`. And you can
also use [conditionals](#conditionals) to determine a value based
on some logic.

You can escape interpolation with double dollar signs: `$${foo}`
will be rendered as a literal `${foo}`.

## Available Variables

There are a variety of available variable references you can use.

#### User string variables

Use the `var.` prefix followed by the variable name. For example,
`${var.foo}` will interpolate the `foo` variable value.

#### User map variables

The syntax is `var.MAP["KEY"]`. For example, `${var.amis["us-east-1"]}`
would get the value of the `us-east-1` key within the `amis` map
variable.

#### User list variables

The syntax is `["${var.LIST}"]`. For example, `["${var.subnets}"]`
would get the value of the `subnets` list, as a list. You can also
return list elements by index: `${var.subnets[idx]}`.

#### Attributes of your own resource

The syntax is `self.ATTRIBUTE`. For example `${self.private_ip_address}`
will interpolate that resource's private IP address.

-> **Note**: The `self.ATTRIBUTE` syntax is only allowed and valid within
provisioners.

#### Attributes of other resources

The syntax is `TYPE.NAME.ATTRIBUTE`. For example,
`${aws_instance.web.id}` will interpolate the ID attribute from the
`aws_instance` resource named `web`. If the resource has a `count`
attribute set, you can access individual attributes with a zero-based
index, such as `${aws_instance.web.0.id}`. You can also use the splat
syntax to get a list of all the attributes: `${aws_instance.web.*.id}`.
This is documented in more detail in the [resource configuration
page](/docs/configuration/resources.html).

#### Outputs from a module

The syntax is `MODULE.NAME.OUTPUT`. For example `${module.foo.bar}` will
interpolate the `bar` output from the `foo`
[module](/docs/modules/index.html).

#### Count information

The syntax is `count.FIELD`. For example, `${count.index}` will
interpolate the current index in a multi-count resource. For more
information on `count`, see the [resource configuration
page](/docs/configuration/resources.html).

<a id="path-variables"></a>

#### Path information

The syntax is `path.TYPE`. TYPE can be `cwd`, `module`, or `root`.
`cwd` will interpolate the current working directory. `module` will
interpolate the path to the current module. `root` will interpolate the
path of the root module.  In general, you probably want the
`path.module` variable.

<a id="conditionals"></a>
## Conditionals

Interpolations may contain conditionals to branch on the final value.

```
resource "aws_instance" "web" {
  subnet = "${var.env == "production" ? var.prod_subnet : var.dev_subnet}"
}
```

The conditional syntax is the well-known ternary operation:

    CONDITION ? TRUEVAL : FALSEVAL

The condition can be any valid interpolation syntax, such as variable
access, a function call, or even another conditional. The true and false
value can also be any valid interpolation syntax. The returned types by
the true and false side must be the same.

The support operators are:

  * Equality: `==` and `!=`
  * Numerical comparison: `>`, `<`, `>=`, `<=`
  * Boolean logic: `&&`, `||`, unary `!`

A common use case for conditionals is to enable/disable a resource by
conditionally setting the count:

```
resource "aws_instance" "vpn" {
  count = "${var.something ? 1 : 0}"
}
```

In the example above, the "vpn" resource will only be included if
"var.something" evaluates to true. Otherwise, the VPN resource will
not be created at all.

<a id="functions"></a>
## Built-in Functions

Terraform ships with built-in functions. Functions are called with the
syntax `name(arg, arg2, ...)`. For example, to read a file:
`${file("path.txt")}`.

### Supported built-in functions

The supported built-in functions are:

  * `base64decode(string)` - Given a base64-encoded string, decodes it and
    returns the original string.

  * `base64encode(string)` - Returns a base64-encoded representation of the
    given string.

  * `base64sha256(string)` - Returns a base64-encoded representation of raw
    SHA-256 sum of the given string.
    **This is not equivalent** of `base64encode(sha256(string))`
    since `sha256()` returns hexadecimal representation.

  * `ceil(float)` - Returns the least integer value greater than or equal
      to the argument.

  * `cidrhost(iprange, hostnum)` - Takes an IP address range in CIDR notation
    and creates an IP address with the given host number. For example,
    `cidrhost("10.0.0.0/8", 2)` returns `10.0.0.2`.

  * `cidrnetmask(iprange)` - Takes an IP address range in CIDR notation
    and returns the address-formatted subnet mask format that some
    systems expect for IPv4 interfaces. For example,
    `cidrnetmask("10.0.0.0/8")` returns `255.0.0.0`. Not applicable
    to IPv6 networks since CIDR notation is the only valid notation for
    IPv4.

  * `cidrsubnet(iprange, newbits, netnum)` - Takes an IP address range in
    CIDR notation (like `10.0.0.0/8`) and extends its prefix to include an
    additional subnet number. For example,
    `cidrsubnet("10.0.0.0/8", 8, 2)` returns `10.2.0.0/16`;
    `cidrsubnet("2607:f298:6051:516c::/64", 8, 2)` returns
    `2607:f298:6051:516c:200::/72`.

  * `coalesce(string1, string2, ...)` - Returns the first non-empty value from
    the given arguments. At least two arguments must be provided.

  * `compact(list)` - Removes empty string elements from a list. This can be
     useful in some cases, for example when passing joined lists as module
     variables or when parsing module outputs.
     Example: `compact(module.my_asg.load_balancer_names)`

  * `concat(list1, list2, ...)` - Combines two or more lists into a single list.
     Example: `concat(aws_instance.db.*.tags.Name, aws_instance.web.*.tags.Name)`

  * `distinct(list)` - Removes duplicate items from a list. Keeps the first
     occurrence of each element, and removes subsequent occurrences. This
     function is only valid for flat lists. Example: `distinct(var.usernames)`

  * `element(list, index)` - Returns a single element from a list
      at the given index. If the index is greater than the number of
      elements, this function will wrap using a standard mod algorithm.
      This function only works on flat lists. Examples:
      * `element(aws_subnet.foo.*.id, count.index)`
      * `element(var.list_of_strings, 2)`

  * `file(path)` - Reads the contents of a file into the string. Variables
      in this file are _not_ interpolated. The contents of the file are
      read as-is. The `path` is interpreted relative to the working directory.
      [Path variables](#path-variables) can be used to reference paths relative
      to other base locations. For example, when using `file()` from inside a
      module, you generally want to make the path relative to the module base,
      like this: `file("${path.module}/file")`.

  * `floor(float)` - Returns the greatest integer value less than or equal to
      the argument.

  * `format(format, args, ...)` - Formats a string according to the given
      format. The syntax for the format is standard `sprintf` syntax.
      Good documentation for the syntax can be [found here](https://golang.org/pkg/fmt/).
      Example to zero-prefix a count, used commonly for naming servers:
      `format("web-%03d", count.index + 1)`.

  * `formatlist(format, args, ...)` - Formats each element of a list
      according to the given format, similarly to `format`, and returns a list.
      Non-list arguments are repeated for each list element.
      For example, to convert a list of DNS addresses to a list of URLs, you might use:
      `formatlist("https://%s:%s/", aws_instance.foo.*.public_dns, var.port)`.
      If multiple args are lists, and they have the same number of elements, then the formatting is applied to the elements of the lists in parallel.
      Example:
      `formatlist("instance %v has private ip %v", aws_instance.foo.*.id, aws_instance.foo.*.private_ip)`.
      Passing lists with different lengths to formatlist results in an error.

  * `index(list, elem)` - Finds the index of a given element in a list.
      This function only works on flat lists.
      Example: `index(aws_instance.foo.*.tags.Name, "foo-test")`

  * `join(delim, list)` - Joins the list with the delimiter for a resultant string.
      This function works only on flat lists.
      Examples:
      * `join(",", aws_instance.foo.*.id)`
      * `join(",", var.ami_list)`

  * `jsonencode(item)` - Returns a JSON-encoded representation of the given
    item, which may be a string, list of strings, or map from string to string.
    Note that if the item is a string, the return value includes the double
    quotes.

  * `keys(map)` - Returns a lexically sorted list of the map keys.

  * `length(list)` - Returns the number of members in a given list or map, or the number of characters in a given string.
      * `${length(split(",", "a,b,c"))}` = 3
      * `${length("a,b,c")}` = 5
      * `${length(map("key", "val"))}` = 1

  * `list(items, ...)` - Returns a list consisting of the arguments to the function.
      This function provides a way of representing list literals in interpolation.
      * `${list("a", "b", "c")}` returns a list of `"a", "b", "c"`.
      * `${list()}` returns an empty list.

  * `lookup(map, key, [default])` - Performs a dynamic lookup into a map
      variable. The `map` parameter should be another variable, such
      as `var.amis`. If `key` does not exist in `map`, the interpolation will
      fail unless you specify a third argument, `default`, which should be a
      string value to return if no `key` is found in `map`. This function
      only works on flat maps and will return an error for maps that
      include nested lists or maps.

  * `lower(string)` - Returns a copy of the string with all Unicode letters mapped to their lower case.

  * `map(key, value, ...)` - Returns a map consisting of the key/value pairs
    specified as arguments. Every odd argument must be a string key, and every
    even argument must have the same type as the other values specified.
    Duplicate keys are not allowed. Examples:
    * `map("hello", "world")`
    * `map("us-east", list("a", "b", "c"), "us-west", list("b", "c", "d"))`

  * `max(float1, float2, ...)` - Returns the largest of the floats.

  * `merge(map1, map2, ...)` - Returns the union of 2 or more maps. The maps
	are consumed in the order provided, and duplicate keys overwrite previous
	entries.
	* `${merge(map("a", "b"), map("c", "d"))}` returns `{"a": "b", "c": "d"}`

  * `min(float1, float2, ...)` - Returns the smallest of the floats.

  * `md5(string)` - Returns a (conventional) hexadecimal representation of the
    MD5 hash of the given string.

  * `replace(string, search, replace)` - Does a search and replace on the
      given string. All instances of `search` are replaced with the value
      of `replace`. If `search` is wrapped in forward slashes, it is treated
      as a regular expression. If using a regular expression, `replace`
      can reference subcaptures in the regular expression by using `$n` where
      `n` is the index or name of the subcapture. If using a regular expression,
      the syntax conforms to the [re2 regular expression syntax](https://code.google.com/p/re2/wiki/Syntax).

  * `sha1(string)` - Returns a (conventional) hexadecimal representation of the
    SHA-1 hash of the given string.
    Example: `"${sha1("${aws_vpc.default.tags.customer}-s3-bucket")}"`

  * `sha256(string)` - Returns a (conventional) hexadecimal representation of the
    SHA-256 hash of the given string.
    Example: `"${sha256("${aws_vpc.default.tags.customer}-s3-bucket")}"`

  * `signum(int)` - Returns `-1` for negative numbers, `0` for `0` and `1` for positive numbers.
      This function is useful when you need to set a value for the first resource and
      a different value for the rest of the resources.
      Example: `element(split(",", var.r53_failover_policy), signum(count.index))`
      where the 0th index points to `PRIMARY` and 1st to `FAILOVER`

  * `sort(list)` - Returns a lexicographically sorted list of the strings contained in
      the list passed as an argument. Sort may only be used with lists which contain only
      strings.
      Examples: `sort(aws_instance.foo.*.id)`, `sort(var.list_of_strings)`

  * `split(delim, string)` - Splits the string previously created by `join`
      back into a list. This is useful for pushing lists through module
      outputs since they currently only support string values. Depending on the
      use, the string this is being performed within may need to be wrapped
      in brackets to indicate that the output is actually a list, e.g.
      `a_resource_param = ["${split(",", var.CSV_STRING)}"]`.
      Example: `split(",", module.amod.server_ids)`

  * `timestamp()` - Returns a UTC timestamp string in RFC 3339 format. This string will change with every
   invocation of the function, so in order to prevent diffs on every plan & apply, it must be used with the
   [`ignore_changes`](/docs/configuration/resources.html#ignore-changes) lifecycle attribute.

  * `title(string)` - Returns a copy of the string with the first characters of all the words capitalized.

  * `trimspace(string)` - Returns a copy of the string with all leading and trailing white spaces removed.

  * `upper(string)` - Returns a copy of the string with all Unicode letters mapped to their upper case.

  * `uuid()` - Returns a UUID string in RFC 4122 v4 format. This string will change with every invocation of the function, so in order to prevent diffs on every plan & apply, it must be used with the [`ignore_changes`](/docs/configuration/resources.html#ignore-changes) lifecycle attribute.

  * `values(map)` - Returns a list of the map values, in the order of the keys
    returned by the `keys` function. This function only works on flat maps and
    will return an error for maps that include nested lists or maps.

  * `zipmap(list, list)` - Creates a map from a list of keys and a list of
      values. The keys must all be of type string, and the length of the lists
      must be the same.
      For example, to output a mapping of AWS IAM user names to the fingerprint
      of the key used to encrypt their initial password, you might use:
      `zipmap(aws_iam_user.users.*.name, aws_iam_user_login_profile.users.*.key_fingerprint)`.

<a id="templates"></a>
## Templates

Long strings can be managed using templates.
[Templates](/docs/providers/template/index.html) are
[data-sources](/docs/configuration/data-sources.html) defined by a
filename and some variables to use during interpolation. They have a
computed `rendered` attribute containing the result.

A template data source looks like:

```
data "template_file" "example" {
  template = "$${hello} $${world}!"
  vars {
    hello = "goodnight"
    world = "moon"
  }
}

output "rendered" {
  value = "${data.template_file.example.rendered}"
}
```

Then the rendered value would be `goodnight moon!`.

You may use any of the built-in functions in your template. For more
details on template usage, please see the
[template_file documentation](/docs/providers/template/d/file.html).

### Using Templates with Count

Here is an example that combines the capabilities of templates with the interpolation
from `count` to give us a parameterized template, unique to each resource instance:

```
variable "count" {
  default = 2
}

variable "hostnames" {
  default = {
    "0" = "example1.org"
    "1" = "example2.net"
  }
}

data "template_file" "web_init" {
  // here we expand multiple template_files - the same number as we have instances
  count    = "${var.count}"
  template = "${file("templates/web_init.tpl")}"
  vars {
    // that gives us access to use count.index to do the lookup
    hostname = "${lookup(var.hostnames, count.index)}"
  }
}

resource "aws_instance" "web" {
  // ...
  count = "${var.count}"
  // here we link each web instance to the proper template_file
  user_data = "${element(data.template_file.web_init.*.rendered, count.index)}"
}
```

With this, we will build a list of `template_file.web_init` data sources which we can
use in combination with our list of `aws_instance.web` resources.

<a id="math"></a>
## Math

Simple math can be performed in interpolations:

```
variable "count" {
  default = 2
}

resource "aws_instance" "web" {
  // ...
  count = "${var.count}"

  // tag the instance with a counter starting at 1, ie. web-001
  tags {
    Name = "${format("web-%03d", count.index + 1)}"
  }
}
```

The supported operations are:

- *Add* (`+`), *Subtract* (`-`), *Multiply* (`*`), and *Divide* (`/`) for **float** types
- *Add* (`+`), *Subtract* (`-`), *Multiply* (`*`), *Divide* (`/`), and *Modulo* (`%`) for **integer** types

Operator precedences is the standard mathematical order of operations:
*Multiply* (`*`), *Divide* (`/`), and *Modulo* (`%`) have precedence over
*Add* (`+`) and *Subtract* (`-`). Parenthesis can be used to force ordering.

```
"${2 * 4 + 3 * 3}" # computes to 17
"${3 * 3 + 2 * 4}" # computes to 17
"${2 * (4 + 3) * 3}" # computes to 42
```

You can use the [terraform console](/docs/commands/console.html) command to
try the math operations.

-> **Note:** Since Terraform allows hyphens in resource and variable names,
it's best to use spaces between math operators to prevent confusion or unexpected
behavior. For example, `${var.instance-count - 1}` will subtract **1** from the
`instance-count` variable value, while `${var.instance-count-1}` will interpolate
the `instance-count-1` variable value.
