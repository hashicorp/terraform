package template

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/hashicorp/hil"
	"github.com/hashicorp/hil/ast"
	"github.com/hashicorp/terraform/config"
	"github.com/hashicorp/terraform/helper/pathorcontents"
	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourceFile() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceFileRead,

		Schema: map[string]*schema.Schema{
			"template": &schema.Schema{
				Type:          schema.TypeString,
				Optional:      true,
				Description:   "Contents of the template",
				ConflictsWith: []string{"filename"},
				ValidateFunc:  validateTemplateAttribute,
			},
			"filename": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Description: "file to read template from",
				// Make a "best effort" attempt to relativize the file path.
				StateFunc: func(v interface{}) string {
					if v == nil || v.(string) == "" {
						return ""
					}
					pwd, err := os.Getwd()
					if err != nil {
						return v.(string)
					}
					rel, err := filepath.Rel(pwd, v.(string))
					if err != nil {
						return v.(string)
					}
					return rel
				},
				Deprecated:    "Use the 'template' attribute instead.",
				ConflictsWith: []string{"template"},
			},
			"vars": &schema.Schema{
				Type:         schema.TypeMap,
				Optional:     true,
				Default:      make(map[string]interface{}),
				Description:  "variables to substitute",
				ValidateFunc: validateVarsAttribute,
			},
			"rendered": &schema.Schema{
				Type:        schema.TypeString,
				Computed:    true,
				Description: "rendered template",
			},
		},
	}
}

func dataSourceFileRead(d *schema.ResourceData, meta interface{}) error {
	rendered, err := renderFile(d)
	if err != nil {
		return err
	}
	d.Set("rendered", rendered)
	d.SetId(hash(rendered))
	return nil
}

type templateRenderError error

func renderFile(d *schema.ResourceData) (string, error) {
	template := d.Get("template").(string)
	filename := d.Get("filename").(string)
	vars := d.Get("vars").(map[string]interface{})

	if template == "" && filename != "" {
		template = filename
	}

	contents, _, err := pathorcontents.Read(template)
	if err != nil {
		return "", err
	}

	rendered, err := execute(contents, vars)
	if err != nil {
		return "", templateRenderError(
			fmt.Errorf("failed to render %v: %v", filename, err),
		)
	}

	return rendered, nil
}

// execute parses and executes a template using vars.
func execute(s string, vars map[string]interface{}) (string, error) {
	root, err := hil.Parse(s)
	if err != nil {
		return "", err
	}

	varmap := make(map[string]ast.Variable)
	for k, v := range vars {
		// As far as I can tell, v is always a string.
		// If it's not, tell the user gracefully.
		s, ok := v.(string)
		if !ok {
			return "", fmt.Errorf("unexpected type for variable %q: %T", k, v)
		}

		// Store the defaults (string and value)
		var val interface{} = s
		typ := ast.TypeString

		// If we can parse a float, then use that
		if v, err := strconv.ParseFloat(s, 64); err == nil {
			val = v
			typ = ast.TypeFloat
		}

		varmap[k] = ast.Variable{
			Value: val,
			Type:  typ,
		}
	}

	cfg := hil.EvalConfig{
		GlobalScope: &ast.BasicScope{
			VarMap:  varmap,
			FuncMap: config.Funcs(),
		},
	}

	result, err := hil.Eval(root, &cfg)
	if err != nil {
		return "", err
	}
	if result.Type != hil.TypeString {
		return "", fmt.Errorf("unexpected output hil.Type: %v", result.Type)
	}

	return result.Value.(string), nil
}

func hash(s string) string {
	sha := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sha[:])
}

func validateTemplateAttribute(v interface{}, key string) (ws []string, es []error) {
	_, wasPath, err := pathorcontents.Read(v.(string))
	if err != nil {
		es = append(es, err)
		return
	}

	if wasPath {
		ws = append(ws, fmt.Sprintf("%s: looks like you specified a path instead of file contents. Use `file()` to load this path. Specifying a path directly is deprecated and will be removed in a future version.", key))
	}

	return
}

func validateVarsAttribute(v interface{}, key string) (ws []string, es []error) {
	// vars can only be primitives right now
	var badVars []string
	for k, v := range v.(map[string]interface{}) {
		switch v.(type) {
		case []interface{}:
			badVars = append(badVars, fmt.Sprintf("%s (list)", k))
		case map[string]interface{}:
			badVars = append(badVars, fmt.Sprintf("%s (map)", k))
		}
	}
	if len(badVars) > 0 {
		es = append(es, fmt.Errorf(
			"%s: cannot contain non-primitives; bad keys: %s",
			key, strings.Join(badVars, ", ")))
	}
	return
}
