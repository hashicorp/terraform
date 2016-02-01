package copystructure

import (
	"fmt"
)

func ExampleCopy() {
	input := map[string]interface{}{
		"bob": map[string]interface{}{
			"emails": []string{"a", "b"},
		},
	}

	dup, err := Copy(input)
	if err != nil {
		panic(err)
	}

	fmt.Printf("%#v", dup)
	// Output:
	// map[string]interface {}{"bob":map[string]interface {}{"emails":[]string{"a", "b"}}}
}
