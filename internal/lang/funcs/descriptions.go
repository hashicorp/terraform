package funcs

import "github.com/zclconf/go-cty/cty/function"

type descriptionEntry struct {
	// Description is a description for the function.
	Description string

	// ParamDescription argument must match the number of parameters of the
	// function. If the function has a VarParam then that counts as one
	// parameter. The given descriptions will be assigned in order starting
	// with the positional arguments in their declared order, followed by the
	// variadic parameter if any.
	ParamDescription []string
}

// descriptionList is a consolidated list containing all descriptions for all
// functions available within Terraform. A functions description should point
// to the matching entry in this list.
//
// We keep this as a single list, so we can quickly review descriptions within
// a single file and copy the whole list to other projects, like
// terraform-schema.
var descriptionList = map[string]descriptionEntry{
	"abs": {
		Description:      "`abs` returns the absolute value of the given number. In other words, if the number is zero or positive then it is returned as-is, but if it is negative then it is multiplied by -1 to make it positive before returning it.",
		ParamDescription: []string{""},
	},
	"abspath": {
		Description:      "`abspath` takes a string containing a filesystem path and converts it to an absolute path. That is, if the path is not absolute, it will be joined with the current working directory.",
		ParamDescription: []string{""},
	},
	"alltrue": {
		Description:      "`alltrue` returns `true` if all elements in a given collection are `true` or `&#34;true&#34;`. It also returns `true` if the collection is empty.",
		ParamDescription: []string{""},
	},
	"anytrue": {
		Description:      "`anytrue` returns `true` if any element in a given collection is `true` or `&#34;true&#34;`. It also returns `false` if the collection is empty.",
		ParamDescription: []string{""},
	},
	"base64decode": {
		Description:      "`base64decode` takes a string containing a Base64 character sequence and returns the original string.",
		ParamDescription: []string{""},
	},
	"base64encode": {
		Description:      "`base64encode` applies Base64 encoding to a string.",
		ParamDescription: []string{""},
	},
	"base64gzip": {
		Description:      "`base64gzip` compresses a string with gzip and then encodes the result in Base64 encoding.",
		ParamDescription: []string{""},
	},
	"base64sha256": {
		Description:      "`base64sha256` computes the SHA256 hash of a given string and encodes it with Base64. This is not equivalent to `base64encode(sha256(&#34;test&#34;))` since `sha256()` returns hexadecimal representation.",
		ParamDescription: []string{""},
	},
	"base64sha512": {
		Description:      "`base64sha512` computes the SHA512 hash of a given string and encodes it with Base64. This is not equivalent to `base64encode(sha512(&#34;test&#34;))` since `sha512()` returns hexadecimal representation.",
		ParamDescription: []string{""},
	},
	"basename": {
		Description:      "`basename` takes a string containing a filesystem path and removes all except the last portion from it.",
		ParamDescription: []string{""},
	},
	"bcrypt": {
		Description:      "`bcrypt` computes a hash of the given string using the Blowfish cipher, returning a string in [the _Modular Crypt Format_](https://passlib.readthedocs.io/en/stable/modular_crypt_format.html) usually expected in the shadow password file on many Unix systems.",
		ParamDescription: []string{"", "The `cost` argument is optional and will default to 10 if unspecified."},
	},
	"can": {
		Description:      "`can` evaluates the given expression and returns a boolean value indicating whether the expression produced a result without any errors.",
		ParamDescription: []string{""},
	},
	"ceil": {
		Description:      "`ceil` returns the closest whole number that is greater than or equal to the given value, which may be a fraction.",
		ParamDescription: []string{""},
	},
	"chomp": {
		Description:      "`chomp` removes newline characters at the end of a string.",
		ParamDescription: []string{""},
	},
	"chunklist": {
		Description:      "`chunklist` splits a single list into fixed-size chunks, returning a list of lists.",
		ParamDescription: []string{"", ""},
	},
	"cidrhost": {
		Description:      "`cidrhost` calculates a full host IP address for a given host number within a given IP network address prefix.",
		ParamDescription: []string{"`prefix` must be given in CIDR notation, as defined in [RFC 4632 section 3.1](https://tools.ietf.org/html/rfc4632#section-3.1).", "`hostnum` is a whole number that can be represented as a binary integer with no more than the number of digits remaining in the address after the given prefix."},
	},
	"cidrnetmask": {
		Description:      "`cidrnetmask` converts an IPv4 address prefix given in CIDR notation into a subnet mask address.",
		ParamDescription: []string{"`prefix` must be given in CIDR notation, as defined in [RFC 4632 section 3.1](https://tools.ietf.org/html/rfc4632#section-3.1)."},
	},
	"cidrsubnet": {
		Description:      "`cidrsubnet` calculates a subnet address within given IP network address prefix.",
		ParamDescription: []string{"", "", ""},
	},
	"cidrsubnets": {
		Description:      "`cidrsubnets` calculates a sequence of consecutive IP address ranges within a particular CIDR prefix.",
		ParamDescription: []string{"", ""},
	},
	"coalesce": {
		Description:      "`coalesce` takes any number of arguments and returns the first one that isn&#39;t null or an empty string.",
		ParamDescription: []string{""},
	},
	"coalescelist": {
		Description:      "`coalescelist` takes any number of list arguments and returns the first one that isn&#39;t empty.",
		ParamDescription: []string{""},
	},
	"compact": {
		Description:      "`compact` takes a list of strings and returns a new list with any empty string elements removed.",
		ParamDescription: []string{""},
	},
	"concat": {
		Description:      "`concat` takes two or more lists and combines them into a single list.",
		ParamDescription: []string{""},
	},
	"contains": {
		Description:      "`contains` determines whether a given list or set contains a given single value as one of its elements.",
		ParamDescription: []string{"", ""},
	},
	"csvdecode": {
		Description:      "`csvdecode` decodes a string containing CSV-formatted data and produces a list of maps representing that data.",
		ParamDescription: []string{""},
	},
	"dirname": {
		Description:      "`dirname` takes a string containing a filesystem path and removes the last portion from it.",
		ParamDescription: []string{""},
	},
	"distinct": {
		Description:      "`distinct` takes a list and returns a new list with any duplicate elements removed.",
		ParamDescription: []string{""},
	},
	"element": {
		Description:      "`element` retrieves a single element from a list.",
		ParamDescription: []string{"", ""},
	},
	"endswith": {
		Description:      "`endswith` takes two values: a string to check and a suffix string. The function returns true if the first string ends with that exact suffix.",
		ParamDescription: []string{"", ""},
	},
	"file": {
		Description:      "`file` reads the contents of a file at the given path and returns them as a string.",
		ParamDescription: []string{""},
	},
	"filebase64": {
		Description:      "`filebase64` reads the contents of a file at the given path and returns them as a base64-encoded string.",
		ParamDescription: []string{""},
	},
	"filebase64sha256": {
		Description:      "`filebase64sha256` is a variant of `base64sha256` that hashes the contents of a given file rather than a literal string.",
		ParamDescription: []string{""},
	},
	"filebase64sha512": {
		Description:      "`filebase64sha512` is a variant of `base64sha512` that hashes the contents of a given file rather than a literal string.",
		ParamDescription: []string{""},
	},
	"fileexists": {
		Description:      "`fileexists` determines whether a file exists at a given path.",
		ParamDescription: []string{""},
	},
	"filemd5": {
		Description:      "`filemd5` is a variant of `md5` that hashes the contents of a given file rather than a literal string.",
		ParamDescription: []string{""},
	},
	"fileset": {
		Description:      "`fileset` enumerates a set of regular file names given a path and pattern. The path is automatically removed from the resulting set of file names and any result still containing path separators always returns forward slash (`/`) as the path separator for cross-system compatibility.",
		ParamDescription: []string{"", ""},
	},
	"filesha1": {
		Description:      "`filesha1` is a variant of `sha1` that hashes the contents of a given file rather than a literal string.",
		ParamDescription: []string{""},
	},
	"filesha256": {
		Description:      "`filesha256` is a variant of `sha256` that hashes the contents of a given file rather than a literal string.",
		ParamDescription: []string{""},
	},
	"filesha512": {
		Description:      "`filesha512` is a variant of `sha512` that hashes the contents of a given file rather than a literal string.",
		ParamDescription: []string{""},
	},
	"flatten": {
		Description:      "`flatten` takes a list and replaces any elements that are lists with a flattened sequence of the list contents.",
		ParamDescription: []string{""},
	},
	"floor": {
		Description:      "`floor` returns the closest whole number that is less than or equal to the given value, which may be a fraction.",
		ParamDescription: []string{""},
	},
	"format": {
		Description:      "The `format` function produces a string by formatting a number of other values according to a specification string. It is similar to the `printf` function in C, and other similar functions in other programming languages.",
		ParamDescription: []string{"", ""},
	},
	"formatdate": {
		Description:      "`formatdate` converts a timestamp into a different time format.",
		ParamDescription: []string{"", ""},
	},
	"formatlist": {
		Description:      "`formatlist` produces a list of strings by formatting a number of other values according to a specification string.",
		ParamDescription: []string{"", ""},
	},
	"indent": {
		Description:      "`indent` adds a given number of spaces to the beginnings of all but the first line in a given multi-line string.",
		ParamDescription: []string{"", ""},
	},
	"index": {
		Description:      "`index` finds the element index for a given value in a list.",
		ParamDescription: []string{"", ""},
	},
	"join": {
		Description:      "`join` produces a string by concatenating together all elements of a given list of strings with the given delimiter.",
		ParamDescription: []string{"", ""},
	},
	"jsondecode": {
		Description:      "`jsondecode` interprets a given string as JSON, returning a representation of the result of decoding that string.",
		ParamDescription: []string{""},
	},
	"jsonencode": {
		Description:      "`jsonencode` encodes a given value to a string using JSON syntax.",
		ParamDescription: []string{""},
	},
	"keys": {
		Description:      "`keys` takes a map and returns a list containing the keys from that map.",
		ParamDescription: []string{""},
	},
	"length": {
		Description:      "`length` determines the length of a given list, map, or string.",
		ParamDescription: []string{""},
	},
	"list": {
		Description:      "The `list` function is no longer available. Prior to Terraform v0.12 it was the only available syntax for writing a literal list inside an expression, but Terraform v0.12 introduced a new first-class syntax.",
		ParamDescription: []string{""},
	},
	"log": {
		Description:      "`log` returns the logarithm of a given number in a given base.",
		ParamDescription: []string{"", ""},
	},
	"lookup": {
		Description:      "`lookup` retrieves the value of a single element from a map, given its key. If the given key does not exist, the given default value is returned instead.",
		ParamDescription: []string{"", "", ""},
	},
	"lower": {
		Description:      "`lower` converts all cased letters in the given string to lowercase.",
		ParamDescription: []string{""},
	},
	"map": {
		Description:      "The `map` function is no longer available. Prior to Terraform v0.12 it was the only available syntax for writing a literal map inside an expression, but Terraform v0.12 introduced a new first-class syntax.",
		ParamDescription: []string{""},
	},
	"matchkeys": {
		Description:      "`matchkeys` constructs a new list by taking a subset of elements from one list whose indexes match the corresponding indexes of values in another list.",
		ParamDescription: []string{"", "", ""},
	},
	"max": {
		Description:      "`max` takes one or more numbers and returns the greatest number from the set.",
		ParamDescription: []string{""},
	},
	"md5": {
		Description:      "`md5` computes the MD5 hash of a given string and encodes it with hexadecimal digits.",
		ParamDescription: []string{""},
	},
	"merge": {
		Description:      "`merge` takes an arbitrary number of maps or objects, and returns a single map or object that contains a merged set of elements from all arguments.",
		ParamDescription: []string{""},
	},
	"min": {
		Description:      "`min` takes one or more numbers and returns the smallest number from the set.",
		ParamDescription: []string{""},
	},
	"nonsensitive": {
		Description:      "`nonsensitive` takes a sensitive value and returns a copy of that value with the sensitive marking removed, thereby exposing the sensitive value.",
		ParamDescription: []string{""},
	},
	"one": {
		Description:      "`one` takes a list, set, or tuple value with either zero or one elements. If the collection is empty, `one` returns `null`. Otherwise, `one` returns the first element. If there are two or more elements then `one` will return an error.",
		ParamDescription: []string{""},
	},
	"parseint": {
		Description:      "`parseint` parses the given string as a representation of an integer in the specified base and returns the resulting number. The base must be between 2 and 62 inclusive.",
		ParamDescription: []string{"", ""},
	},
	"pathexpand": {
		Description:      "`pathexpand` takes a filesystem path that might begin with a `~` segment, and if so it replaces that segment with the current user&#39;s home directory path.",
		ParamDescription: []string{""},
	},
	"pow": {
		Description:      "`pow` calculates an exponent, by raising its first argument to the power of the second argument.",
		ParamDescription: []string{"", ""},
	},
	"range": {
		Description:      "`range` generates a list of numbers using a start value, a limit value, and a step value.",
		ParamDescription: []string{""},
	},
	"regex": {
		Description:      "`regex` applies a [regular expression](https://en.wikipedia.org/wiki/Regular_expression) to a string and returns the matching substrings.",
		ParamDescription: []string{"", ""},
	},
	"regexall": {
		Description:      "`regexall` applies a [regular expression](https://en.wikipedia.org/wiki/Regular_expression) to a string and returns a list of all matches.",
		ParamDescription: []string{"", ""},
	},
	"replace": {
		Description:      "`replace` searches a given string for another given substring, and replaces each occurrence with a given replacement string.",
		ParamDescription: []string{"", "", ""},
	},
	"reverse": {
		Description:      "`reverse` takes a sequence and produces a new sequence of the same length with all of the same elements as the given sequence but in reverse order.",
		ParamDescription: []string{""},
	},
	"rsadecrypt": {
		Description:      "`rsadecrypt` decrypts an RSA-encrypted ciphertext, returning the corresponding cleartext.",
		ParamDescription: []string{"", ""},
	},
	"sensitive": {
		Description:      "`sensitive` takes any value and returns a copy of it marked so that Terraform will treat it as sensitive, with the same meaning and behavior as for [sensitive input variables](/language/values/variables#suppressing-values-in-cli-output).",
		ParamDescription: []string{""},
	},
	"setintersection": {
		Description:      "The `setintersection` function takes multiple sets and produces a single set containing only the elements that all of the given sets have in common. In other words, it computes the [intersection](https://en.wikipedia.org/wiki/Intersection_\\(set_theory\\)) of the sets.",
		ParamDescription: []string{"", ""},
	},
	"setproduct": {
		Description:      "The `setproduct` function finds all of the possible combinations of elements from all of the given sets by computing the [Cartesian product](https://en.wikipedia.org/wiki/Cartesian_product).",
		ParamDescription: []string{""},
	},
	"setsubtract": {
		Description:      "The `setsubtract` function returns a new set containing the elements from the first set that are not present in the second set. In other words, it computes the [relative complement](https://en.wikipedia.org/wiki/Complement_\\(set_theory\\)#Relative_complement) of the second set.",
		ParamDescription: []string{"", ""},
	},
	"setunion": {
		Description:      "The `setunion` function takes multiple sets and produces a single set containing the elements from all of the given sets. In other words, it computes the [union](https://en.wikipedia.org/wiki/Union_\\(set_theory\\)) of the sets.",
		ParamDescription: []string{"", ""},
	},
	"sha1": {
		Description:      "`sha1` computes the SHA1 hash of a given string and encodes it with hexadecimal digits.",
		ParamDescription: []string{""},
	},
	"sha256": {
		Description:      "`sha256` computes the SHA256 hash of a given string and encodes it with hexadecimal digits.",
		ParamDescription: []string{""},
	},
	"sha512": {
		Description:      "`sha512` computes the SHA512 hash of a given string and encodes it with hexadecimal digits.",
		ParamDescription: []string{""},
	},
	"signum": {
		Description:      "`signum` determines the sign of a number, returning a number between -1 and 1 to represent the sign.",
		ParamDescription: []string{""},
	},
	"slice": {
		Description:      "`slice` extracts some consecutive elements from within a list.",
		ParamDescription: []string{"", "", ""},
	},
	"sort": {
		Description:      "`sort` takes a list of strings and returns a new list with those strings sorted lexicographically.",
		ParamDescription: []string{""},
	},
	"split": {
		Description:      "`split` produces a list by dividing a given string at all occurrences of a given separator.",
		ParamDescription: []string{"", ""},
	},
	"startswith": {
		Description:      "`startswith` takes two values: a string to check and a prefix string. The function returns true if the string begins with that exact prefix.",
		ParamDescription: []string{"", ""},
	},
	"strrev": {
		Description:      "`strrev` reverses the characters in a string. Note that the characters are treated as _Unicode characters_ (in technical terms, Unicode [grapheme cluster boundaries](https://unicode.org/reports/tr29/#Grapheme_Cluster_Boundaries) are respected).",
		ParamDescription: []string{""},
	},
	"substr": {
		Description:      "`substr` extracts a substring from a given string by offset and (maximum) length.",
		ParamDescription: []string{"", "", ""},
	},
	"sum": {
		Description:      "`sum` takes a list or set of numbers and returns the sum of those numbers.",
		ParamDescription: []string{""},
	},
	"templatefile": {
		Description:      "`templatefile` reads the file at the given path and renders its content as a template using a supplied set of template variables.",
		ParamDescription: []string{"", ""},
	},
	"textdecodebase64": {
		Description:      "`textdecodebase64` function decodes a string that was previously Base64-encoded, and then interprets the result as characters in a specified character encoding.",
		ParamDescription: []string{"", ""},
	},
	"textencodebase64": {
		Description:      "`textencodebase64` encodes the unicode characters in a given string using a specified character encoding, returning the result base64 encoded because Terraform language strings are always sequences of unicode characters.",
		ParamDescription: []string{"", ""},
	},
	"timeadd": {
		Description:      "`timeadd` adds a duration to a timestamp, returning a new timestamp.",
		ParamDescription: []string{"", ""},
	},
	"timecmp": {
		Description:      "`timecmp` compares two timestamps and returns a number that represents the ordering of the instants those timestamps represent.",
		ParamDescription: []string{"", ""},
	},
	"timestamp": {
		Description:      "`timestamp` returns a UTC timestamp string in [RFC 3339](https://tools.ietf.org/html/rfc3339) format.",
		ParamDescription: []string{},
	},
	"title": {
		Description:      "`title` converts the first letter of each word in the given string to uppercase.",
		ParamDescription: []string{""},
	},
	"tobool": {
		Description:      "`tobool` converts its argument to a boolean value.",
		ParamDescription: []string{""},
	},
	"tolist": {
		Description:      "`tolist` converts its argument to a list value.",
		ParamDescription: []string{""},
	},
	"tomap": {
		Description:      "`tomap` converts its argument to a map value.",
		ParamDescription: []string{""},
	},
	"tonumber": {
		Description:      "`tonumber` converts its argument to a number value.",
		ParamDescription: []string{""},
	},
	"toset": {
		Description:      "`toset` converts its argument to a set value.",
		ParamDescription: []string{""},
	},
	"tostring": {
		Description:      "`tostring` converts its argument to a string value.",
		ParamDescription: []string{""},
	},
	"transpose": {
		Description:      "`transpose` takes a map of lists of strings and swaps the keys and values to produce a new map of lists of strings.",
		ParamDescription: []string{""},
	},
	"trim": {
		Description:      "`trim` removes the specified set of characters from the start and end of the given string.",
		ParamDescription: []string{"", ""},
	},
	"trimprefix": {
		Description:      "`trimprefix` removes the specified prefix from the start of the given string. If the string does not start with the prefix, the string is returned unchanged.",
		ParamDescription: []string{"", ""},
	},
	"trimspace": {
		Description:      "`trimspace` removes any space characters from the start and end of the given string.",
		ParamDescription: []string{""},
	},
	"trimsuffix": {
		Description:      "`trimsuffix` removes the specified suffix from the end of the given string.",
		ParamDescription: []string{"", ""},
	},
	"try": {
		Description:      "`try` evaluates all of its argument expressions in turn and returns the result of the first one that does not produce any errors.",
		ParamDescription: []string{""},
	},
	"type": {
		Description:      "`type` returns the type of a given value.",
		ParamDescription: []string{""},
	},
	"upper": {
		Description:      "`upper` converts all cased letters in the given string to uppercase.",
		ParamDescription: []string{""},
	},
	"urlencode": {
		Description:      "`urlencode` applies URL encoding to a given string.",
		ParamDescription: []string{""},
	},
	"uuid": {
		Description:      "`uuid` generates a unique identifier string.",
		ParamDescription: []string{},
	},
	"uuidv5": {
		Description:      "`uuidv5` generates a _name-based_ UUID, as described in [RFC 4122 section 4.3](https://tools.ietf.org/html/rfc4122#section-4.3), also known as a &#34;version 5&#34; UUID.",
		ParamDescription: []string{"", ""},
	},
	"values": {
		Description:      "`values` takes a map and returns a list containing the values of the elements in that map.",
		ParamDescription: []string{""},
	},
	"yamldecode": {
		Description:      "`yamldecode` parses a string as a subset of YAML, and produces a representation of its value.",
		ParamDescription: []string{""},
	},
	"yamlencode": {
		Description:      "`yamlencode` encodes a given value to a string using [YAML 1.2](https://yaml.org/spec/1.2/spec.html) block syntax.",
		ParamDescription: []string{""},
	},
	"zipmap": {
		Description:      "`zipmap` constructs a map from a list of keys and a corresponding list of values.",
		ParamDescription: []string{"", ""},
	},
}

// WithDescription looks up the description for a given function and uses
// go-cty's WithNewDescriptions to replace the functions description and
// parameter descriptions.
func WithDescription(name string, f function.Function) function.Function {
	desc, ok := descriptionList[name]
	if !ok {
		return f
	}

	// Will panic if ParamDescription doesn't match the number of parameters
	// the function expects
	return f.WithNewDescriptions(desc.Description, desc.ParamDescription)
}
