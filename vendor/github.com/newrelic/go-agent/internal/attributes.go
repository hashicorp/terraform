package internal

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
)

// New agent attributes must be added in the following places:
// * Constants here.
// * Top level attributes.go file.
// * agentAttributes
// * agentAttributeDests
// * calculateAgentAttributeDests
// * writeAgentAttributes
const (
	responseCode          = "httpResponseCode"
	requestMethod         = "request.method"
	requestAccept         = "request.headers.accept"
	requestContentType    = "request.headers.contentType"
	requestContentLength  = "request.headers.contentLength"
	requestHost           = "request.headers.host"
	responseContentType   = "response.headers.contentType"
	responseContentLength = "response.headers.contentLength"
	hostDisplayName       = "host.displayName"
	requestUserAgent      = "request.headers.User-Agent"
	requestReferer        = "request.headers.referer"
)

// https://source.datanerd.us/agents/agent-specs/blob/master/Agent-Attributes-PORTED.md

// AttributeDestinationConfig matches newrelic.AttributeDestinationConfig to
// avoid circular dependency issues.
type AttributeDestinationConfig struct {
	Enabled bool
	Include []string
	Exclude []string
}

type destinationSet int

const (
	destTxnEvent destinationSet = 1 << iota
	destError
	destTxnTrace
	destBrowser
)

const (
	destNone destinationSet = 0
	// DestAll contains all destinations.
	DestAll destinationSet = destTxnEvent | destTxnTrace | destError | destBrowser
)

const (
	attributeWildcardSuffix = '*'
)

type attributeModifier struct {
	match string // This will not contain a trailing '*'.
	includeExclude
}

type byMatch []*attributeModifier

func (m byMatch) Len() int           { return len(m) }
func (m byMatch) Swap(i, j int)      { m[i], m[j] = m[j], m[i] }
func (m byMatch) Less(i, j int) bool { return m[i].match < m[j].match }

// AttributeConfig is created at application creation and shared between all
// transactions.
type AttributeConfig struct {
	disabledDestinations destinationSet
	exactMatchModifiers  map[string]*attributeModifier
	// Once attributeConfig is constructed, wildcardModifiers is sorted in
	// lexicographical order.  Modifiers appearing later have precedence
	// over modifiers appearing earlier.
	wildcardModifiers []*attributeModifier
	agentDests        agentAttributeDests
}

type includeExclude struct {
	include destinationSet
	exclude destinationSet
}

func modifierApply(m *attributeModifier, d destinationSet) destinationSet {
	// Include before exclude, since exclude has priority.
	d |= m.include
	d &^= m.exclude
	return d
}

func applyAttributeConfig(c *AttributeConfig, key string, d destinationSet) destinationSet {
	// Important: The wildcard modifiers must be applied before the exact
	// match modifiers, and the slice must be iterated in a forward
	// direction.
	for _, m := range c.wildcardModifiers {
		if strings.HasPrefix(key, m.match) {
			d = modifierApply(m, d)
		}
	}

	if m, ok := c.exactMatchModifiers[key]; ok {
		d = modifierApply(m, d)
	}

	d &^= c.disabledDestinations

	return d
}

func addModifier(c *AttributeConfig, match string, d includeExclude) {
	if "" == match {
		return
	}
	exactMatch := true
	if attributeWildcardSuffix == match[len(match)-1] {
		exactMatch = false
		match = match[0 : len(match)-1]
	}
	mod := &attributeModifier{
		match:          match,
		includeExclude: d,
	}

	if exactMatch {
		if m, ok := c.exactMatchModifiers[mod.match]; ok {
			m.include |= mod.include
			m.exclude |= mod.exclude
		} else {
			c.exactMatchModifiers[mod.match] = mod
		}
	} else {
		for _, m := range c.wildcardModifiers {
			// Important: Duplicate entries for the same match
			// string would not work because exclude needs
			// precedence over include.
			if m.match == mod.match {
				m.include |= mod.include
				m.exclude |= mod.exclude
				return
			}
		}
		c.wildcardModifiers = append(c.wildcardModifiers, mod)
	}
}

func processDest(c *AttributeConfig, dc *AttributeDestinationConfig, d destinationSet) {
	if !dc.Enabled {
		c.disabledDestinations |= d
	}
	for _, match := range dc.Include {
		addModifier(c, match, includeExclude{include: d})
	}
	for _, match := range dc.Exclude {
		addModifier(c, match, includeExclude{exclude: d})
	}
}

// AttributeConfigInput is used as the input to CreateAttributeConfig:  it
// transforms newrelic.Config settings into an AttributeConfig.
type AttributeConfigInput struct {
	Attributes        AttributeDestinationConfig
	ErrorCollector    AttributeDestinationConfig
	TransactionEvents AttributeDestinationConfig
	browserMonitoring AttributeDestinationConfig
	TransactionTracer AttributeDestinationConfig
}

var (
	sampleAttributeConfigInput = AttributeConfigInput{
		Attributes:        AttributeDestinationConfig{Enabled: true},
		ErrorCollector:    AttributeDestinationConfig{Enabled: true},
		TransactionEvents: AttributeDestinationConfig{Enabled: true},
		TransactionTracer: AttributeDestinationConfig{Enabled: true},
	}
)

// CreateAttributeConfig creates a new AttributeConfig.
func CreateAttributeConfig(input AttributeConfigInput) *AttributeConfig {
	c := &AttributeConfig{
		exactMatchModifiers: make(map[string]*attributeModifier),
		wildcardModifiers:   make([]*attributeModifier, 0, 64),
	}

	processDest(c, &input.Attributes, DestAll)
	processDest(c, &input.ErrorCollector, destError)
	processDest(c, &input.TransactionEvents, destTxnEvent)
	processDest(c, &input.TransactionTracer, destTxnTrace)
	processDest(c, &input.browserMonitoring, destBrowser)

	sort.Sort(byMatch(c.wildcardModifiers))

	c.agentDests = calculateAgentAttributeDests(c)

	return c
}

type userAttribute struct {
	value interface{}
	dests destinationSet
}

// Attributes are key value pairs attached to the various collected data types.
type Attributes struct {
	config *AttributeConfig
	user   map[string]userAttribute
	Agent  agentAttributes
}

type agentAttributes struct {
	HostDisplayName              string
	RequestMethod                string
	RequestAcceptHeader          string
	RequestContentType           string
	RequestContentLength         int
	RequestHeadersHost           string
	RequestHeadersUserAgent      string
	RequestHeadersReferer        string
	ResponseHeadersContentType   string
	ResponseHeadersContentLength int
	ResponseCode                 string
}

type agentAttributeDests struct {
	HostDisplayName              destinationSet
	RequestMethod                destinationSet
	RequestAcceptHeader          destinationSet
	RequestContentType           destinationSet
	RequestContentLength         destinationSet
	RequestHeadersHost           destinationSet
	RequestHeadersUserAgent      destinationSet
	RequestHeadersReferer        destinationSet
	ResponseHeadersContentType   destinationSet
	ResponseHeadersContentLength destinationSet
	ResponseCode                 destinationSet
}

func calculateAgentAttributeDests(c *AttributeConfig) agentAttributeDests {
	usual := DestAll &^ destBrowser
	traces := destTxnTrace | destError
	return agentAttributeDests{
		HostDisplayName:              applyAttributeConfig(c, hostDisplayName, usual),
		RequestMethod:                applyAttributeConfig(c, requestMethod, usual),
		RequestAcceptHeader:          applyAttributeConfig(c, requestAccept, usual),
		RequestContentType:           applyAttributeConfig(c, requestContentType, usual),
		RequestContentLength:         applyAttributeConfig(c, requestContentLength, usual),
		RequestHeadersHost:           applyAttributeConfig(c, requestHost, usual),
		RequestHeadersUserAgent:      applyAttributeConfig(c, requestUserAgent, traces),
		RequestHeadersReferer:        applyAttributeConfig(c, requestReferer, traces),
		ResponseHeadersContentType:   applyAttributeConfig(c, responseContentType, usual),
		ResponseHeadersContentLength: applyAttributeConfig(c, responseContentLength, usual),
		ResponseCode:                 applyAttributeConfig(c, responseCode, usual),
	}
}

type agentAttributeWriter struct {
	jsonFieldsWriter
	d destinationSet
}

func (w *agentAttributeWriter) writeString(name string, val string, d destinationSet) {
	if "" != val && 0 != w.d&d {
		w.stringField(name, truncateStringValueIfLong(val))
	}
}

func (w *agentAttributeWriter) writeInt(name string, val int, d destinationSet) {
	if val >= 0 && 0 != w.d&d {
		w.intField(name, int64(val))
	}
}

func writeAgentAttributes(buf *bytes.Buffer, d destinationSet, values agentAttributes, dests agentAttributeDests) {
	w := &agentAttributeWriter{
		jsonFieldsWriter: jsonFieldsWriter{buf: buf},
		d:                d,
	}
	buf.WriteByte('{')
	w.writeString(hostDisplayName, values.HostDisplayName, dests.HostDisplayName)
	w.writeString(requestMethod, values.RequestMethod, dests.RequestMethod)
	w.writeString(requestAccept, values.RequestAcceptHeader, dests.RequestAcceptHeader)
	w.writeString(requestContentType, values.RequestContentType, dests.RequestContentType)
	w.writeInt(requestContentLength, values.RequestContentLength, dests.RequestContentLength)
	w.writeString(requestHost, values.RequestHeadersHost, dests.RequestHeadersHost)
	w.writeString(requestUserAgent, values.RequestHeadersUserAgent, dests.RequestHeadersUserAgent)
	w.writeString(requestReferer, values.RequestHeadersReferer, dests.RequestHeadersReferer)
	w.writeString(responseContentType, values.ResponseHeadersContentType, dests.ResponseHeadersContentType)
	w.writeInt(responseContentLength, values.ResponseHeadersContentLength, dests.ResponseHeadersContentLength)
	w.writeString(responseCode, values.ResponseCode, dests.ResponseCode)
	buf.WriteByte('}')
}

// NewAttributes creates a new Attributes.
func NewAttributes(config *AttributeConfig) *Attributes {
	return &Attributes{
		config: config,
		Agent: agentAttributes{
			RequestContentLength:         -1,
			ResponseHeadersContentLength: -1,
		},
	}
}

// ErrInvalidAttribute is returned when the value is not valid.
type ErrInvalidAttribute struct{ typeString string }

func (e ErrInvalidAttribute) Error() string {
	return fmt.Sprintf("attribute value type %s is invalid", e.typeString)
}

func valueIsValid(val interface{}) error {
	switch val.(type) {
	case string, bool, nil,
		uint8, uint16, uint32, uint64, int8, int16, int32, int64,
		float32, float64, uint, int, uintptr:
		return nil
	default:
		return ErrInvalidAttribute{
			typeString: fmt.Sprintf("%T", val),
		}
	}
}

type invalidAttributeKeyErr struct{ key string }

func (e invalidAttributeKeyErr) Error() string {
	return fmt.Sprintf("attribute key '%.32s...' exceeds length limit %d",
		e.key, attributeKeyLengthLimit)
}

type userAttributeLimitErr struct{ key string }

func (e userAttributeLimitErr) Error() string {
	return fmt.Sprintf("attribute '%s' discarded: limit of %d reached", e.key,
		attributeUserLimit)
}

func validAttributeKey(key string) error {
	// Attributes whose keys are excessively long are dropped rather than
	// truncated to avoid worrying about the application of configuration to
	// truncated values or performing the truncation after configuration.
	if len(key) > attributeKeyLengthLimit {
		return invalidAttributeKeyErr{key: key}
	}
	return nil
}

func truncateStringValueIfLong(val string) string {
	if len(val) > attributeValueLengthLimit {
		return StringLengthByteLimit(val, attributeValueLengthLimit)
	}
	return val
}

func truncateStringValueIfLongInterface(val interface{}) interface{} {
	if str, ok := val.(string); ok {
		val = interface{}(truncateStringValueIfLong(str))
	}
	return val
}

// AddUserAttribute adds a user attribute.
func AddUserAttribute(a *Attributes, key string, val interface{}, d destinationSet) error {
	val = truncateStringValueIfLongInterface(val)
	if err := valueIsValid(val); nil != err {
		return err
	}
	if err := validAttributeKey(key); nil != err {
		return err
	}
	dests := applyAttributeConfig(a.config, key, d)
	if destNone == dests {
		return nil
	}
	if nil == a.user {
		a.user = make(map[string]userAttribute)
	}

	if _, exists := a.user[key]; !exists && len(a.user) >= attributeUserLimit {
		return userAttributeLimitErr{key}
	}

	// Note: Duplicates are overridden: last attribute in wins.
	a.user[key] = userAttribute{
		value: val,
		dests: dests,
	}
	return nil
}

func writeAttributeValueJSON(w *jsonFieldsWriter, key string, val interface{}) {
	switch v := val.(type) {
	case nil:
		w.rawField(key, `null`)
	case string:
		w.stringField(key, v)
	case bool:
		if v {
			w.rawField(key, `true`)
		} else {
			w.rawField(key, `false`)
		}
	case uint8:
		w.intField(key, int64(v))
	case uint16:
		w.intField(key, int64(v))
	case uint32:
		w.intField(key, int64(v))
	case uint64:
		w.intField(key, int64(v))
	case uint:
		w.intField(key, int64(v))
	case uintptr:
		w.intField(key, int64(v))
	case int8:
		w.intField(key, int64(v))
	case int16:
		w.intField(key, int64(v))
	case int32:
		w.intField(key, int64(v))
	case int64:
		w.intField(key, v)
	case int:
		w.intField(key, int64(v))
	case float32:
		w.floatField(key, float64(v))
	case float64:
		w.floatField(key, v)
	default:
		w.stringField(key, fmt.Sprintf("%T", v))
	}
}

type agentAttributesJSONWriter struct {
	attributes *Attributes
	dest       destinationSet
}

func (w agentAttributesJSONWriter) WriteJSON(buf *bytes.Buffer) {
	if nil == w.attributes {
		buf.WriteString("{}")
		return
	}
	writeAgentAttributes(buf, w.dest, w.attributes.Agent, w.attributes.config.agentDests)
}

func agentAttributesJSON(a *Attributes, buf *bytes.Buffer, d destinationSet) {
	agentAttributesJSONWriter{
		attributes: a,
		dest:       d,
	}.WriteJSON(buf)
}

type userAttributesJSONWriter struct {
	attributes *Attributes
	dest       destinationSet
}

func (u userAttributesJSONWriter) WriteJSON(buf *bytes.Buffer) {
	buf.WriteByte('{')
	if nil != u.attributes {
		w := jsonFieldsWriter{buf: buf}
		for name, atr := range u.attributes.user {
			if 0 != atr.dests&u.dest {
				writeAttributeValueJSON(&w, name, atr.value)
			}
		}
	}
	buf.WriteByte('}')
}

func userAttributesJSON(a *Attributes, buf *bytes.Buffer, d destinationSet) {
	userAttributesJSONWriter{
		attributes: a,
		dest:       d,
	}.WriteJSON(buf)
}

func userAttributesStringJSON(a *Attributes, d destinationSet) JSONString {
	if nil == a {
		return JSONString("{}")
	}
	estimate := len(a.user) * 128
	buf := bytes.NewBuffer(make([]byte, 0, estimate))
	userAttributesJSON(a, buf, d)
	bs := buf.Bytes()
	return JSONString(bs)
}

func agentAttributesStringJSON(a *Attributes, d destinationSet) JSONString {
	if nil == a {
		return JSONString("{}")
	}
	estimate := 1024
	buf := bytes.NewBuffer(make([]byte, 0, estimate))
	agentAttributesJSON(a, buf, d)
	return JSONString(buf.Bytes())
}

func getUserAttributes(a *Attributes, d destinationSet) map[string]interface{} {
	v := make(map[string]interface{})
	json.Unmarshal([]byte(userAttributesStringJSON(a, d)), &v)
	return v
}

func getAgentAttributes(a *Attributes, d destinationSet) map[string]interface{} {
	v := make(map[string]interface{})
	json.Unmarshal([]byte(agentAttributesStringJSON(a, d)), &v)
	return v
}

// RequestAgentAttributes gathers agent attributes out of the request.
func RequestAgentAttributes(a *Attributes, r *http.Request) {
	a.Agent.RequestMethod = r.Method

	h := r.Header
	if nil == h {
		return
	}
	a.Agent.RequestAcceptHeader = h.Get("Accept")
	a.Agent.RequestContentType = h.Get("Content-Type")
	a.Agent.RequestHeadersHost = h.Get("Host")
	a.Agent.RequestHeadersUserAgent = h.Get("User-Agent")
	a.Agent.RequestHeadersReferer = SafeURLFromString(h.Get("Referer"))

	if cl := h.Get("Content-Length"); "" != cl {
		if x, err := strconv.Atoi(cl); nil == err {
			a.Agent.RequestContentLength = x
		}
	}
}

// ResponseHeaderAttributes gather agent attributes from the response headers.
func ResponseHeaderAttributes(a *Attributes, h http.Header) {
	if nil == h {
		return
	}
	a.Agent.ResponseHeadersContentType = h.Get("Content-Type")
	if val := h.Get("Content-Length"); "" != val {
		if x, err := strconv.Atoi(val); nil == err {
			a.Agent.ResponseHeadersContentLength = x
		}
	}
}

var (
	// statusCodeLookup avoids a strconv.Itoa call.
	statusCodeLookup = map[int]string{
		100: "100", 101: "101",
		200: "200", 201: "201", 202: "202", 203: "203", 204: "204", 205: "205", 206: "206",
		300: "300", 301: "301", 302: "302", 303: "303", 304: "304", 305: "305", 307: "307",
		400: "400", 401: "401", 402: "402", 403: "403", 404: "404", 405: "405", 406: "406",
		407: "407", 408: "408", 409: "409", 410: "410", 411: "411", 412: "412", 413: "413",
		414: "414", 415: "415", 416: "416", 417: "417", 418: "418", 428: "428", 429: "429",
		431: "431", 451: "451",
		500: "500", 501: "501", 502: "502", 503: "503", 504: "504", 505: "505", 511: "511",
	}
)

// ResponseCodeAttribute sets the response code agent attribute.
func ResponseCodeAttribute(a *Attributes, code int) {
	a.Agent.ResponseCode = statusCodeLookup[code]
	if a.Agent.ResponseCode == "" {
		a.Agent.ResponseCode = strconv.Itoa(code)
	}
}
