/*
Package jsonapi provides a serializer and deserializer for jsonapi.org spec payloads.

You can keep your model structs as is and use struct field tags to indicate to jsonapi
how you want your response built or your request deserialzied. What about my relationships?
jsonapi supports relationships out of the box and will even side load them in your response
into an "included" array--that contains associated objects.

jsonapi uses StructField tags to annotate the structs fields that you already have and use
in your app and then reads and writes jsonapi.org output based on the instructions you give
the library in your jsonapi tags.

Example structs using a Blog > Post > Comment structure,

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
	}

jsonapi Tag Reference

Value, primary: "primary,<type field output>"

This indicates that this is the primary key field for this struct type. Tag
value arguments are comma separated.  The first argument must be, "primary", and
the second must be the name that should appear in the "type" field for all data
objects that represent this type of model.

Value, attr: "attr,<key name in attributes hash>[,<extra arguments>]"

These fields' values should end up in the "attribute" hash for a record.  The first
argument must be, "attr', and the second should be the name for the key to display in
the the "attributes" hash for that record.

The following extra arguments are also supported:

"omitempty": excludes the fields value from the "attribute" hash.
"iso8601": uses the ISO8601 timestamp format when serialising or deserialising the time.Time value.

Value, relation: "relation,<key name in relationships hash>"

Relations are struct fields that represent a one-to-one or one-to-many to other structs.
jsonapi will traverse the graph of relationships and marshal or unmarshal records.  The first
argument must be, "relation", and the second should be the name of the relationship, used as
the key in the "relationships" hash for the record.

Use the methods below to Marshal and Unmarshal jsonapi.org json payloads.

Visit the readme at https://github.com/google/jsonapi
*/
package jsonapi
