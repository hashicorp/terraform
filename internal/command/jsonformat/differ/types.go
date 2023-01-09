package differ

// JsonType is a wrapper around a string type to describe the various different
// kinds of JSON types. This is used when processing dynamic types and outputs.
type JsonType string

// NestingMode is a wrapper around a string type to describe the various
// different kinds of nesting modes that can be applied to nested blocks and
// objects.
type NestingMode string

const (
	jsonNumber JsonType = "number"
	jsonObject JsonType = "object"
	jsonArray  JsonType = "array"
	jsonBool   JsonType = "bool"
	jsonString JsonType = "string"
	jsonNull   JsonType = "null"

	nestingModeSet    NestingMode = "set"
	nestingModeList   NestingMode = "list"
	nestingModeMap    NestingMode = "map"
	nestingModeSingle NestingMode = "single"
	nestingModeGroup  NestingMode = "group"
)
