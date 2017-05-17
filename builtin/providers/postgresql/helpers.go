package postgresql

import (
	"fmt"
	"strings"
)

// pqQuoteLiteral returns a string literal safe for inclusion in a PostgreSQL
// query as a parameter.  The resulting string still needs to be wrapped in
// single quotes in SQL (i.e. fmt.Sprintf(`'%s'`, pqQuoteLiteral("str"))).  See
// quote_literal_internal() in postgresql/backend/utils/adt/quote.c:77.
func pqQuoteLiteral(in string) string {
	in = strings.Replace(in, `\`, `\\`, -1)
	in = strings.Replace(in, `'`, `''`, -1)
	return in
}

func validateConnLimit(v interface{}, key string) (warnings []string, errors []error) {
	value := v.(int)
	if value < -1 {
		errors = append(errors, fmt.Errorf("%s can not be less than -1", key))
	}
	return
}
