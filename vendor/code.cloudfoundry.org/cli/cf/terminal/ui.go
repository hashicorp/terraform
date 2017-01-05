package terminal

import (
	"fmt"
	"io"
	"strings"

	. "code.cloudfoundry.org/cli/cf/i18n"

	"bytes"

	"bufio"

	"code.cloudfoundry.org/cli/cf"
	"code.cloudfoundry.org/cli/cf/configuration/coreconfig"
	"code.cloudfoundry.org/cli/cf/trace"
)

type ColoringFunction func(value string, row int, col int) string

func NotLoggedInText() string {
	return fmt.Sprintf(T("Not logged in. Use '{{.CFLoginCommand}}' to log in.", map[string]interface{}{"CFLoginCommand": CommandColor(cf.Name + " " + "login")}))
}

//go:generate counterfeiter . UI
type UI interface {
	PrintPaginator(rows []string, err error)
	Say(message string, args ...interface{})

	// ProgressReader
	PrintCapturingNoOutput(message string, args ...interface{})
	Warn(message string, args ...interface{})
	Ask(prompt string) (answer string)
	AskForPassword(prompt string) (answer string)
	Confirm(message string) bool
	ConfirmDelete(modelType, modelName string) bool
	ConfirmDeleteWithAssociations(modelType, modelName string) bool
	Ok()
	Failed(message string, args ...interface{})
	ShowConfiguration(coreconfig.Reader) error
	LoadingIndication()
	Table(headers []string) *UITable
	NotifyUpdateIfNeeded(coreconfig.Reader)

	Writer() io.Writer
}

type Printer interface {
	Print(a ...interface{}) (n int, err error)
	Printf(format string, a ...interface{}) (n int, err error)
	Println(a ...interface{}) (n int, err error)
}

type terminalUI struct {
	stdin   io.Reader
	stdout  io.Writer
	printer Printer
	logger  trace.Printer
}

func NewUI(r io.Reader, w io.Writer, printer Printer, logger trace.Printer) UI {
	return &terminalUI{
		stdin:   r,
		stdout:  w,
		printer: printer,
		logger:  logger,
	}
}

func (ui terminalUI) Writer() io.Writer {
	return ui.stdout
}

func (ui *terminalUI) PrintPaginator(rows []string, err error) {
	if err != nil {
		ui.Failed(err.Error())
		return
	}

	for _, row := range rows {
		ui.Say(row)
	}
}

func (ui *terminalUI) PrintCapturingNoOutput(message string, args ...interface{}) {
	if len(args) == 0 {
		fmt.Fprintf(ui.stdout, "%s", message)
	} else {
		fmt.Fprintf(ui.stdout, message, args...)
	}
}

func (ui *terminalUI) Say(message string, args ...interface{}) {
	if len(args) == 0 {
		_, _ = ui.printer.Printf("%s\n", message)
	} else {
		_, _ = ui.printer.Printf(message+"\n", args...)
	}
}

func (ui *terminalUI) Warn(message string, args ...interface{}) {
	message = fmt.Sprintf(message, args...)
	ui.Say(WarningColor(message))
	return
}

func (ui *terminalUI) Ask(prompt string) string {
	fmt.Fprintf(ui.stdout, "\n%s%s ", prompt, PromptColor(">"))

	rd := bufio.NewReader(ui.stdin)
	line, err := rd.ReadString('\n')
	if err == nil {
		return strings.TrimSpace(line)
	}
	return ""
}

func (ui *terminalUI) ConfirmDeleteWithAssociations(modelType, modelName string) bool {
	return ui.confirmDelete(T("Really delete the {{.ModelType}} {{.ModelName}} and everything associated with it?",
		map[string]interface{}{
			"ModelType": modelType,
			"ModelName": EntityNameColor(modelName),
		}))
}

func (ui *terminalUI) ConfirmDelete(modelType, modelName string) bool {
	return ui.confirmDelete(T("Really delete the {{.ModelType}} {{.ModelName}}?",
		map[string]interface{}{
			"ModelType": modelType,
			"ModelName": EntityNameColor(modelName),
		}))
}

func (ui *terminalUI) confirmDelete(message string) bool {
	result := ui.Confirm(message)

	if !result {
		ui.Warn(T("Delete cancelled"))
	}

	return result
}

func (ui *terminalUI) Confirm(message string) bool {
	response := ui.Ask(message)
	switch strings.ToLower(response) {
	case "y", "yes", T("yes"):
		return true
	}
	return false
}

func (ui *terminalUI) Ok() {
	ui.Say(SuccessColor(T("OK")))
}

func (ui *terminalUI) Failed(message string, args ...interface{}) {
	message = fmt.Sprintf(message, args...)

	failed := "FAILED"
	if T != nil {
		failed = T("FAILED")
	}

	ui.logger.Print(failed)
	ui.logger.Print(message)

	if !ui.logger.WritesToConsole() {
		ui.Say(FailureColor(failed))
		ui.Say(message)
	}
}

func (ui *terminalUI) ShowConfiguration(config coreconfig.Reader) error {
	var err error
	table := ui.Table([]string{"", ""})

	if config.HasAPIEndpoint() {
		table.Add(
			T("API endpoint:"),
			T("{{.APIEndpoint}} (API version: {{.APIVersionString}})",
				map[string]interface{}{
					"APIEndpoint":      EntityNameColor(config.APIEndpoint()),
					"APIVersionString": EntityNameColor(config.APIVersion()),
				}),
		)
	}

	if !config.IsLoggedIn() {
		err = table.Print()
		if err != nil {
			return err
		}
		ui.Say(NotLoggedInText())
		return nil
	}

	table.Add(T("User:"), EntityNameColor(config.UserEmail()))

	if !config.HasOrganization() && !config.HasSpace() {
		err = table.Print()
		if err != nil {
			return err
		}
		command := fmt.Sprintf("%s target -o ORG -s SPACE", cf.Name)
		ui.Say(T("No org or space targeted, use '{{.CFTargetCommand}}'",
			map[string]interface{}{
				"CFTargetCommand": CommandColor(command),
			}))
		return nil
	}

	if config.HasOrganization() {
		table.Add(
			T("Org:"),
			EntityNameColor(config.OrganizationFields().Name),
		)
	} else {
		command := fmt.Sprintf("%s target -o Org", cf.Name)
		table.Add(
			T("Org:"),
			T("No org targeted, use '{{.CFTargetCommand}}'",
				map[string]interface{}{
					"CFTargetCommand": CommandColor(command),
				}),
		)
	}

	if config.HasSpace() {
		table.Add(
			T("Space:"),
			EntityNameColor(config.SpaceFields().Name),
		)
	} else {
		command := fmt.Sprintf("%s target -s SPACE", cf.Name)
		table.Add(
			T("Space:"),
			T("No space targeted, use '{{.CFTargetCommand}}'", map[string]interface{}{"CFTargetCommand": CommandColor(command)}),
		)
	}

	err = table.Print()
	if err != nil {
		return err
	}
	return nil
}

func (ui *terminalUI) LoadingIndication() {
	_, _ = ui.printer.Print(".")
}

func (ui *terminalUI) Table(headers []string) *UITable {
	return &UITable{
		UI:    ui,
		Table: NewTable(headers),
	}
}

type UITable struct {
	UI    UI
	Table *Table
}

func (u *UITable) Add(row ...string) {
	u.Table.Add(row...)
}

// Print formats the table and then prints it to the UI specified at
// the time of the construction. Afterwards the table is cleared,
// becoming ready for another round of rows and printing.
func (u *UITable) Print() error {
	result := &bytes.Buffer{}
	t := u.Table

	err := t.PrintTo(result)
	if err != nil {
		return err
	}

	// DevNote. With the change to printing into a buffer all
	// lines now come with a terminating \n. The t.ui.Say() below
	// will then add another \n to that. To avoid this additional
	// line we chop off the last \n from the output (if there is
	// any). Operating on the slice avoids string copying.
	//
	// WIBNI if the terminal API had a variant of Say not assuming
	// that each output is a single line.

	r := result.Bytes()
	if len(r) > 0 {
		r = r[0 : len(r)-1]
	}

	// Only generate output for a non-empty table.
	if len(r) > 0 {
		u.UI.Say("%s", string(r))
	}
	return nil
}

func (ui *terminalUI) NotifyUpdateIfNeeded(config coreconfig.Reader) {
	if !config.IsMinCLIVersion(cf.Version) {
		ui.Say("")
		ui.Say(T("Cloud Foundry API version {{.APIVer}} requires CLI version {{.CLIMin}}.  You are currently on version {{.CLIVer}}. To upgrade your CLI, please visit: https://github.com/cloudfoundry/cli#downloads",
			map[string]interface{}{
				"APIVer": config.APIVersion(),
				"CLIMin": config.MinCLIVersion(),
				"CLIVer": cf.Version,
			}))
	}
}
