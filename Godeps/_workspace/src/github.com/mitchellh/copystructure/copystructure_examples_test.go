package copystructure

import (
	"fmt"
)

func ExampleCopy() {
	input := map[string]interface{}{
		"bob": map[string]interface{}{
			"name":   "bob",
			"emails": []string{"a", "b"},
		},
		"jane": map[string]interface{}{
			"name": "jane",
		},
	}

	dup, err := Copy(input)
	if err != nil {
		panic(err)
	}

	fmt.Printf("%#v", dup)
	// Output:
	// map[string]interface {}{"bob":map[string]interface {}{"name":"bob", "emails":[]string{"a", "b"}}, "jane":map[string]interface {}{"name":"jane"}}
}
