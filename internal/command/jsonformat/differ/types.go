package differ

type JsonType string
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
