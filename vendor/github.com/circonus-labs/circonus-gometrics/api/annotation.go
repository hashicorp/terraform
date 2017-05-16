// Copyright 2016 Circonus, Inc. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Annotation API support - Fetch, Create, Update, Delete, and Search
// See: https://login.circonus.com/resources/api/calls/annotation

package api

import (
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"

	"github.com/circonus-labs/circonus-gometrics/api/config"
)

// Annotation defines a annotation. See https://login.circonus.com/resources/api/calls/annotation for more information.
type Annotation struct {
	Category       string   `json:"category"`                    // string
	CID            string   `json:"_cid,omitempty"`              // string
	Created        uint     `json:"_created,omitempty"`          // uint
	Description    string   `json:"description"`                 // string
	LastModified   uint     `json:"_last_modified,omitempty"`    // uint
	LastModifiedBy string   `json:"_last_modified_by,omitempty"` // string
	RelatedMetrics []string `json:"rel_metrics"`                 // [] len >= 0
	Start          uint     `json:"start"`                       // uint
	Stop           uint     `json:"stop"`                        // uint
	Title          string   `json:"title"`                       // string
}

// NewAnnotation returns a new Annotation (with defaults, if applicable)
func NewAnnotation() *Annotation {
	return &Annotation{}
}

// FetchAnnotation retrieves annotation with passed cid.
func (a *API) FetchAnnotation(cid CIDType) (*Annotation, error) {
	if cid == nil || *cid == "" {
		return nil, fmt.Errorf("Invalid annotation CID [none]")
	}

	annotationCID := string(*cid)

	matched, err := regexp.MatchString(config.AnnotationCIDRegex, annotationCID)
	if err != nil {
		return nil, err
	}
	if !matched {
		return nil, fmt.Errorf("Invalid annotation CID [%s]", annotationCID)
	}

	result, err := a.Get(annotationCID)
	if err != nil {
		return nil, err
	}

	if a.Debug {
		a.Log.Printf("[DEBUG] fetch annotation, received JSON: %s", string(result))
	}

	annotation := &Annotation{}
	if err := json.Unmarshal(result, annotation); err != nil {
		return nil, err
	}

	return annotation, nil
}

// FetchAnnotations retrieves all annotations available to the API Token.
func (a *API) FetchAnnotations() (*[]Annotation, error) {
	result, err := a.Get(config.AnnotationPrefix)
	if err != nil {
		return nil, err
	}

	var annotations []Annotation
	if err := json.Unmarshal(result, &annotations); err != nil {
		return nil, err
	}

	return &annotations, nil
}

// UpdateAnnotation updates passed annotation.
func (a *API) UpdateAnnotation(cfg *Annotation) (*Annotation, error) {
	if cfg == nil {
		return nil, fmt.Errorf("Invalid annotation config [nil]")
	}

	annotationCID := string(cfg.CID)

	matched, err := regexp.MatchString(config.AnnotationCIDRegex, annotationCID)
	if err != nil {
		return nil, err
	}
	if !matched {
		return nil, fmt.Errorf("Invalid annotation CID [%s]", annotationCID)
	}

	jsonCfg, err := json.Marshal(cfg)
	if err != nil {
		return nil, err
	}

	if a.Debug {
		a.Log.Printf("[DEBUG] update annotation, sending JSON: %s", string(jsonCfg))
	}

	result, err := a.Put(annotationCID, jsonCfg)
	if err != nil {
		return nil, err
	}

	annotation := &Annotation{}
	if err := json.Unmarshal(result, annotation); err != nil {
		return nil, err
	}

	return annotation, nil
}

// CreateAnnotation creates a new annotation.
func (a *API) CreateAnnotation(cfg *Annotation) (*Annotation, error) {
	if cfg == nil {
		return nil, fmt.Errorf("Invalid annotation config [nil]")
	}

	jsonCfg, err := json.Marshal(cfg)
	if err != nil {
		return nil, err
	}

	if a.Debug {
		a.Log.Printf("[DEBUG] create annotation, sending JSON: %s", string(jsonCfg))
	}

	result, err := a.Post(config.AnnotationPrefix, jsonCfg)
	if err != nil {
		return nil, err
	}

	annotation := &Annotation{}
	if err := json.Unmarshal(result, annotation); err != nil {
		return nil, err
	}

	return annotation, nil
}

// DeleteAnnotation deletes passed annotation.
func (a *API) DeleteAnnotation(cfg *Annotation) (bool, error) {
	if cfg == nil {
		return false, fmt.Errorf("Invalid annotation config [nil]")
	}

	return a.DeleteAnnotationByCID(CIDType(&cfg.CID))
}

// DeleteAnnotationByCID deletes annotation with passed cid.
func (a *API) DeleteAnnotationByCID(cid CIDType) (bool, error) {
	if cid == nil || *cid == "" {
		return false, fmt.Errorf("Invalid annotation CID [none]")
	}

	annotationCID := string(*cid)

	matched, err := regexp.MatchString(config.AnnotationCIDRegex, annotationCID)
	if err != nil {
		return false, err
	}
	if !matched {
		return false, fmt.Errorf("Invalid annotation CID [%s]", annotationCID)
	}

	_, err = a.Delete(annotationCID)
	if err != nil {
		return false, err
	}

	return true, nil
}

// SearchAnnotations returns annotations matching the specified
// search query and/or filter. If nil is passed for both parameters
// all annotations will be returned.
func (a *API) SearchAnnotations(searchCriteria *SearchQueryType, filterCriteria *SearchFilterType) (*[]Annotation, error) {
	q := url.Values{}

	if searchCriteria != nil && *searchCriteria != "" {
		q.Set("search", string(*searchCriteria))
	}

	if filterCriteria != nil && len(*filterCriteria) > 0 {
		for filter, criteria := range *filterCriteria {
			for _, val := range criteria {
				q.Add(filter, val)
			}
		}
	}

	if q.Encode() == "" {
		return a.FetchAnnotations()
	}

	reqURL := url.URL{
		Path:     config.AnnotationPrefix,
		RawQuery: q.Encode(),
	}

	result, err := a.Get(reqURL.String())
	if err != nil {
		return nil, fmt.Errorf("[ERROR] API call error %+v", err)
	}

	var annotations []Annotation
	if err := json.Unmarshal(result, &annotations); err != nil {
		return nil, err
	}

	return &annotations, nil
}
