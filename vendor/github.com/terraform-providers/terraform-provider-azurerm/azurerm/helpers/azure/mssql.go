package azure

import (
	"fmt"

	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/helpers/validate"
)

//Your server name can contain only lowercase letters, numbers, and '-', but can't start or end with '-' or have more than 63 characters.
func ValidateMsSqlServerName(i interface{}, k string) (_ []string, errors []error) {
	if m, regexErrs := validate.RegExHelper(i, k, `^[0-9a-z]([-0-9a-z]{0,61}[0-9a-z])?$`); !m {
		errors = append(regexErrs, fmt.Errorf("%q can contain only lowercase letters, numbers, and '-', but can't start or end with '-' or have more than 63 characters.", k))
	}

	return nil, errors
}

//Your database name can't end with '.' or ' ', can't contain '<,>,*,%,&,:,\,/,?' or control characters, and can't have more than 128 characters.
func ValidateMsSqlDatabaseName(i interface{}, k string) (_ []string, errors []error) {
	if m, regexErrs := validate.RegExHelper(i, k, `^[^<>*%&:\\\/?]{0,127}[^\s.<>*%&:\\\/?]$`); !m {
		errors = append(regexErrs, fmt.Errorf(`%q can't end with '.' or ' ', can't contain '<,>,*,%%,&,:,\,/,?' or control characters, and can't have more than 128 characters.`, k))
	}

	return nil, errors
}

//Following characters and any control characters are not allowed for resource name '%,&,\\\\,?,/'.\"
//The name can not end with characters: '. '
//TODO: unsure about length, was able to deploy one at 120
func ValidateMsSqlElasticPoolName(i interface{}, k string) (_ []string, errors []error) {
	if m, regexErrs := validate.RegExHelper(i, k, `^[^&%\\\/?]{0,127}[^\s.&%\\\/?]$`); !m {
		errors = append(regexErrs, fmt.Errorf(`%q can't end with '.' or ' ', can't contain '%%,&,\,/,?' or control characters, and can't have more than 128 characters.`, k))
	}

	return nil, errors
}
