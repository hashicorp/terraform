package circonus

// NOTE(sean): (ab)use the fact that _ is considered a letter type but not
// exported.  This allows us to create an intrapackage type system that won't
// leak to the outside world, but retains most of the style conventions of
// types, structs, and consts having upper case identifiers.  It's not perfect,
// but it helps ensure that the compiler is working for us down the road in
// terms of maintenance.
//
// One objectives of this was to ensure that based on aesthetics alone there
// should be very few locations where type assertions or casting in the main
// resource files is required (mainly when interacting with the external API
// structs).  As a rule of thumb, all type assertions should happen in the utils
// file and casting is only done at assignment time when storing a result to a
// struct.

type _APICheckType string

type _AttrDescr string
type _AttrDescrs map[_SchemaAttr]_AttrDescr

type _MetricType string
type _SchemaAttr string

type _MetricID string
type _MetricName string

type _Unit string

type _ValidString string
type _ValidStringValues []_ValidString
