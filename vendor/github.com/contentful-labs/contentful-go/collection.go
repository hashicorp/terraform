package contentful

import (
	"bytes"
	"encoding/json"
	"net/http"
)

// CollectionOptions holds init options
type CollectionOptions struct {
	Limit uint16
}

// Collection model
type Collection struct {
	Query
	c     *Contentful
	req   *http.Request
	page  uint16
	Sys   *Sys          `json:"sys"`
	Total int           `json:"total"`
	Skip  int           `json:"skip"`
	Limit int           `json:"limit"`
	Items []interface{} `json:"items"`
}

// NewCollection initilazies a new collection
func NewCollection(options *CollectionOptions) *Collection {
	query := NewQuery()
	query.Order("sys.createdAt", true)

	if options.Limit > 0 {
		query.Limit(options.Limit)
	}

	return &Collection{
		Query: *query,
		page:  1,
	}
}

// Next makes the col.req
func (col *Collection) Next() (*Collection, error) {
	// setup query params
	skip := col.Query.limit * (col.page - 1)
	col.Query.Skip(skip)

	// override request query
	col.req.URL.RawQuery = col.Query.String()

	// makes api call
	err := col.c.do(col.req, col)
	if err != nil {
		return nil, err
	}

	col.page++

	return col, nil
}

// ToContentType cast Items to ContentType model
func (col *Collection) ToContentType() []*ContentType {
	var contentTypes []*ContentType

	byteArray, _ := json.Marshal(col.Items)
	json.NewDecoder(bytes.NewReader(byteArray)).Decode(&contentTypes)

	return contentTypes
}

// ToSpace cast Items to ContentType model
func (col *Collection) ToSpace() []*Space {
	var spaces []*Space

	byteArray, _ := json.Marshal(col.Items)
	json.NewDecoder(bytes.NewReader(byteArray)).Decode(&spaces)

	return spaces
}

// ToLocale cast Items to Locale model
func (col *Collection) ToLocale() []*Locale {
	var locales []*Locale

	byteArray, _ := json.Marshal(col.Items)
	json.NewDecoder(bytes.NewReader(byteArray)).Decode(&locales)

	return locales
}

// ToAsset cast Items to Asset model
func (col *Collection) ToAsset() []*Asset {
	var assets []*Asset

	byteArray, _ := json.Marshal(col.Items)
	json.NewDecoder(bytes.NewReader(byteArray)).Decode(&assets)

	return assets
}

// ToAPIKey cast Items to APIKey model
func (col *Collection) ToAPIKey() []*APIKey {
	var apiKeys []*APIKey

	byteArray, _ := json.Marshal(col.Items)
	json.NewDecoder(bytes.NewReader(byteArray)).Decode(&apiKeys)

	return apiKeys
}

// ToWebhook cast Items to Webhook model
func (col *Collection) ToWebhook() []*Webhook {
	var webhooks []*Webhook

	byteArray, _ := json.Marshal(col.Items)
	json.NewDecoder(bytes.NewReader(byteArray)).Decode(&webhooks)

	return webhooks
}
