package circonus

// NOTE(sean): One of the objectives of the use of types is to ensure that based
// on aesthetics alone are very few locations where type assertions or casting
// in the main resource files is required (mainly when interacting with the
// external API structs).  As a rule of thumb, all type assertions should happen
// in the utils file and casting is only done at assignment time when storing a
// result to a struct.  Said differently, contained tedium should enable
// compiler enforcement of types and easy verification.

type apiCheckType string

type attrDescr string
type attrDescrs map[schemaAttr]attrDescr

type schemaAttr string

type metricID string

type validString string
type validStringValues []validString
