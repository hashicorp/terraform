package contentful

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
)

// ContentTypesService service
type ContentTypesService service

// ContentType model
type ContentType struct {
	Sys          *Sys     `json:"sys"`
	Name         string   `json:"name,omitempty"`
	Description  string   `json:"description,omitempty"`
	Fields       []*Field `json:"fields,omitempty"`
	DisplayField string   `json:"displayField,omitempty"`
}

const (
	FieldTypeText     = "Text"
	FieldTypeArray    = "Array"
	FieldTypeLink     = "Link"
	FieldTypeInteger  = "Integer"
	FieldTypeLocation = "Location"
	FieldTypeBoolean  = "Boolean"
	FieldTypeDate     = "Date"
	FieldTypeObject   = "Object"
)

// Field model
type Field struct {
	ID          string              `json:"id,omitempty"`
	Name        string              `json:"name"`
	Type        string              `json:"type"`
	LinkType    string              `json:"linkType,omitempty"`
	Items       *FieldTypeArrayItem `json:"items,omitempty"`
	Required    bool                `json:"required,omitempty"`
	Localized   bool                `json:"localized,omitempty"`
	Disabled    bool                `json:"disabled,omitempty"`
	Omitted     bool                `json:"omitted,omitempty"`
	Validations []FieldValidation   `json:"validations,omitempty"`
}

// UnmarshalJSON for custom json unmarshaling
func (field *Field) UnmarshalJSON(data []byte) error {
	payload := map[string]interface{}{}
	if err := json.Unmarshal(data, &payload); err != nil {
		return err
	}

	if val, ok := payload["id"]; ok {
		field.ID = val.(string)
	}

	if val, ok := payload["name"]; ok {
		field.Name = val.(string)
	}

	if val, ok := payload["type"]; ok {
		field.Type = val.(string)
	}

	if val, ok := payload["linkType"]; ok {
		field.LinkType = val.(string)
	}

	if val, ok := payload["items"]; ok {
		byteArray, err := json.Marshal(val)
		if err != nil {
			return nil
		}

		var fieldTypeArrayItem FieldTypeArrayItem
		if err := json.Unmarshal(byteArray, &fieldTypeArrayItem); err != nil {
			return err
		}

		field.Items = &fieldTypeArrayItem
	}

	if val, ok := payload["required"]; ok {
		field.Required = val.(bool)
	}

	if val, ok := payload["localized"]; ok {
		field.Localized = val.(bool)
	}

	if val, ok := payload["disabled"]; ok {
		field.Disabled = val.(bool)
	}

	if val, ok := payload["omitted"]; ok {
		field.Omitted = val.(bool)
	}

	if val, ok := payload["validations"]; ok {
		validations, err := ParseValidations(val.([]interface{}))
		if err != nil {
			return err
		}

		field.Validations = validations
	}

	return nil
}

func ParseValidations(data []interface{}) (validations []FieldValidation, err error) {
	for _, value := range data {
		var validation map[string]interface{}
		var byteArray []byte

		if validationStr, ok := value.(string); ok {
			if err := json.Unmarshal([]byte(validationStr), &validation); err != nil {
				return nil, err
			}

			byteArray = []byte(validationStr)
		}

		if validationMap, ok := value.(map[string]interface{}); ok {
			byteArray, err = json.Marshal(validationMap)
			if err != nil {
				return nil, err
			}

			validation = validationMap
		}

		if _, ok := validation["linkContentType"]; ok {
			var fieldValidationLink FieldValidationLink
			if err := json.Unmarshal(byteArray, &fieldValidationLink); err != nil {
				return nil, err
			}

			validations = append(validations, fieldValidationLink)
		}

		if _, ok := validation["linkMimetypeGroup"]; ok {
			var fieldValidationMimeType FieldValidationMimeType
			if err := json.Unmarshal(byteArray, &fieldValidationMimeType); err != nil {
				return nil, err
			}

			validations = append(validations, fieldValidationMimeType)
		}

		if _, ok := validation["assetImageDimensions"]; ok {
			var fieldValidationDimension FieldValidationDimension
			if err := json.Unmarshal(byteArray, &fieldValidationDimension); err != nil {
				return nil, err
			}

			validations = append(validations, fieldValidationDimension)
		}

		if _, ok := validation["assetFileSize"]; ok {
			var fieldValidationFileSize FieldValidationFileSize
			if err := json.Unmarshal(byteArray, &fieldValidationFileSize); err != nil {
				return nil, err
			}

			validations = append(validations, fieldValidationFileSize)
		}

		if _, ok := validation["unique"]; ok {
			var fieldValidationUnique FieldValidationUnique
			if err := json.Unmarshal(byteArray, &fieldValidationUnique); err != nil {
				return nil, err
			}

			validations = append(validations, fieldValidationUnique)
		}

		if _, ok := validation["in"]; ok {
			var fieldValidationPredefinedValues FieldValidationPredefinedValues
			if err := json.Unmarshal(byteArray, &fieldValidationPredefinedValues); err != nil {
				return nil, err
			}

			validations = append(validations, fieldValidationPredefinedValues)
		}

		if _, ok := validation["range"]; ok {
			var fieldValidationRange FieldValidationRange
			if err := json.Unmarshal(byteArray, &fieldValidationRange); err != nil {
				return nil, err
			}

			validations = append(validations, fieldValidationRange)
		}

		if _, ok := validation["dateRange"]; ok {
			var fieldValidationDate FieldValidationDate
			if err := json.Unmarshal(byteArray, &fieldValidationDate); err != nil {
				return nil, err
			}

			validations = append(validations, fieldValidationDate)
		}

		if _, ok := validation["size"]; ok {
			var fieldValidationSize FieldValidationSize
			if err := json.Unmarshal(byteArray, &fieldValidationSize); err != nil {
				return nil, err
			}

			validations = append(validations, fieldValidationSize)
		}

		if _, ok := validation["regexp"]; ok {
			var fieldValidationRegex FieldValidationRegex
			if err := json.Unmarshal(byteArray, &fieldValidationRegex); err != nil {
				return nil, err
			}

			validations = append(validations, fieldValidationRegex)
		}
	}

	return validations, nil
}

// FieldTypeArrayItem model
type FieldTypeArrayItem struct {
	Type        string            `json:"type,omitempty"`
	Validations []FieldValidation `json:"validations,omitempty"`
	LinkType    string            `json:"linkType,omitempty"`
}

// UnmarshalJSON for custom json unmarshaling
func (item *FieldTypeArrayItem) UnmarshalJSON(data []byte) error {
	payload := map[string]interface{}{}
	if err := json.Unmarshal(data, &payload); err != nil {
		return err
	}

	if val, ok := payload["type"]; ok {
		item.Type = val.(string)
	}

	if val, ok := payload["validations"]; ok {
		validations, err := ParseValidations(val.([]interface{}))
		if err != nil {
			return err
		}

		item.Validations = validations
	}

	if val, ok := payload["linktype"]; ok {
		item.LinkType = val.(string)
	}

	return nil
}

// GetVersion returns entity version
func (ct *ContentType) GetVersion() int {
	version := 1
	if ct.Sys != nil {
		version = ct.Sys.Version
	}

	return version
}

// List return a content type collection
func (service *ContentTypesService) List(spaceID string) *Collection {
	path := fmt.Sprintf("/spaces/%s/content_types", spaceID)
	method := "GET"

	req, err := service.c.newRequest(method, path, nil, nil)
	if err != nil {
		return nil
	}

	col := NewCollection(&CollectionOptions{})
	col.c = service.c
	col.req = req

	return col
}

func (service *ContentTypesService) Get(spaceID, contentTypeID string) (*ContentType, error) {
	path := fmt.Sprintf("/spaces/%s/content_types/%s", spaceID, contentTypeID)
	method := "GET"

	req, err := service.c.newRequest(method, path, nil, nil)
	if err != nil {
		return nil, err
	}

	var ct ContentType
	if err = service.c.do(req, &ct); err != nil {
		return nil, err
	}

	return &ct, nil
}

// Upsert updates or creates a new content type
func (service *ContentTypesService) Upsert(spaceID string, ct *ContentType) error {
	bytesArray, err := json.Marshal(ct)
	if err != nil {
		return err
	}

	var path string
	var method string

	if ct.Sys != nil && ct.Sys.CreatedAt != "" {
		path = fmt.Sprintf("/spaces/%s/content_types/%s", spaceID, ct.Sys.ID)
		method = "PUT"
	} else {
		path = fmt.Sprintf("/spaces/%s/content_types", spaceID)
		method = "POST"
	}

	req, err := service.c.newRequest(method, path, nil, bytes.NewReader(bytesArray))
	if err != nil {
		return err
	}

	req.Header.Set("X-Contentful-Version", strconv.Itoa(ct.GetVersion()))

	if err = service.c.do(req, ct); err != nil {
		return err
	}

	return nil
}

// Delete the content_type
func (service *ContentTypesService) Delete(spaceID string, ct *ContentType) error {
	path := fmt.Sprintf("/spaces/%s/content_types/%s", spaceID, ct.Sys.ID)
	method := "DELETE"

	req, err := service.c.newRequest(method, path, nil, nil)
	if err != nil {
		return err
	}

	version := strconv.Itoa(ct.Sys.Version)
	req.Header.Set("X-Contentful-Version", version)

	if err = service.c.do(req, nil); err != nil {
		return err
	}

	return nil
}

// Activate the contenttype, a.k.a publish
func (service *ContentTypesService) Activate(spaceID string, ct *ContentType) error {
	path := fmt.Sprintf("/spaces/%s/content_types/%s/published", spaceID, ct.Sys.ID)
	method := "PUT"

	req, err := service.c.newRequest(method, path, nil, nil)
	if err != nil {
		return err
	}

	version := strconv.Itoa(ct.Sys.Version)
	req.Header.Set("X-Contentful-Version", version)

	if err = service.c.do(req, ct); err != nil {
		return err
	}

	return nil
}

// Deactivate the contenttype, a.k.a unpublish
func (service *ContentTypesService) Deactivate(spaceID string, ct *ContentType) error {
	path := fmt.Sprintf("/spaces/%s/content_types/%s/published", spaceID, ct.Sys.ID)
	method := "DELETE"

	req, err := service.c.newRequest(method, path, nil, nil)
	if err != nil {
		return err
	}

	version := strconv.Itoa(ct.Sys.Version)
	req.Header.Set("X-Contentful-Version", version)

	if err = service.c.do(req, ct); err != nil {
		fmt.Println(err)
		return err
	}

	return nil
}
