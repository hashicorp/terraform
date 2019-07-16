package suppress

import (
	"encoding/xml"
	"io"
	"reflect"
	"strings"

	"github.com/hashicorp/terraform/helper/schema"
)

func XmlDiff(_, old, new string, _ *schema.ResourceData) bool {
	oldTokens, err := expandXmlTokensFromString(old)
	if err != nil {
		return false
	}

	newTokens, err := expandXmlTokensFromString(new)
	if err != nil {
		return false
	}

	return reflect.DeepEqual(oldTokens, newTokens)
}

// This function will extract all XML tokens from a string, but ignoring all white-space tokens
func expandXmlTokensFromString(input string) ([]xml.Token, error) {
	decoder := xml.NewDecoder(strings.NewReader(input))
	tokens := make([]xml.Token, 0)
	for {
		token, err := decoder.Token()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		if chars, ok := token.(xml.CharData); ok {
			text := string(chars)
			if strings.TrimSpace(text) == "" {
				continue
			}
		}
		tokens = append(tokens, xml.CopyToken(token))
	}
	return tokens, nil
}
