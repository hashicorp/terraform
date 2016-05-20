form
====

A Form Encoding & Decoding Package for Go, written by [Alvaro J. Genial](http://alva.ro).

[![Build Status](https://travis-ci.org/ajg/form.png?branch=master)](https://travis-ci.org/ajg/form)
[![GoDoc](https://godoc.org/github.com/ajg/form?status.png)](https://godoc.org/github.com/ajg/form)

Synopsis
--------

This library is designed to allow seamless, high-fidelity encoding and decoding of arbitrary data in `application/x-www-form-urlencoded` format and as [`url.Values`](http://golang.org/pkg/net/url/#Values). It is intended to be useful primarily in dealing with web forms and URI query strings, both of which natively employ said format.

Unsurprisingly, `form` is modeled after other Go [`encoding`](http://golang.org/pkg/encoding/) packages, in particular [`encoding/json`](http://golang.org/pkg/encoding/json/), and follows the same conventions (see below for more.) It aims to automatically handle any kind of concrete Go [data value](#values) (i.e., not functions, channels, etc.) while providing mechanisms for custom behavior.

Status
------

The implementation is in usable shape and is fairly well tested with its accompanying test suite. The API is unlikely to change much, but still may. Lastly, the code has not yet undergone a security review to ensure it is free of vulnerabilities. Please file an issue or send a pull request for fixes & improvements.

Dependencies
------------

The only requirement is [Go 1.2](http://golang.org/doc/go1.2) or later.

Usage
-----

```go
import "github.com/ajg/form"
```

Given a type like the following...

```go
type User struct {
	Name         string            `form:"name"`
	Email        string            `form:"email"`
	Joined       time.Time         `form:"joined,omitempty"`
	Posts        []int             `form:"posts"`
	Preferences  map[string]string `form:"prefs"`
	Avatar       []byte            `form:"avatar"`
	PasswordHash int64             `form:"-"`
}
```

...it is easy to encode data of that type...


```go
func PostUser(url string, u User) error {
	var c http.Client
	_, err := c.PostForm(url, form.EncodeToValues(u))
	return err
}
```

...as well as decode it...


```go
func Handler(w http.ResponseWriter, r *http.Request) {
	var u User

	d := form.NewDecoder(r.Body)
	if err := d.Decode(&u); err != nil {
		http.Error(w, "Form could not be decoded", http.StatusBadRequest)
		return
	}

	fmt.Fprintf(w, "Decoded: %#v", u)
}
```

...without having to do any grunt work.

Field Tags
----------

Like other encoding packages, `form` supports the following options for fields:

 - `` `form:"-"` ``: Causes the field to be ignored during encoding and decoding.
 - `` `form:"<name>"` ``: Overrides the field's name; useful especially when dealing with external identifiers in camelCase, as are commonly found on the web.
 - `` `form:",omitempty"` ``: Elides the field during encoding if it is empty (typically meaning equal to the type's zero value.)
 - `` `form:"<name>,omitempty"` ``: The way to combine the two options above.

Values
------

### Simple Values

Values of the following types are all considered simple:

 - `bool`
 - `int`, `int8`, `int16`, `int32`, `int64`, `rune`
 - `uint`, `uint8`, `uint16`, `uint32`, `uint64`, `byte`
 - `float32`, `float64`
 - `complex64`, `complex128`
 - `string`
 - `[]byte` (see note)
 - [`time.Time`](http://golang.org/pkg/time/#Time)
 - [`url.URL`](http://golang.org/pkg/net/url/#URL)
 - An alias of any of the above
 - A pointer to any of the above

### Composite Values

A composite value is one that can contain other values. Values of the following kinds...

 - Maps
 - Slices; except `[]byte` (see note)
 - Structs; except [`time.Time`](http://golang.org/pkg/time/#Time) and [`url.URL`](http://golang.org/pkg/net/url/#URL)
 - Arrays
 - An alias of any of the above
 - A pointer to any of the above

...are considered composites in general, unless they implement custom marshaling/unmarshaling. Composite values are encoded as a flat mapping of paths to values, where the paths are constructed by joining the parent and child paths with a period (`.`).

(Note: a byte slice is treated as a `string` by default because it's more efficient, but can also be decoded as a slice—i.e., with indexes.)

### Untyped Values

While encouraged, it is not necessary to define a type (e.g. a `struct`) in order to use `form`, since it is able to encode and decode untyped data generically using the following rules:

 - Simple values will be treated as a `string`.
 - Composite values will be treated as a `map[string]interface{}`, itself able to contain nested values (both scalar and compound) ad infinitum.
 - However, if there is a value (of any supported type) already present in a map for a given key, then it will be used when possible, rather than being replaced with a generic value as specified above; this makes it possible to handle partially typed, dynamic or schema-less values.

### Unsupported Values

Values of the following kinds aren't supported and, if present, must be ignored.

 - Channel
 - Function
 - Unsafe pointer
 - An alias of any of the above
 - A pointer to any of the above

Custom Marshaling
-----------------

There is a default (generally lossless) marshaling & unmarshaling scheme for any concrete data value in Go, which is good enough in most cases. However, it is possible to override it and use a custom scheme. For instance, a "binary" field could be marshaled more efficiently using [base64](http://golang.org/pkg/encoding/base64/) to prevent it from being percent-escaped during serialization to `application/x-www-form-urlencoded` format.

Because `form` provides support for [`encoding.TextMarshaler`](http://golang.org/pkg/encoding/#TextMarshaler) and [`encoding.TextUnmarshaler`](http://golang.org/pkg/encoding/#TextUnmarshaler) it is easy to do that; for instance, like this:

```go
import "encoding"

type Binary []byte

var (
	_ encoding.TextMarshaler   = &Binary{}
	_ encoding.TextUnmarshaler = &Binary{}
)

func (b Binary) MarshalText() ([]byte, error) {
	return []byte(base64.URLEncoding.EncodeToString([]byte(b))), nil
}

func (b *Binary) UnmarshalText(text []byte) error {
	bs, err := base64.URLEncoding.DecodeString(string(text))
	if err == nil {
		*b = Binary(bs)
	}
	return err
}
```

Now any value with type `Binary` will automatically be encoded using the [URL](http://golang.org/pkg/encoding/base64/#URLEncoding) variant of base64. It is left as an exercise to the reader to improve upon this scheme by eliminating the need for padding (which, besides being superfluous, uses `=`, a character that will end up percent-escaped.)

Keys
----

In theory any value can be a key as long as it has a string representation. However, periods have special meaning to `form`, and thus, under the hood (i.e. in encoded form) they are transparently escaped using a preceding backslash (`\`). Backslashes within keys, themselves, are also escaped in this manner (e.g. as `\\`) in order to permit representing `\.` itself (as `\\\.`).

(Note: it is normally unnecessary to deal with this issue unless keys are being constructed manually—e.g. literally embedded in HTML or in a URI.)

Limitations
-----------

 - Circular (self-referential) values are untested.

Future Work
-----------

The following items would be nice to have in the future—though they are not being worked on yet:

 - An option to treat all values as if they had been tagged with `omitempty`.
 - An option to automatically treat all field names in `camelCase` or `underscore_case`.
 - Built-in support for the types in [`math/big`](http://golang.org/pkg/math/big/).
 - Built-in support for the types in [`image/color`](http://golang.org/pkg/image/color/).
 - Improve encoding/decoding by reading/writing directly from/to the `io.Reader`/`io.Writer` when possible, rather than going through an intermediate representation (i.e. `node`) which requires more memory.

(Feel free to implement any of these and then send a pull request.)

Related Work
------------

 - Package [gorilla/schema](https://github.com/gorilla/schema), which only implements decoding.
 - Package [google/go-querystring](https://github.com/google/go-querystring), which only implements encoding.

License
-------

This library is distributed under a BSD-style [LICENSE](./LICENSE).
