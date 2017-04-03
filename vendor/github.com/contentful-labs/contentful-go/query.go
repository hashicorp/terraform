package contentful

import (
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"time"
)

//Query model
type Query struct {
	include     uint16
	contentType string
	fields      []string
	e           map[string]interface{}
	ne          map[string]interface{}
	all         map[string][]string
	in          map[string][]string
	nin         map[string][]string
	exists      []string
	notExists   []string
	lt          map[string]interface{}
	lte         map[string]interface{}
	gt          map[string]interface{}
	gte         map[string]interface{}
	q           string
	match       map[string]string
	near        map[string]string
	within      map[string]string
	order       []string
	limit       uint16
	skip        uint16
	mime        string
	locale      string
}

//NewQuery initilazies a new query
func NewQuery() *Query {
	return &Query{
		include:     0,
		contentType: "",
		fields:      []string{},
		e:           make(map[string]interface{}),
		ne:          make(map[string]interface{}),
		all:         make(map[string][]string),
		in:          make(map[string][]string),
		nin:         make(map[string][]string),
		exists:      []string{},
		notExists:   []string{},
		lt:          make(map[string]interface{}),
		lte:         make(map[string]interface{}),
		gt:          make(map[string]interface{}),
		gte:         make(map[string]interface{}),
		q:           "",
		match:       make(map[string]string),
		near:        make(map[string]string),
		within:      make(map[string]string),
		order:       []string{},
		limit:       100,
		skip:        0,
		mime:        "",
		locale:      "",
	}
}

//Include query
func (q *Query) Include(include uint16) *Query {
	q.include = include
	return q
}

//ContentType query
func (q *Query) ContentType(ct string) *Query {
	q.contentType = ct
	return q
}

//Select query
func (q *Query) Select(fields []string) *Query {
	q.fields = fields
	return q
}

//Equal equality query
func (q *Query) Equal(field string, value interface{}) *Query {
	q.e[field] = value
	return q
}

//NotEqual [ne] query
func (q *Query) NotEqual(field string, value interface{}) *Query {
	q.ne[field] = value
	return q
}

//All [all] query
func (q *Query) All(field string, value []string) *Query {
	q.all[field] = value
	return q
}

//In [in] query
func (q *Query) In(field string, value []string) *Query {
	q.in[field] = value
	return q
}

//NotIn [nin] query
func (q *Query) NotIn(field string, value []string) *Query {
	q.nin[field] = value
	return q
}

//Exists [exists] query
func (q *Query) Exists(field string) *Query {
	q.exists = append(q.exists, field)
	return q
}

//NotExists [exists] query
func (q *Query) NotExists(field string) *Query {
	q.notExists = append(q.notExists, field)
	return q
}

//LessThan [lt] query
func (q *Query) LessThan(field string, value interface{}) *Query {
	q.lt[field] = value
	return q
}

//LessThanOrEqual [lte] query
func (q *Query) LessThanOrEqual(field string, value interface{}) *Query {
	q.lte[field] = value
	return q
}

//GreaterThan [gt] query
func (q *Query) GreaterThan(field string, value interface{}) *Query {
	q.gt[field] = value
	return q
}

//GreaterThanOrEqual [lte] query
func (q *Query) GreaterThanOrEqual(field string, value interface{}) *Query {
	q.gte[field] = value
	return q
}

//Query param
func (q *Query) Query(qStr string) *Query {
	q.q = qStr
	return q
}

//Match param
func (q *Query) Match(field, match string) *Query {
	q.match[field] = match
	return q
}

//Near param
func (q *Query) Near(field string, lat, lon int16) *Query {
	q.near[field] = strconv.Itoa(int(lat)) + "," + strconv.Itoa(int(lon))
	return q
}

//Within param
func (q *Query) Within(field string, lat1, lon1, lat2, lon2 int16) *Query {
	q.within[field] = strconv.Itoa(int(lat1)) + "," + strconv.Itoa(int(lon1)) + "," + strconv.Itoa(int(lat2)) + "," + strconv.Itoa(int(lon2))
	return q
}

//WithinRadius param
func (q *Query) WithinRadius(field string, lat1, lon1, radius int16) *Query {
	q.within[field] = strconv.Itoa(int(lat1)) + "," + strconv.Itoa(int(lon1)) + "," + strconv.Itoa(int(radius))
	return q
}

//Order param
func (q *Query) Order(field string, reverse bool) *Query {
	if reverse {
		q.order = append(q.order, "-"+field)
	} else {
		q.order = append(q.order, field)
	}

	return q
}

//Limit query
func (q *Query) Limit(limit uint16) *Query {
	q.limit = limit
	return q
}

//Skip query
func (q *Query) Skip(skip uint16) *Query {
	q.skip = skip
	return q
}

//MimeType query
func (q *Query) MimeType(mime string) *Query {
	q.mime = mime
	return q
}

//Locale query
func (q *Query) Locale(locale string) *Query {
	q.locale = locale
	return q
}

// Values constructs url.Values
func (q *Query) Values() url.Values {
	params := url.Values{}

	if q.include != 0 {
		if q.include > 10 {
			panic("include value should be between 0 and 10")
		}

		params.Set("include", strconv.Itoa(int(q.include)))
	}

	if q.contentType != "" {
		params.Set("content_type", q.contentType)
	}

	if len(q.fields) > 0 {
		if len(q.fields) > 100 {
			panic("You can select up to 100 properties for `select`")
		}

		for _, sel := range q.fields {
			if len(strings.Split(sel, ".")) > 2 {
				panic("you should provide at most 2 depth for `select`")
			}
		}

		if q.contentType == "" {
			panic("you should provide content_type parameter")
		}

		params.Set("select", strings.Join(q.fields, ","))
	}

	for k, v := range q.e {
		switch reflect.TypeOf(v).Kind() {
		case reflect.Int:
			{
				intV := reflect.ValueOf(v).Interface().(int)
				params.Set(k, strconv.Itoa(intV))
			}
		case reflect.String:
			{
				strV := reflect.ValueOf(v).Interface().(string)
				params.Set(k, strV)
			}
		}
	}

	for k, v := range q.ne {
		switch reflect.TypeOf(v).Kind() {
		case reflect.Int:
			{
				intV := reflect.ValueOf(v).Interface().(int)
				params.Set(k+"[ne]", strconv.Itoa(intV))
			}
		case reflect.String:
			{
				strV := reflect.ValueOf(v).Interface().(string)
				params.Set(k+"[ne]", strV)
			}
		}
	}

	for k, v := range q.all {
		params.Set(k+"[all]", strings.Join(v, ","))
	}

	for k, v := range q.in {
		params.Set(k+"[in]", strings.Join(v, ","))
	}

	for k, v := range q.nin {
		params.Set(k+"[nin]", strings.Join(v, ","))
	}

	for _, v := range q.exists {
		params.Set(v+"[exists]", "true")
	}

	for _, v := range q.notExists {
		params.Set(v+"[exists]", "false")
	}

	for k, v := range q.lt {
		switch reflect.TypeOf(v).Kind() {
		case reflect.Int:
			{
				intV := reflect.ValueOf(v).Interface().(int)
				params.Set(k+"[lt]", strconv.Itoa(intV))
			}
		case reflect.Struct:
			{
				timeV := reflect.ValueOf(v).Interface().(time.Time)
				params.Set(k+"[lt]", timeV.Format("2006-01-02 15:04:05"))
			}
		}
	}

	for k, v := range q.lte {
		switch reflect.TypeOf(v).Kind() {
		case reflect.Int:
			{
				intV := reflect.ValueOf(v).Interface().(int)
				params.Set(k+"[lte]", strconv.Itoa(intV))
			}
		case reflect.Struct:
			{
				timeV := reflect.ValueOf(v).Interface().(time.Time)
				params.Set(k+"[lte]", timeV.Format("2006-01-02 15:04:05"))
			}
		}
	}

	for k, v := range q.gt {
		switch reflect.TypeOf(v).Kind() {
		case reflect.Int:
			{
				intV := reflect.ValueOf(v).Interface().(int)
				params.Set(k+"[gt]", strconv.Itoa(intV))
			}
		case reflect.Struct:
			{
				timeV := reflect.ValueOf(v).Interface().(time.Time)
				params.Set(k+"[gt]", timeV.Format("2006-01-02 15:04:05"))
			}
		}
	}

	for k, v := range q.gte {
		switch reflect.TypeOf(v).Kind() {
		case reflect.Int:
			{
				intV := reflect.ValueOf(v).Interface().(int)
				params.Set(k+"[gte]", strconv.Itoa(intV))
			}
		case reflect.Struct:
			{
				timeV := reflect.ValueOf(v).Interface().(time.Time)
				params.Set(k+"[gte]", timeV.Format("2006-01-02 15:04:05"))
			}
		}
	}

	if q.q != "" {
		params.Set("query", q.q)
	}

	for k, v := range q.match {
		params.Set(k+"[match]", v)
	}

	for k, v := range q.near {
		params.Set(k+"[near]", v)
	}

	for k, v := range q.within {
		params.Set(k+"[within]", v)
	}

	if len(q.order) > 0 {
		// if q.contentType == "" {
		// panic("you should provide a content type for order queries")
		// }

		params.Set("order", strings.Join(q.order, ","))
	}

	if q.limit != 0 {
		if q.limit > 1000 {
			panic("limit value should be between 0 and 1000")
		}

		params.Set("limit", strconv.Itoa(int(q.limit)))
	}

	if q.skip != 0 {
		params.Set("skip", strconv.Itoa(int(q.skip)))
	}

	if q.mime != "" {
		params.Set("mimetype_group", q.mime)
	}

	if q.locale != "" {
		params.Set("locale", q.locale)
	}

	return params
}

func (q *Query) String() string {
	return q.Values().Encode()
}
