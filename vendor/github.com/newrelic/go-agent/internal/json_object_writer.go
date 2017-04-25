package internal

import (
	"bytes"

	"github.com/newrelic/go-agent/internal/jsonx"
)

type jsonWriter interface {
	WriteJSON(buf *bytes.Buffer)
}

type jsonFieldsWriter struct {
	buf        *bytes.Buffer
	needsComma bool
}

func (w *jsonFieldsWriter) addKey(key string) {
	if w.needsComma {
		w.buf.WriteByte(',')
	} else {
		w.needsComma = true
	}
	// defensively assume that the key needs escaping:
	jsonx.AppendString(w.buf, key)
	w.buf.WriteByte(':')
}

func (w *jsonFieldsWriter) stringField(key string, val string) {
	w.addKey(key)
	jsonx.AppendString(w.buf, val)
}

func (w *jsonFieldsWriter) intField(key string, val int64) {
	w.addKey(key)
	jsonx.AppendInt(w.buf, val)
}

func (w *jsonFieldsWriter) floatField(key string, val float64) {
	w.addKey(key)
	jsonx.AppendFloat(w.buf, val)
}

func (w *jsonFieldsWriter) rawField(key string, val JSONString) {
	w.addKey(key)
	w.buf.WriteString(string(val))
}

func (w *jsonFieldsWriter) writerField(key string, val jsonWriter) {
	w.addKey(key)
	val.WriteJSON(w.buf)
}
