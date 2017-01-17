package rundeck

import (
	"fmt"
	"testing"
)

func TestUnmarshalJobDetail(t *testing.T) {
	testUnmarshalXML(t, []unmarshalTest{
		unmarshalTest{
			"with-config",
			`<job><uuid>baz</uuid><dispatch><rankOrder>ascending</rankOrder></dispatch></job>`,
			&JobDetail{},
			func (rv interface {}) error {
				v := rv.(*JobDetail)
				if v.ID != "baz" {
					return fmt.Errorf("got ID %s, but expecting baz", v.ID)
				}
				if v.Dispatch.RankOrder != "ascending" {
					return fmt.Errorf("Dispatch.RankOrder = \"%v\", but expecting \"ascending\"", v.Dispatch.RankOrder)
				}
				return nil
			},
		},
		unmarshalTest{
			"with-empty-config",
			`<JobPlugin type="foo-plugin"><configuration/></JobPlugin>`,
			&JobPlugin{},
			func (rv interface {}) error {
				v := rv.(*JobPlugin)
				if v.Type != "foo-plugin" {
					return fmt.Errorf("got Type %s, but expecting foo-plugin", v.Type)
				}
				if len(v.Config) != 0 {
					return fmt.Errorf("got %i Config values, but expecting 0", len(v.Config))
				}
				return nil
			},
		},
	})
}

func TestMarshalJobPlugin(t *testing.T) {
	testMarshalXML(t, []marshalTest{
		marshalTest{
			"with-config",
			JobPlugin{
				Type: "foo-plugin",
				Config: map[string]string{
					"woo": "foo",
					"bar": "baz",
				},
			},
			`<JobPlugin type="foo-plugin"><configuration><entry key="bar" value="baz"></entry><entry key="woo" value="foo"></entry></configuration></JobPlugin>`,
		},
		marshalTest{
			"with-empty-config",
			JobPlugin{
				Type: "foo-plugin",
				Config: map[string]string{},
			},
			`<JobPlugin type="foo-plugin"></JobPlugin>`,
		},
		marshalTest{
			"with-zero-value-config",
			JobPlugin{
				Type: "foo-plugin",
			},
			`<JobPlugin type="foo-plugin"></JobPlugin>`,
		},
	})
}

func TestUnmarshalJobPlugin(t *testing.T) {
	testUnmarshalXML(t, []unmarshalTest{
		unmarshalTest{
			"with-config",
			`<JobPlugin type="foo-plugin"><configuration><entry key="woo" value="foo"/><entry key="bar" value="baz"/></configuration></JobPlugin>`,
			&JobPlugin{},
			func (rv interface {}) error {
				v := rv.(*JobPlugin)
				if v.Type != "foo-plugin" {
					return fmt.Errorf("got Type %s, but expecting foo-plugin", v.Type)
				}
				if len(v.Config) != 2 {
					return fmt.Errorf("got %v Config values, but expecting 2", len(v.Config))
				}
				if v.Config["woo"] != "foo" {
					return fmt.Errorf("Config[\"woo\"] = \"%s\", but expecting \"foo\"", v.Config["woo"])
				}
				if v.Config["bar"] != "baz" {
					return fmt.Errorf("Config[\"bar\"] = \"%s\", but expecting \"baz\"", v.Config["bar"])
				}
				return nil
			},
		},
		unmarshalTest{
			"with-empty-config",
			`<JobPlugin type="foo-plugin"><configuration/></JobPlugin>`,
			&JobPlugin{},
			func (rv interface {}) error {
				v := rv.(*JobPlugin)
				if v.Type != "foo-plugin" {
					return fmt.Errorf("got Type %s, but expecting foo-plugin", v.Type)
				}
				if len(v.Config) != 0 {
					return fmt.Errorf("got %i Config values, but expecting 0", len(v.Config))
				}
				return nil
			},
		},
	})
}

func TestMarshalJobCommand(t *testing.T) {
	testMarshalXML(t, []marshalTest{
		marshalTest{
			"with-shell",
			JobCommand{
				ShellCommand: "command",
			},
			`<JobCommand><exec>command</exec></JobCommand>`,
		},
		marshalTest{
			"with-script",
			JobCommand{
				Script: "script",
			},
			`<JobCommand><script>script</script></JobCommand>`,
		},
		marshalTest{
			"with-script-interpreter",
			JobCommand{
				FileExtension: "sh",
				Script: "Hello World!",
			  ScriptInterpreter: &JobCommandScriptInterpreter{
						InvocationString: "sudo",
				},
			},
			`<JobCommand><fileExtension>sh</fileExtension><script>Hello World!</script><scriptinterpreter>sudo</scriptinterpreter></JobCommand>`,
		},
	})
}

func TestUnmarshalJobCommand(t *testing.T) {
	testUnmarshalXML(t, []unmarshalTest{
		unmarshalTest{
			"with-shell",
			`<JobCommand><exec>command</exec></JobCommand>`,
			&JobCommand{},
			func (rv interface {}) error {
				v := rv.(*JobCommand)
				if v.ShellCommand != "command" {
					return fmt.Errorf("got ShellCommand %s, but expecting command", v.ShellCommand)
				}
				return nil
			},
		},
		unmarshalTest{
			"with-script",
			`<JobCommand><script>script</script></JobCommand>`,
			&JobCommand{},
			func (rv interface {}) error {
				v := rv.(*JobCommand)
				if v.Script != "script" {
					return fmt.Errorf("got Script %s, but expecting script", v.Script)
				}
				return nil
			},
		},
		unmarshalTest{
			"with-script-interpreter",
			`<JobCommand><script>Hello World!</script><fileExtension>sh</fileExtension><scriptinterpreter>sudo</scriptinterpreter></JobCommand>`,
			&JobCommand{},
			func (rv interface {}) error {
				v := rv.(*JobCommand)
				if v.FileExtension != "sh" {
					return fmt.Errorf("got FileExtension %s, but expecting sh", v.FileExtension)
				}
				if v.Script != "Hello World!" {
					return fmt.Errorf("got Script %s, but expecting Hello World!", v.Script)
				}
				if v.ScriptInterpreter == nil {
					return fmt.Errorf("got %s, but expecting not nil", v.ScriptInterpreter)
				}
				if v.ScriptInterpreter.InvocationString != "sudo" {
					return fmt.Errorf("got InvocationString %s, but expecting sudo", v.ScriptInterpreter.InvocationString)
				}
				return nil
			},
		},
	})
}

func TestMarshalScriptInterpreter(t *testing.T) {
	testMarshalXML(t, []marshalTest{
		marshalTest{
			"with-script-interpreter",
			JobCommandScriptInterpreter{
					InvocationString: "sudo",
			},
			`<scriptinterpreter>sudo</scriptinterpreter>`,
		},
		marshalTest{
			"with-script-interpreter-quoted",
			JobCommandScriptInterpreter{
					ArgsQuoted: true,
					InvocationString: "sudo",
			},
			`<scriptinterpreter argsquoted="true">sudo</scriptinterpreter>`,
		},
	})
}

func TestUnmarshalScriptInterpreter(t *testing.T) {
	testUnmarshalXML(t, []unmarshalTest{
		unmarshalTest{
			"with-script-interpreter",
			`<scriptinterpreter>sudo</scriptinterpreter>`,
			&JobCommandScriptInterpreter{},
			func (rv interface {}) error {
				v := rv.(*JobCommandScriptInterpreter)
				if v.InvocationString != "sudo" {
					return fmt.Errorf("got InvocationString %s, but expecting sudo", v.InvocationString)
				}
				if v.ArgsQuoted {
					return fmt.Errorf("got ArgsQuoted %s, but expecting false", v.ArgsQuoted)
				}
				return nil
			},
		},
		unmarshalTest{
			"with-script-interpreter-quoted",
			`<scriptinterpreter argsquoted="true">sudo</scriptinterpreter>`,
			&JobCommandScriptInterpreter{},
			func (rv interface {}) error {
				v := rv.(*JobCommandScriptInterpreter)
				if v.InvocationString != "sudo" {
					return fmt.Errorf("got InvocationString %s, but expecting sudo", v.InvocationString)
				}
				if ! v.ArgsQuoted {
					return fmt.Errorf("got ArgsQuoted %s, but expecting true", v.ArgsQuoted)
				}
				return nil
			},
		},
	})
}

func TestMarshalErrorHanlder(t *testing.T) {
	testMarshalXML(t, []marshalTest{
		marshalTest{
			"with-errorhandler",
			JobCommandSequence{
				ContinueOnError: true,
				OrderingStrategy: "step-first",
				Commands: []JobCommand{
					JobCommand{
						Script: "inline_script",
						ErrorHandler: &JobCommand{
							ContinueOnError: true,
							Script: "error_script",
						},
					},
				},
			},
			`<sequence keepgoing="true" strategy="step-first"><command><errorhandler keepgoingOnSuccess="true"><script>error_script</script></errorhandler><script>inline_script</script></command></sequence>`,
		},
	})
}


func TestMarshalJobOption(t *testing.T) {
	testMarshalXML(t, []marshalTest{
		marshalTest{
			"with-option-basic",
			JobOption{
				Name: "basic",
			},
			`<option name="basic"></option>`,
		},
		marshalTest{
			"with-option-multivalued",
			JobOption{
				Name: "Multivalued",
				MultiValueDelimiter: "|",
				RequirePredefinedChoice: true,
				AllowsMultipleValues: true,
				IsRequired: true,
				ValueChoices: JobValueChoices([]string{"myValues"}),
			},
			`<option delimiter="|" enforcedvalues="true" multivalued="true" name="Multivalued" required="true" values="myValues"></option>`,
		},
		marshalTest{
			"with-all-attributes",
			JobOption{
				Name: "advanced",
				MultiValueDelimiter: "|",
				RequirePredefinedChoice: true,
				AllowsMultipleValues: true,
				ValidationRegex: ".+",
				IsRequired: true,
				ObscureInput: true,
				StoragePath: "myKey",
				DefaultValue: "myValue",
				ValueIsExposedToScripts: true,
				ValueChoices: JobValueChoices([]string{"myValues"}),
				ValueChoicesURL: "myValuesUrl",
			},
			`<option delimiter="|" enforcedvalues="true" multivalued="true" name="advanced" regex=".+" required="true" secure="true" storagePath="myKey" value="myValue" valueExposed="true" values="myValues" valuesUrl="myValuesUrl"></option>`,
		},
	})
}

