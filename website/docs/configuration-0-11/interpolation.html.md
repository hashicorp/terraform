---
layout: "docs"
page_title: "Interpolation Syntax - 0.11 Configuration Language"
sidebar_current: "docs-conf-old-interpolation"
description: |-
  Embedded within strings in Terraform, whether you're using the Terraform syntax or JSON syntax, you can interpolate other values into strings. These interpolations are wrapped in `${}`, such as `${var.foo}`.
---

# Interpolation Syntax

-> **Note:** This page is about Terraform 0.11 and earlier. For Terraform 0.12
and later, see
[Configuration Language: Expressions](../configuration/expressions.html) and
[Configuration Language: Functions](../configuration/functions.html).

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

The syntax is `"${var.LIST}"`. For example, `"${var.subnets}"`
would get the value of the `subnets` list, as a list. You can also
return list elements by index: `${var.subnets[idx]}`.

#### Attributes of your own resource

The syntax is `self.ATTRIBUTE`. For example `${self.private_ip}`
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

#### Attributes of a data source

The syntax is `data.TYPE.NAME.ATTRIBUTE`. For example. `${data.aws_ami.ubuntu.id}` will interpolate the `id` attribute from the `aws_ami` [data source](./data-sources.html) named `ubuntu`. If the data source has a `count`
attribute set, you can access individual attributes with a zero-based
index, such as `${data.aws_subnet.example.0.cidr_block}`. You can also use the splat
syntax to get a list of all the attributes: `${data.aws_subnet.example.*.cidr_block}`.

#### Outputs from a module

The syntax is `MODULE.NAME.OUTPUT`. For example `${module.foo.bar}` will
interpolate the `bar` output from the `foo`
[module](/docs/modules/index.html).

#### Count information

The syntax is `count.FIELD`. For example, `${count.index}` will
interpolate the current index in a multi-count resource. For more
information on `count`, see the [resource configuration
page](./resources.html).

#### Path information

The syntax is `path.TYPE`. TYPE can be `cwd`, `module`, or `root`.
`cwd` will interpolate the current working directory. `module` will
interpolate the path to the current module. `root` will interpolate the
path of the root module.  In general, you probably want the
`path.module` variable.

#### Terraform meta information

The syntax is `terraform.FIELD`. This variable type contains metadata about
the currently executing Terraform run. FIELD can currently only be `env` to
reference the currently active [state environment](/docs/state/environments.html).

## Conditionals

Interpolations may contain conditionals to branch on the final value.

```hcl
resource "aws_instance" "web" {
  subnet = "${var.env == "production" ? var.prod_subnet : var.dev_subnet}"
}
```

The conditional syntax is the well-known ternary operation:

```text
CONDITION ? TRUEVAL : FALSEVAL
```

The condition can be any valid interpolation syntax, such as variable
access, a function call, or even another conditional. The true and false
value can also be any valid interpolation syntax. The returned types by
the true and false side must be the same.

The supported operators are:

  * Equality: `==` and `!=`
  * Numerical comparison: `>`, `<`, `>=`, `<=`
  * Boolean logic: `&&`, `||`, unary `!`

A common use case for conditionals is to enable/disable a resource by
conditionally setting the count:

```hcl
resource "aws_instance" "vpn" {
  count = "${var.something ? 1 : 0}"
}
```

In the example above, the "vpn" resource will only be included if
"var.something" evaluates to true. Otherwise, the VPN resource will
not be created at all.

## Built-in Functions

Terraform ships with built-in functions. Functions are called with the
syntax `name(arg, arg2, ...)`. For example, to read a file:
`${file("path.txt")}`.

~> **NOTE**: Proper escaping is required for JSON field values containing quotes
(`"`) such as `environment` values. If directly setting the JSON, they should be
escaped as `\"` in the JSON,  e.g. `"value": "I \"love\" escaped quotes"`. If
using a Terraform variable value, they should be escaped as `\\\"` in the
variable, e.g. `value = "I \\\"love\\\" escaped quotes"` in the variable and
`"value": "${var.myvariable}"` in the JSON.

### Supported built-in functions

The supported built-in functions are:

  * `abs(float)` - Returns the absolute value of a given float.
    Example: `abs(1)` returns `1`, and `abs(-1)` would also return `1`,
    whereas `abs(-3.14)` would return `3.14`. See also the `signum` function.

  * `basename(path)` - Returns the last element of a path.

  * `base64decode(string)` - Given a base64-encoded string, decodes it and
    returns the original string.

  * `base64encode(string)` - Returns a base64-encoded representation of the
    given string.

  * `base64gzip(string)` - Compresses the given string with gzip and then
    encodes the result to base64. This can be used with certain resource
    arguments that allow binary data to be passed with base64 encoding, since
    Terraform strings are required to be valid UTF-8.

  * `base64sha256(string)` - Returns a base64-encoded representation of raw
    SHA-256 sum of the given string.
    **This is not equivalent** of `base64encode(sha256(string))`
    since `sha256()` returns hexadecimal representation.

  * `base64sha512(string)` - Returns a base64-encoded representation of raw
    SHA-512 sum of the given string.
    **This is not equivalent** of `base64encode(sha512(string))`
    since `sha512()` returns hexadecimal representation.

  * `bcrypt(password, cost)` - Returns the Blowfish encrypted hash of the string 
    at the given cost. A default `cost` of 10 will be used if not provided.

  * `ceil(float)` - Returns the least integer value greater than or equal
      to the argument.

  * `chomp(string)` - Removes trailing newlines from the given string.

  * `chunklist(list, size)` - Returns the `list` items chunked by `size`.
    Examples:
    * `chunklist(aws_subnet.foo.*.id, 1)`: will outputs `[["id1"], ["id2"], ["id3"]]`
    * `chunklist(var.list_of_strings, 2)`: will outputs `[["id1", "id2"], ["id3", "id4"], ["id5"]]`

  * `cidrhost(iprange, hostnum)` - Takes an IP address range in CIDR notation
    and creates an IP address with the given host number. If given host
    number is negative, the count starts from the end of the range.
    For example, `cidrhost("10.0.0.0/8", 2)` returns `10.0.0.2` and
    `cidrhost("10.0.0.0/8", -2)` returns `10.255.255.254`.

  * `cidrnetmask(iprange)` - Takes an IP address range in CIDR notation
    and returns the address-formatted subnet mask format that some
    systems expect for IPv4 interfaces. For example,
    `cidrnetmask("10.0.0.0/8")` returns `255.0.0.0`. Not applicable
    to IPv6 networks since CIDR notation is the only valid notation for
    IPv6.

  * `cidrsubnet(iprange, newbits, netnum)` - Takes an IP address range in
    CIDR notation (like `10.0.0.0/8`) and extends its prefix to include an
    additional subnet number. For example,
    `cidrsubnet("10.0.0.0/8", 8, 2)` returns `10.2.0.0/16`;
    `cidrsubnet("2607:f298:6051:516c::/64", 8, 2)` returns
    `2607:f298:6051:516c:200::/72`.

  * `coalesce(string1, string2, ...)` - Returns the first non-empty value from
    the given arguments. At least two arguments must be provided.

  * `coalescelist(list1, list2, ...)` - Returns the first non-empty list from
    the given arguments. At least two arguments must be provided.

  * `compact(list)` - Removes empty string elements from a list. This can be
     useful in some cases, for example when passing joined lists as module
     variables or when parsing module outputs.
     Example: `compact(module.my_asg.load_balancer_names)`

  * `concat(list1, list2, ...)` - Combines two or more lists into a single list.
     Example: `concat(aws_instance.db.*.tags.Name, aws_instance.web.*.tags.Name)`

  * `contains(list, element)` - Returns *true* if a list contains the given element
     and returns *false* otherwise. Examples: `contains(var.list_of_strings, "an_element")`

  * `dirname(path)` - Returns all but the last element of path, typically the path's directory.

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
      [Path variables](#path-information) can be used to reference paths relative
      to other base locations. For example, when using `file()` from inside a
      module, you generally want to make the path relative to the module base,
      like this: `file("${path.module}/file")`.

  * `floor(float)` - Returns the greatest integer value less than or equal to
      the argument.

  * `flatten(list of lists)` - Flattens lists of lists down to a flat list of
       primitive values, eliminating any nested lists recursively. Examples:
       * `flatten(data.github_user.user.*.gpg_keys)`

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

  * `indent(numspaces, string)` - Prepends the specified number of spaces to all but the first
      line of the given multi-line string. May be useful when inserting a multi-line string
      into an already-indented context. The first line is not indented, to allow for the
      indented string to be placed after some sort of already-indented preamble.
      Example: `"    \"items\": ${ indent(4, "[\n    \"item1\"\n]") },"`

  * `index(list, elem)` - Finds the index of a given element in a list.
      This function only works on flat lists.
      Example: `index(aws_instance.foo.*.tags.Name, "foo-test")`

  * `join(delim, list)` - Joins the list with the delimiter for a resultant string.
      This function works only on flat lists.
      Examples:
      * `join(",", aws_instance.foo.*.id)`
      * `join(",", var.ami_list)`

  * `jsonencode(value)` - Returns a JSON-encoded representation of the given
      value, which can contain arbitrarily-nested lists and maps. Note that if
      the value is a string then its value will be placed in quotes.

  * `keys(map)` - Returns a lexically sorted list of the map keys.

  * `length(list)` - Returns the number of members in a given list or map, or the number of characters in a given string.
      * `${length(split(",", "a,b,c"))}` = 3
      * `${length("a,b,c")}` = 5
      * `${length(map("key", "val"))}` = 1

  * `list(items, ...)` - Returns a list consisting of the arguments to the function.
      This function provides a way of representing list literals in interpolation.
      * `${list("a", "b", "c")}` returns a list of `"a", "b", "c"`.
      * `${list()}` returns an empty list.

  * `log(x, base)` - Returns the logarithm of `x`.

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

  * `matchkeys(values, keys, searchset)` - For two lists `values` and `keys` of
      equal length, returns all elements from `values` where the corresponding
      element from `keys` exists in the `searchset` list.  E.g.
      `matchkeys(aws_instance.example.*.id,
      aws_instance.example.*.availability_zone, list("us-west-2a"))` will return a
      list of the instance IDs of the `aws_instance.example` instances in
      `"us-west-2a"`. No match will result in empty list. Items of `keys` are
      processed sequentially, so the order of returned `values` is preserved.

  * `max(float1, float2, ...)` - Returns the largest of the floats.

  * `merge(map1, map2, ...)` - Returns the union of 2 or more maps. The maps
	are consumed in the order provided, and duplicate keys overwrite previous
	entries.
	* `${merge(map("a", "b"), map("c", "d"))}` returns `{"a": "b", "c": "d"}`

  * `min(float1, float2, ...)` - Returns the smallest of the floats.

  * `md5(string)` - Returns a (conventional) hexadecimal representation of the
    MD5 hash of the given string.

  * `pathexpand(string)` - Returns a filepath string with `~` expanded to the home directory. Note:
    This will create a plan diff between two different hosts, unless the filepaths are the same.

  * `pow(x, y)` - Returns the base `x` of exponential `y` as a float.

    Example:
    * `${pow(3,2)}` = 9
    * `${pow(4,0)}` = 1

  * `replace(string, search, replace)` - Does a search and replace on the
      given string. All instances of `search` are replaced with the value
      of `replace`. If `search` is wrapped in forward slashes, it is treated
      as a regular expression. If using a regular expression, `replace`
      can reference subcaptures in the regular expression by using `$n` where
      `n` is the index or name of the subcapture. If using a regular expression,
      the syntax conforms to the [re2 regular expression syntax](https://github.com/google/re2/wiki/Syntax).

  * `rsadecrypt(string, key)` - Decrypts `string` using RSA. The padding scheme
    PKCS #1 v1.5 is used. The `string` must be base64-encoded. `key` must be an
    RSA private key in PEM format. You may use `file()` to load it from a file.

  * `sha1(string)` - Returns a (conventional) hexadecimal representation of the
    SHA-1 hash of the given string.
    Example: `"${sha1("${aws_vpc.default.tags.customer}-s3-bucket")}"`

  * `sha256(string)` - Returns a (conventional) hexadecimal representation of the
    SHA-256 hash of the given string.
    Example: `"${sha256("${aws_vpc.default.tags.customer}-s3-bucket")}"`

  * `sha512(string)` - Returns a (conventional) hexadecimal representation of the
    SHA-512 hash of the given string.
    Example: `"${sha512("${aws_vpc.default.tags.customer}-s3-bucket")}"`

  * `signum(integer)` - Returns `-1` for negative numbers, `0` for `0` and `1` for positive numbers.
      This function is useful when you need to set a value for the first resource and
      a different value for the rest of the resources.
      Example: `element(split(",", var.r53_failover_policy), signum(count.index))`
      where the 0th index points to `PRIMARY` and 1st to `FAILOVER`

  * `slice(list, from, to)` - Returns the portion of `list` between `from` (inclusive) and `to` (exclusive).
      Example: `slice(var.list_of_strings, 0, length(var.list_of_strings) - 1)`

  * `sort(list)` - Returns a lexographically sorted list of the strings contained in
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

  * `substr(string, offset, length)` - Extracts a substring from the input string. A negative offset is interpreted as being equivalent to a positive offset measured backwards from the end of the string. A length of `-1` is interpreted as meaning "until the end of the string".

  * `timestamp()` - Returns a UTC timestamp string in RFC 3339 format. This string will change with every
   invocation of the function, so in order to prevent diffs on every plan & apply, it must be used with the
   [`ignore_changes`](./resources.html#ignore-changes) lifecycle attribute.

  * `timeadd(time, duration)` - Returns a UTC timestamp string corresponding to adding a given `duration` to `time` in RFC 3339 format.      
    For example, `timeadd("2017-11-22T00:00:00Z", "10m")` produces a value `"2017-11-22T00:10:00Z"`. 
    
  * `title(string)` - Returns a copy of the string with the first characters of all the words capitalized.

  * `transpose(map)` - Swaps the keys and list values in a map of lists of strings. For example, transpose(map("a", list("1", "2"), "b", list("2", "3")) produces a value equivalent to map("1", list("a"), "2", list("a", "b"), "3", list("b")).

  * `trimspace(string)` - Returns a copy of the string with all leading and trailing white spaces removed.

  * `upper(string)` - Returns a copy of the string with all Unicode letters mapped to their upper case.

  * `urlencode(string)` - Returns an URL-safe copy of the string.

  * `uuid()` - Returns a random UUID string. This string will change with every invocation of the function, so in order to prevent diffs on every plan & apply, it must be used with the [`ignore_changes`](./resources.html#ignore-changes) lifecycle attribute.

  * `values(map)` - Returns a list of the map values, in the order of the keys
    returned by the `keys` function. This function only works on flat maps and
    will return an error for maps that include nested lists or maps.

  * `zipmap(list, list)` - Creates a map from a list of keys and a list of
      values. The keys must all be of type string, and the length of the lists
      must be the same.
      For example, to output a mapping of AWS IAM user names to the fingerprint
      of the key used to encrypt their initial password, you might use:
      `zipmap(aws_iam_user.users.*.name, aws_iam_user_login_profile.users.*.key_fingerprint)`.

The hashing functions `base64sha256`, `base64sha512`, `md5`, `sha1`, `sha256`,
and `sha512` all have variants with a `file` prefix, like `filesha1`, which
interpret their first argument as a path to a file on disk rather than as a
literal string. This allows safely creating hashes of binary files that might
otherwise be corrupted in memory if loaded into Terraform strings (which are
assumed to be UTF-8). `filesha1(filename)` is equivalent to `sha1(file(filename))`
in Terraform 0.11 and earlier, but the latter will fail for binary files in
Terraform 0.12 and later.

## Templates

Long strings can be managed using templates.
[Templates](/docs/providers/template/index.html) are
[data-sources](./data-sources.html) defined by a
filename and some variables to use during interpolation. They have a
computed `rendered` attribute containing the result.

A template data source looks like:

```hcl
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

Note that the double dollar signs (`$$`) are needed in inline templates. Otherwise Terraform will return an error.

You may use any of the built-in functions in your template. For more
details on template usage, please see the
[template_file documentation](/docs/providers/template/d/file.html).

### Using Templates with Count

Here is an example that combines the capabilities of templates with the interpolation
from `count` to give us a parameterized template, unique to each resource instance:

```hcl
variable "hostnames" {
  default = {
    "0" = "example1.org"
    "1" = "example2.net"
  }
}

data "template_file" "web_init" {
  # Render the template once for each instance
  count    = "${length(var.hostnames)}"
  template = "${file("templates/web_init.tpl")}"
  vars {
    # count.index tells us the index of the instance we are rendering
    hostname = "${var.hostnames[count.index]}"
  }
}

resource "aws_instance" "web" {
  # Create one instance for each hostname
  count     = "${length(var.hostnames)}"

  # Pass each instance its corresponding template_file
  user_data = "${data.template_file.web_init.*.rendered[count.index]}"
}
```

With this, we will build a list of `template_file.web_init` data resources
which we can use in combination with our list of `aws_instance.web` resources.

## Math

Simple math can be performed in interpolations:

```hcl
variable "count" {
  default = 2
}

resource "aws_instance" "web" {
  # ...

  count = "${var.count}"

  # Tag the instance with a counter starting at 1, ie. web-001
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

```text
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
