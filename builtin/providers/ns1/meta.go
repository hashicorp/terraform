package ns1

import (
	"fmt"
	"reflect"

	"github.com/hashicorp/terraform/helper/schema"
	"gopkg.in/ns1/ns1-go.v2/rest/model/data"
)

type TfSchemaBuilder func(*schema.Schema)

func mtSimple(t schema.ValueType) TfSchemaBuilder {
	return func(s *schema.Schema) {
		s.Type = t
	}
}

func mtStringEnum(se *StringEnum) TfSchemaBuilder {
	return func(s *schema.Schema) {
		s.Type = schema.TypeString
		s.ValidateFunc = func(v interface{}, k string) ([]string, []error) {
			_, err := se.Check(v.(string))
			if err != nil {
				return nil, []error{err}
			}
			return nil, nil
		}
	}
}

var mtInt TfSchemaBuilder = mtSimple(schema.TypeInt)
var mtBool TfSchemaBuilder = mtSimple(schema.TypeBool)
var mtString TfSchemaBuilder = mtSimple(schema.TypeString)
var mtFloat64 TfSchemaBuilder = mtSimple(schema.TypeFloat)

func mtList(elementSchemaBuilder TfSchemaBuilder) TfSchemaBuilder {
	return func(s *schema.Schema) {
		s.Type = schema.TypeList
		elementSchema := &schema.Schema{}
		elementSchemaBuilder(elementSchema)
		s.Elem = elementSchema
	}
}

var mtStringList TfSchemaBuilder = mtList(mtString)

type MetaFieldSpec struct {
	NameInDynamic string
	NameInStruct  string
	SchemaBuilder TfSchemaBuilder
}

type MetaField struct {
	MetaFieldSpec
	NameInDynamicForFeed string
	StructIndex          int
	StructGoType         reflect.Type
}

var georegionEnum *StringEnum = NewStringEnum([]string{
	"US-WEST",
	"US-EAST",
	"US-CENTRAL",
	"EUROPE",
	"AFRICA",
	"ASIAPAC",
	"SOUTH-AMERICA",
})

func makeMetaFields() []MetaField {
	var specs []MetaFieldSpec = []MetaFieldSpec{
		{"up", "Up", mtBool},
		{"connections", "Connections", mtInt},
		{"requests", "Requests", mtInt},
		{"loadavg", "LoadAvg", mtFloat64},
		{"pulsar", "Pulsar", mtInt},
		{"latitude", "Latitude", mtFloat64},
		{"longitude", "Longitude", mtFloat64},
		{"georegion", "Georegion", mtList(mtStringEnum(georegionEnum))},
		{"country", "Country", mtStringList},
		{"us_state", "USState", mtStringList},
		{"ca_province", "CAProvince", mtStringList},
		{"note", "Note", mtString},
		{"ip_prefixes", "IPPrefixes", mtStringList},
		{"asn", "ASN", mtList(mtInt)},
		{"priority", "Priority", mtInt},
		{"weight", "Weight", mtFloat64},
		{"low_watermark", "LowWatermark", mtInt},
		{"high_watermark", "HighWatermark", mtInt},
	}

	// Figure out the field indexes (in data.Meta) for all the fields.
	// This way we can later lookup by index, which should be faster than by name.

	rt := reflect.TypeOf(data.Meta{})
	fields := make([]MetaField, len(specs))
	for i, spec := range specs {
		rf, present := rt.FieldByName(spec.NameInStruct)
		if !present {
			panic(fmt.Sprintf("Field %q not present", spec.NameInStruct))
		}
		if len(rf.Index) != 1 {
			panic(fmt.Sprintf("Expecting a single index, got %#v", rf.Index))
		}
		index := rf.Index[0]
		fields[i] = MetaField{
			MetaFieldSpec:        spec,
			StructIndex:          index,
			NameInDynamicForFeed: spec.NameInDynamic + "_feed",
			StructGoType:         rf.Type,
		}
	}

	return fields
}

var metaFields []MetaField = makeMetaFields()

func makeMetaSchema() *schema.Schema {
	fields := make(map[string]*schema.Schema)

	for _, f := range metaFields {
		fieldSchema := &schema.Schema{
			Optional: true,
			ForceNew: true,
			// TODO: Fields that arent in configuration shouldnt show up in resource data
			// ConflictsWith: []string{f.NameInDynamicForFeed},
		}
		f.SchemaBuilder(fieldSchema)

		fields[f.NameInDynamic] = fieldSchema

		// Add an "_feed"-suffixed field for the {"feed":...} value.
		fields[f.NameInDynamicForFeed] = &schema.Schema{
			Optional: true,
			ForceNew: true,
			// TODO: Fields that arent in configuration shouldnt show up in resource data
			// ConflictsWith: []string{f.NameInDynamic},
			Type: schema.TypeString,
		}
	}

	metaSchemaInner := &schema.Resource{
		Schema: fields,
	}

	// Wrap it in a list because that seems to be the only way to have nested structs.
	return &schema.Schema{
		Type:     schema.TypeList,
		Optional: true,
		MaxItems: 1,
		Elem:     metaSchemaInner,
	}
}

var metaSchema *schema.Schema = makeMetaSchema()

func metaStructToDynamic(m *data.Meta) interface{} {
	d := make(map[string]interface{})
	mr := reflect.ValueOf(m).Elem()
	for _, f := range metaFields {
		fr := mr.Field(f.StructIndex)
		fv := fr.Interface()

		if fv == nil {
			continue
		}

		if mapVal, isMap := fv.(map[string]interface{}); isMap {
			if len(mapVal) == 1 {
				if feedVal, ok := mapVal["feed"]; ok {
					if feedStr, ok := feedVal.(string); ok {
						d[f.NameInDynamicForFeed] = feedStr
						continue
					}
				}
			}
			panic(fmt.Sprintf("expecting feed dict, got %+v", mapVal))
		}

		d[f.NameInDynamic] = fv
	}
	return []interface{}{d}
}

func metaDynamicToStruct(m *data.Meta, raw interface{}) {
	l := raw.([]interface{})
	if len(l) > 1 {
		panic(fmt.Sprintf("list too long %#v", l))
	}
	if len(l) == 0 {
		return
	}
	if l[0] == nil {
		return
	}

	d := l[0].(map[string]interface{})

	mr := reflect.ValueOf(m).Elem()
	for _, f := range metaFields {
		val, present := d[f.NameInDynamic]
		if present {
			fr := mr.Field(f.StructIndex)
			fr.Set(reflect.ValueOf(val))
		}

		feed, present := d[f.NameInDynamicForFeed]
		if present && feed != "" {
			if feed == nil {
				panic("unexpected nil")
			}
			fr := mr.Field(f.StructIndex)
			fr.Set(reflect.ValueOf(map[string]interface{}{"feed": feed.(string)}))
		}
	}
}
