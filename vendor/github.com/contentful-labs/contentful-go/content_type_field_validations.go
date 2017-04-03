package contentful

import (
	"encoding/json"
	"time"
)

// FieldValidation interface
type FieldValidation interface{}

// FieldValidationLink model
type FieldValidationLink struct {
	LinkContentType []string `json:"linkContentType,omitempty"`
}

const (
	MimeTypeAttachment   = "attachment"
	MimeTypePlainText    = "plaintext"
	MimeTypeImage        = "image"
	MimeTypeAudio        = "audio"
	MimeTypeVideo        = "video"
	MimeTypeRichText     = "richtext"
	MimeTypePresentation = "presentation"
	MimeTypeSpreadSheet  = "spreadsheet"
	MimeTypePDF          = "pdfdocument"
	MimeTypeArchive      = "archive"
	MimeTypeCode         = "code"
	MimeTypeMarkup       = "markup"
)

// FieldValidationMimeType model
type FieldValidationMimeType struct {
	MimeTypes []string `json:"linkMimetypeGroup,omitempty"`
}

// MinMax model
type MinMax struct {
	Min float64 `json:"min,omitempty"`
	Max float64 `json:"max,omitempty"`
}

// DateMinMax model
type DateMinMax struct {
	Min time.Time `json:"min,omitempty"`
	Max time.Time `json:"max,omitempty"`
}

// FieldValidationDimension model
type FieldValidationDimension struct {
	Width        *MinMax `json:"width,omitempty"`
	Height       *MinMax `json:"height,omitempty"`
	ErrorMessage string  `json:"message,omitempty"`
}

// MarshalJSON for custom json marshaling
func (v *FieldValidationDimension) MarshalJSON() ([]byte, error) {
	type dimension struct {
		Width  *MinMax `json:"width,omitempty"`
		Height *MinMax `json:"height,omitempty"`
	}

	return json.Marshal(&struct {
		AssetImageDimensions *dimension `json:"assetImageDimensions,omitempty"`
		Message              string     `json:"message,omitempty"`
	}{
		AssetImageDimensions: &dimension{
			Width:  v.Width,
			Height: v.Height,
		},
		Message: v.ErrorMessage,
	})
}

// UnmarshalJSON for custom json unmarshaling
func (v *FieldValidationDimension) UnmarshalJSON(data []byte) error {
	payload := map[string]interface{}{}
	if err := json.Unmarshal(data, &payload); err != nil {
		return err
	}

	dimensionData := payload["assetImageDimensions"].(map[string]interface{})

	if width, ok := dimensionData["width"].(map[string]interface{}); ok {
		v.Width = &MinMax{}

		if min, ok := width["min"].(float64); ok {
			v.Width.Min = min
		}

		if max, ok := width["min"].(float64); ok {
			v.Width.Max = max
		}
	}

	if height, ok := dimensionData["height"].(map[string]interface{}); ok {
		v.Height = &MinMax{}

		if min, ok := height["min"].(float64); ok {
			v.Height.Min = min
		}

		if max, ok := height["max"].(float64); ok {
			v.Height.Max = max
		}
	}

	if val, ok := payload["message"].(string); ok {
		v.ErrorMessage = val
	}

	return nil
}

// FieldValidationFileSize model
type FieldValidationFileSize struct {
	Size         *MinMax `json:"assetFileSize,omitempty"`
	ErrorMessage string  `json:"message,omitempty"`
}

// FieldValidationUnique model
type FieldValidationUnique struct {
	Unique bool `json:"unique"`
}

// FieldValidationPredefinedValues model
type FieldValidationPredefinedValues struct {
	In           []interface{} `json:"in,omitempty"`
	ErrorMessage string        `json:"message"`
}

// FieldValidationRange model
type FieldValidationRange struct {
	Range        *MinMax `json:"range,omitempty"`
	ErrorMessage string  `json:"message,omitempty"`
}

// FieldValidationDate model
type FieldValidationDate struct {
	Range        *DateMinMax `json:"dateRange,omitempty"`
	ErrorMessage string      `json:"message,omitempty"`
}

// MarshalJSON for custom json marshaling
func (v *FieldValidationDate) MarshalJSON() ([]byte, error) {
	type dateRange struct {
		Min string `json:"min,omitempty"`
		Max string `json:"max,omitempty"`
	}

	return json.Marshal(&struct {
		DateRange *dateRange `json:"dateRange,omitempty"`
		Message   string     `json:"message,omitempty"`
	}{
		DateRange: &dateRange{
			Min: v.Range.Max.Format("2006-01-02T03:04:05"),
			Max: v.Range.Max.Format("2006-01-02T03:04:05"),
		},
		Message: v.ErrorMessage,
	})
}

// UnmarshalJSON for custom json unmarshaling
func (v *FieldValidationDate) UnmarshalJSON(data []byte) error {
	payload := map[string]interface{}{}
	if err := json.Unmarshal(data, &payload); err != nil {
		return err
	}

	dateRangeData := payload["dateRange"].(map[string]interface{})

	v.Range = &DateMinMax{}

	if min, ok := dateRangeData["min"].(string); ok {
		minDate, err := time.Parse("2006-01-02T03:04:05", min)
		if err != nil {
			return err
		}

		v.Range.Min = minDate
	}

	if max, ok := dateRangeData["max"].(string); ok {
		maxDate, err := time.Parse("2006-01-02T03:04:05", max)
		if err != nil {
			return err
		}

		v.Range.Max = maxDate
	}

	if val, ok := payload["message"].(string); ok {
		v.ErrorMessage = val
	}

	return nil
}

// FieldValidationSize model
type FieldValidationSize struct {
	Size         *MinMax `json:"size,omitempty"`
	ErrorMessage string  `json:"message,omitempty"`
}

const (
	FieldValidationRegexPatternEmail         = `^\w[\w.-]*@([\w-]+\.)+[\w-]+$`
	FieldValidationRegexPatternURL           = `^(ftp|http|https):\/\/(\w+:{0,1}\w*@)?(\S+)(:[0-9]+)?(\/|\/([\w#!:.?+=&%@!\-\/]))?$`
	FieldValidationRegexPatternUSDate        = `^(0?[1-9]|[12][0-9]|3[01])[- \/.](0?[1-9]|1[012])[- \/.](19|20)?\d\d$`
	FieldValidationRegexPatternEuorpeanDate  = `^(0?[1-9]|[12][0-9]|3[01])[- \/.](0?[1-9]|1[012])[- \/.](19|20)?\d\d$`
	FieldValidationRegexPattern12HourTime    = `^(0?[1-9]|1[012]):[0-5][0-9](:[0-5][0-9])?\s*[aApP][mM]$`
	FieldValidationRegexPattern24HourTime    = `^(0?[0-9]|1[0-9]|2[0-3]):[0-5][0-9](:[0-5][0-9])?$`
	FieldValidationRegexPatternUSPhoneNumber = `^\d[ -.]?\(?\d\d\d\)?[ -.]?\d\d\d[ -.]?\d\d\d\d$`
	FieldValidationRegexPatternUSZipCode     = `^\d{5}$|^\d{5}-\d{4}$}`
)

// Regex model
type Regex struct {
	Pattern string `json:"pattern,omitempty"`
	Flags   string `json:"flags,omitempty"`
}

// FieldValidationRegex model
type FieldValidationRegex struct {
	Regex        *Regex `json:"regexp,omitempty"`
	ErrorMessage string `json:"message,omitempty"`
}
