package clevercloud

import (
	"fmt"
	"sort"
	"strings"

	"github.com/hashicorp/terraform/helper/schema"
)

func resourceCleverCloudApplicationJava() *schema.Resource {
	javaRuntimes := []string{"sbt", "play1", "play2", "maven", "war", "gradle", "jar"}
	sort.Strings(javaRuntimes)
	validateJavaRuntimes := func(v interface{}, k string) (ws []string, es []error) {
		runtime := strings.ToLower(v.(string))
		if i := sort.SearchStrings(javaRuntimes, runtime); i >= len(javaRuntimes) {
			es = append(es, fmt.Errorf(runtime+" is not available as java type instance runtime"))
		}
		return
	}

	tpl := resourceCleverCloudApplication(
		"java",
		[]string{"par", "mtl"},
		[]string{"git"},
		[]string{"pico", "nano", "xs", "s", "m", "l", "xl"},
	)

	tpl.Schema["runtime"] = &schema.Schema{
		Type:         schema.TypeString,
		Required:     true,
		ForceNew:     true,
		ValidateFunc: validateJavaRuntimes,
	}

	return tpl
}
