# jsonapi

[![Build Status](https://travis-ci.org/google/jsonapi.svg?branch=master)](https://travis-ci.org/google/jsonapi)
[![Go Report Card](https://goreportcard.com/badge/github.com/google/jsonapi)](https://goreportcard.com/report/github.com/google/jsonapi)
[![GoDoc](https://godoc.org/github.com/google/jsonapi?status.svg)](http://godoc.org/github.com/google/jsonapi)

A serializer/deserializer for JSON payloads that comply to the
[JSON API - jsonapi.org](http://jsonapi.org) spec in go.

## Installation

```
go get -u github.com/google/jsonapi
```

Or, see [Alternative Installation](#alternative-installation).

## Background

You are working in your Go web application and you have a struct that is
organized similarly to your database schema.  You need to send and
receive json payloads that adhere to the JSON API spec.  Once you realize that
your json needed to take on this special form, you go down the path of
creating more structs to be able to serialize and deserialize JSON API
payloads.  Then there are more models required with this additional
structure.  Ugh! With JSON API, you can keep your model structs as is and
use [StructTags](http://golang.org/pkg/reflect/#StructTag) to indicate
to JSON API how you want your response built or your request
deserialized.  What about your relationships?  JSON API supports
relationships out of the box and will even put them in your response
into an `included` side-loaded slice--that contains associated records.

## Introduction

JSON API uses [StructField](http://golang.org/pkg/reflect/#StructField)
tags to annotate the structs fields that you already have and use in
your app and then reads and writes [JSON API](http://jsonapi.org)
output based on the instructions you give the library in your JSON API
tags.  Let's take an example.  In your app, you most likely have structs
that look similar to these:


```go
type Blog struct {
	ID            int       `json:"id"`
	Title         string    `json:"title"`
	Posts         []*Post   `json:"posts"`
	CurrentPost   *Post     `json:"current_post"`
	CurrentPostId int       `json:"current_post_id"`
	CreatedAt     time.Time `json:"created_at"`
	ViewCount     int       `json:"view_count"`
}

type Post struct {
	ID       int        `json:"id"`
	BlogID   int        `json:"blog_id"`
	Title    string     `json:"title"`
	Body     string     `json:"body"`
	Comments []*Comment `json:"comments"`
}

type Comment struct {
	Id     int    `json:"id"`
	PostID int    `json:"post_id"`
	Body   string `json:"body"`
	Likes  uint   `json:"likes_count,omitempty"`
}
```

These structs may or may not resemble the layout of your database.  But
these are the ones that you want to use right?  You wouldn't want to use
structs like those that JSON API sends because it is difficult to get at
all of your data easily.

## Example App

[examples/app.go](https://github.com/google/jsonapi/blob/master/examples/app.go)

This program demonstrates the implementation of a create, a show,
and a list [http.Handler](http://golang.org/pkg/net/http#Handler).  It
outputs some example requests and responses as well as serialized
examples of the source/target structs to json.  That is to say, I show
you that the library has successfully taken your JSON API request and
turned it into your struct types.

To run,

* Make sure you have [Go installed](https://golang.org/doc/install)
* Create the following directories or similar: `~/go`
* Set `GOPATH` to `PWD` in your shell session, `export GOPATH=$PWD`
* `go get github.com/google/jsonapi`.  (Append `-u` after `get` if you
  are updating.)
* `cd $GOPATH/src/github.com/google/jsonapi/examples`
* `go build && ./examples`

## `jsonapi` Tag Reference

### Example

The `jsonapi` [StructTags](http://golang.org/pkg/reflect/#StructTag)
tells this library how to marshal and unmarshal your structs into
JSON API payloads and your JSON API payloads to structs, respectively.
Then Use JSON API's Marshal and Unmarshal methods to construct and read
your responses and replies.  Here's an example of the structs above
using JSON API tags:

```go
type Blog struct {
	ID            int       `jsonapi:"primary,blogs"`
	Title         string    `jsonapi:"attr,title"`
	Posts         []*Post   `jsonapi:"relation,posts"`
	CurrentPost   *Post     `jsonapi:"relation,current_post"`
	CurrentPostID int       `jsonapi:"attr,current_post_id"`
	CreatedAt     time.Time `jsonapi:"attr,created_at"`
	ViewCount     int       `jsonapi:"attr,view_count"`
}

type Post struct {
	ID       int        `jsonapi:"primary,posts"`
	BlogID   int        `jsonapi:"attr,blog_id"`
	Title    string     `jsonapi:"attr,title"`
	Body     string     `jsonapi:"attr,body"`
	Comments []*Comment `jsonapi:"relation,comments"`
}

type Comment struct {
	ID     int    `jsonapi:"primary,comments"`
	PostID int    `jsonapi:"attr,post_id"`
	Body   string `jsonapi:"attr,body"`
	Likes  uint   `jsonapi:"attr,likes-count,omitempty"`
}
```

### Permitted Tag Values

#### `primary`

```
`jsonapi:"primary,<type field output>"`
```

This indicates this is the primary key field for this struct type.
Tag value arguments are comma separated.  The first argument must be,
`primary`, and the second must be the name that should appear in the
`type`\* field for all data objects that represent this type of model.

\* According the [JSON API](http://jsonapi.org) spec, the plural record
types are shown in the examples, but not required.

#### `attr`

```
`jsonapi:"attr,<key name in attributes hash>,<optional: omitempty>"`
```

These fields' values will end up in the `attributes`hash for a record.
The first argument must be, `attr`, and the second should be the name
for the key to display in the `attributes` hash for that record. The optional
third argument is `omitempty` - if it is present the field will not be present
in the `"attributes"` if the field's value is equivalent to the field types
empty value (ie if the `count` field is of type `int`, `omitempty` will omit the
field when `count` has a value of `0`). Lastly, the spec indicates that
`attributes` key names should be dasherized for multiple word field names.

#### `relation`

```
`jsonapi:"relation,<key name in relationships hash>,<optional: omitempty>"`
```

Relations are struct fields that represent a one-to-one or one-to-many
relationship with other structs. JSON API will traverse the graph of
relationships and marshal or unmarshal records.  The first argument must
be, `relation`, and the second should be the name of the relationship,
used as the key in the `relationships` hash for the record. The optional
third argument is `omitempty` - if present will prevent non existent to-one and
to-many from being serialized.

## Methods Reference

**All `Marshal` and `Unmarshal` methods expect pointers to struct
instance or slices of the same contained with the `interface{}`s**

Now you have your structs prepared to be seralized or materialized, What
about the rest?

### Create Record Example

You can Unmarshal a JSON API payload using
[jsonapi.UnmarshalPayload](http://godoc.org/github.com/google/jsonapi#UnmarshalPayload).
It reads from an [io.Reader](https://golang.org/pkg/io/#Reader)
containing a JSON API payload for one record (but can have related
records).  Then, it materializes a struct that you created and passed in
(using new or &).  Again, the method supports single records only, at
the top level, in request payloads at the moment. Bulk creates and
updates are not supported yet.

After saving your record, you can use,
[MarshalOnePayload](http://godoc.org/github.com/google/jsonapi#MarshalOnePayload),
to write the JSON API response to an
[io.Writer](https://golang.org/pkg/io/#Writer).

#### `UnmarshalPayload`

```go
UnmarshalPayload(in io.Reader, model interface{})
```

Visit [godoc](http://godoc.org/github.com/google/jsonapi#UnmarshalPayload)

#### `MarshalPayload`

```go
MarshalPayload(w io.Writer, models interface{}) error
```

Visit [godoc](http://godoc.org/github.com/google/jsonapi#MarshalPayload)

Writes a JSON API response, with related records sideloaded, into an
`included` array.  This method encodes a response for either a single record or
many records.

##### Handler Example Code

```go
func CreateBlog(w http.ResponseWriter, r *http.Request) {
	blog := new(Blog)

	if err := jsonapi.UnmarshalPayload(r.Body, blog); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// ...save your blog...

	w.Header().Set("Content-Type", jsonapi.MediaType)
	w.WriteHeader(http.StatusCreated)

	if err := jsonapi.MarshalPayload(w, blog); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
```

### Create Records Example

#### `UnmarshalManyPayload`

```go
UnmarshalManyPayload(in io.Reader, t reflect.Type) ([]interface{}, error)
```

Visit [godoc](http://godoc.org/github.com/google/jsonapi#UnmarshalManyPayload)

Takes an `io.Reader` and a `reflect.Type` representing the uniform type
contained within the `"data"` JSON API member.

##### Handler Example Code

```go
func CreateBlogs(w http.ResponseWriter, r *http.Request) {
	// ...create many blogs at once

	blogs, err := UnmarshalManyPayload(r.Body, reflect.TypeOf(new(Blog)))
	if err != nil {
		t.Fatal(err)
	}

	for _, blog := range blogs {
		b, ok := blog.(*Blog)
		// ...save each of your blogs
	}

	w.Header().Set("Content-Type", jsonapi.MediaType)
	w.WriteHeader(http.StatusCreated)

	if err := jsonapi.MarshalPayload(w, blogs); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
```


### Links

If you need to include [link objects](http://jsonapi.org/format/#document-links) along with response data, implement the `Linkable` interface for document-links, and `RelationshipLinkable` for relationship links:

```go
func (post Post) JSONAPILinks() *Links {
	return &Links{
		"self": "href": fmt.Sprintf("https://example.com/posts/%d", post.ID),
		"comments": Link{
			Href: fmt.Sprintf("https://example.com/api/blogs/%d/comments", post.ID),
			Meta: map[string]interface{}{
				"counts": map[string]uint{
					"likes":    4,
				},
			},
		},
	}
}

// Invoked for each relationship defined on the Post struct when marshaled
func (post Post) JSONAPIRelationshipLinks(relation string) *Links {
	if relation == "comments" {
		return &Links{
			"related": fmt.Sprintf("https://example.com/posts/%d/comments", post.ID),
		}
	}
	return nil
}
```

### Meta

 If you need to include [meta objects](http://jsonapi.org/format/#document-meta) along with response data, implement the `Metable` interface for document-meta, and `RelationshipMetable` for relationship meta:

 ```go
func (post Post) JSONAPIMeta() *Meta {
	return &Meta{
		"details": "sample details here",
	}
}

// Invoked for each relationship defined on the Post struct when marshaled
func (post Post) JSONAPIRelationshipMeta(relation string) *Meta {
	if relation == "comments" {
		return &Meta{
			"this": map[string]interface{}{
				"can": map[string]interface{}{
					"go": []interface{}{
						"as",
						"deep",
						map[string]interface{}{
							"as": "required",
						},
					},
				},
			},
		}
	}
	return nil
}
```

### Errors
This package also implements support for JSON API compatible `errors` payloads using the following types.

#### `MarshalErrors`
```go
MarshalErrors(w io.Writer, errs []*ErrorObject) error
```

Writes a JSON API response using the given `[]error`.

#### `ErrorsPayload`
```go
type ErrorsPayload struct {
	Errors []*ErrorObject `json:"errors"`
}
```

ErrorsPayload is a serializer struct for representing a valid JSON API errors payload.

#### `ErrorObject`
```go
type ErrorObject struct { ... }

// Error implements the `Error` interface.
func (e *ErrorObject) Error() string {
	return fmt.Sprintf("Error: %s %s\n", e.Title, e.Detail)
}
```

ErrorObject is an `Error` implementation as well as an implementation of the JSON API error object.

The main idea behind this struct is that you can use it directly in your code as an error type and pass it directly to `MarshalErrors` to get a valid JSON API errors payload.

##### Errors Example Code
```go
// An error has come up in your code, so set an appropriate status, and serialize the error.
if err := validate(&myStructToValidate); err != nil {
	context.SetStatusCode(http.StatusBadRequest) // Or however you need to set a status.
	jsonapi.MarshalErrors(w, []*ErrorObject{{
		Title: "Validation Error",
		Detail: "Given request body was invalid.",
		Status: "400",
		Meta: map[string]interface{}{"field": "some_field", "error": "bad type", "expected": "string", "received": "float64"},
	}})
	return
}
```

## Testing

### `MarshalOnePayloadEmbedded`

```go
MarshalOnePayloadEmbedded(w io.Writer, model interface{}) error
```

Visit [godoc](http://godoc.org/github.com/google/jsonapi#MarshalOnePayloadEmbedded)

This method is not strictly meant to for use in implementation code,
although feel free.  It was mainly created for use in tests; in most cases,
your request payloads for create will be embedded rather than sideloaded
for related records.  This method will serialize a single struct pointer
into an embedded json response.  In other words, there will be no,
`included`, array in the json; all relationships will be serialized
inline with the data.

However, in tests, you may want to construct payloads to post to create
methods that are embedded to most closely model the payloads that will
be produced by the client.  This method aims to enable that.

### Example

```go
out := bytes.NewBuffer(nil)

// testModel returns a pointer to a Blog
jsonapi.MarshalOnePayloadEmbedded(out, testModel())

h := new(BlogsHandler)

w := httptest.NewRecorder()
r, _ := http.NewRequest(http.MethodPost, "/blogs", out)

h.CreateBlog(w, r)

blog := new(Blog)
jsonapi.UnmarshalPayload(w.Body, blog)

// ... assert stuff about blog here ...
```

## Alternative Installation
I use git subtrees to manage dependencies rather than `go get` so that
the src is committed to my repo.

```
git subtree add --squash --prefix=src/github.com/google/jsonapi https://github.com/google/jsonapi.git master
```

To update,

```
git subtree pull --squash --prefix=src/github.com/google/jsonapi https://github.com/google/jsonapi.git master
```

This assumes that I have my repo structured with a `src` dir containing
a collection of packages and `GOPATH` is set to the root
folder--containing `src`.

## Contributing

Fork, Change, Pull Request *with tests*.
