package validate

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/hashicorp/terraform/helper/schema"
)

func URLIsHTTPS(i interface{}, k string) (_ []string, errors []error) {
	return URLWithScheme([]string{"https"})(i, k)
}

func URLIsHTTPOrHTTPS(i interface{}, k string) (_ []string, errors []error) {
	return URLWithScheme([]string{"http", "https"})(i, k)
}

func URLWithScheme(validSchemes []string) schema.SchemaValidateFunc {
	return func(i interface{}, k string) (_ []string, errors []error) {
		v, ok := i.(string)
		if !ok {
			errors = append(errors, fmt.Errorf("expected type of %q to be string", k))
			return
		}

		if v == "" {
			errors = append(errors, fmt.Errorf("expected %q url to not be empty", k))
			return
		}

		u, err := url.Parse(v)
		if err != nil {
			errors = append(errors, fmt.Errorf("%q url is in an invalid format: %q (%+v)", k, v, err))
			return
		}

		if u.Host == "" {
			errors = append(errors, fmt.Errorf("%q url has no host: %q", k, v))
			return
		}

		for _, s := range validSchemes {
			if u.Scheme == s {
				return //last check so just return
			}
		}

		errors = append(errors, fmt.Errorf("expected %q url %q to have a schema of: %q", k, v, strings.Join(validSchemes, ",")))
		return
	}
}
