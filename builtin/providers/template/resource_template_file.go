package template

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/hashicorp/hil"
	"github.com/hashicorp/hil/ast"
	"github.com/hashicorp/terraform/config"
	"github.com/hashicorp/terraform/helper/pathorcontents"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceFile() *schema.Resource {
	return &schema.Resource{
		Create: resourceFileCreate,
		Delete: resourceFileDelete,
		Exists: resourceFileExists,
		Read:   resourceFileRead,

		Schema: map[string]*schema.Schema{
			"template": &schema.Schema{
				Type:          schema.TypeString,
				Optional:      true,
				Description:   "Contents of the template",
				ForceNew:      true,
				ConflictsWith: []string{"filename"},
				ValidateFunc:  validateTemplateAttribute,
			},
			"filename": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Description: "file to read template from",
				ForceNew:    true,
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
				Type:        schema.TypeMap,
				Optional:    true,
				Default:     make(map[string]interface{}),
				Description: "variables to substitute",
				ForceNew:    true,
			},
			"rendered": &schema.Schema{
				Type:        schema.TypeString,
				Computed:    true,
				Description: "rendered template",
			},
		},
	}
}

func resourceFileCreate(d *schema.ResourceData, meta interface{}) error {
	rendered, err := renderFile(d)
	if err != nil {
		return err
	}
	d.Set("rendered", rendered)
	d.SetId(hash(rendered))
	return nil
}

func resourceFileDelete(d *schema.ResourceData, meta interface{}) error {
	d.SetId("")
	return nil
}

func resourceFileExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	rendered, err := renderFile(d)
	if err != nil {
		if _, ok := err.(templateRenderError); ok {
			log.Printf("[DEBUG] Got error while rendering in Exists: %s", err)
			log.Printf("[DEBUG] Returning false so the template re-renders using latest variables from config.")
			return false, nil
		} else {
			return false, err
		}
	}
	return hash(rendered) == d.Id(), nil
}

func resourceFileRead(d *schema.ResourceData, meta interface{}) error {
	// Logic is handled in Exists, which only returns true if the rendered
	// contents haven't changed. That means if we get here there's nothing to
	// do.
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
		varmap[k] = ast.Variable{
			Value: s,
			Type:  ast.TypeString,
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
