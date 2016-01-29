package rundeck

import (
	"encoding/xml"
	"fmt"
	"strings"
)

// JobSummary is an abbreviated description of a job that includes only its basic
// descriptive information and identifiers.
type JobSummary struct {
	XMLName     xml.Name `xml:"job"`
	ID          string   `xml:"id,attr"`
	Name        string   `xml:"name"`
	GroupName   string   `xml:"group"`
	ProjectName string   `xml:"project"`
	Description string   `xml:"description,omitempty"`
}

type jobSummaryList struct {
	XMLName xml.Name     `xml:"jobs"`
	Jobs    []JobSummary `xml:"job"`
}

// JobDetail is a comprehensive description of a job, including its entire definition.
type JobDetail struct {
	XMLName                   xml.Name            `xml:"job"`
	ID                        string              `xml:"uuid,omitempty"`
	Name                      string              `xml:"name"`
	GroupName                 string              `xml:"group,omitempty"`
	ProjectName               string              `xml:"context>project,omitempty"`
	OptionsConfig             *JobOptions         `xml:"context>options,omitempty"`
	Description               string              `xml:"description,omitempty"`
	LogLevel                  string              `xml:"loglevel,omitempty"`
	AllowConcurrentExecutions bool                `xml:"multipleExecutions"`
	Dispatch                  *JobDispatch        `xml:"dispatch"`
	CommandSequence           *JobCommandSequence `xml:"sequence,omitempty"`
	NodeFilter                *JobNodeFilter      `xml:"nodefilters,omitempty"`
}

type jobDetailList struct {
	XMLName xml.Name    `xml:"joblist"`
	Jobs    []JobDetail `xml:"job"`
}

// JobOptions represents the set of options on a job, if any.
type JobOptions struct {
	PreserveOrder bool        `xml:"preserveOrder,attr,omitempty"`
	Options       []JobOption `xml:"option"`
}

// JobOption represents a single option on a job.
type JobOption struct {
	XMLName xml.Name `xml:"option"`

	// The name of the option, which can be used to interpolate its value
	// into job commands.
	Name string `xml:"name,attr,omitempty"`

	// The default value of the option.
	DefaultValue string `xml:"value,attr,omitempty"`

	// A sequence of predefined choices for this option. Mutually exclusive with ValueChoicesURL.
	ValueChoices JobValueChoices `xml:"values,attr"`

	// A URL from which the predefined choices for this option will be retrieved.
	// Mutually exclusive with ValueChoices
	ValueChoicesURL string `xml:"valuesUrl,attr,omitempty"`

	// If set, Rundeck will reject values that are not in the set of predefined choices.
	RequirePredefinedChoice bool `xml:"enforcedvalues,attr,omitempty"`

	// Regular expression to be used to validate the option value.
	ValidationRegex string `xml:"regex,attr,omitempty"`

	// Description of the value to be shown in the Rundeck UI.
	Description string `xml:"description,omitempty"`

	// If set, Rundeck requires a value to be set for this option.
	IsRequired bool `xml:"required,attr,omitempty"`

	// When either ValueChoices or ValueChoicesURL is set, controls whether more than one
	// choice may be selected as the value.
	AllowsMultipleValues bool `xml:"multivalued,attr,omitempty"`

	// If AllowsMultipleChoices is set, the string that will be used to delimit the multiple
	// chosen options.
	MultiValueDelimiter string `xml:"delimeter,attr,omitempty"`

	// If set, the input for this field will be obscured in the UI. Useful for passwords
	// and other secrets.
	ObscureInput bool `xml:"secure,attr,omitempty"`

	// If set, the value can be accessed from scripts.
	ValueIsExposedToScripts bool `xml:"valueExposed,attr,omitempty"`
}

// JobValueChoices is a specialization of []string representing a sequence of predefined values
// for a job option.
type JobValueChoices []string

// JobCommandSequence describes the sequence of operations that a job will perform.
type JobCommandSequence struct {
	XMLName xml.Name `xml:"sequence"`

	// If set, Rundeck will continue with subsequent commands after a command fails.
	ContinueOnError bool `xml:"keepgoing,attr"`

	// Chooses the strategy by which Rundeck will execute commands. Can either be "node-first" or
	// "step-first".
	OrderingStrategy string `xml:"strategy,attr,omitempty"`

	// Sequence of commands to run in the sequence.
	Commands []JobCommand `xml:"command"`
}

// JobCommand describes a particular command to run within the sequence of commands on a job.
// The members of this struct are mutually-exclusive except for the pair of ScriptFile and
// ScriptFileArgs.
type JobCommand struct {
	XMLName xml.Name

	// A literal shell command to run.
	ShellCommand string `xml:"exec,omitempty"`

	// An inline program to run. This will be written to disk and executed, so if it is
	// a shell script it should have an appropriate #! line.
	Script string `xml:"script,omitempty"`

	// A pre-existing file (on the target nodes) that will be executed.
	ScriptFile string `xml:"scriptfile,omitempty"`

	// When ScriptFile is set, the arguments to provide to the script when executing it.
	ScriptFileArgs string `xml:"scriptargs,omitempty"`

	// A reference to another job to run as this command.
	Job *JobCommandJobRef `xml:"jobref"`

	// Configuration for a step plugin to run as this command.
	StepPlugin *JobPlugin `xml:"step-plugin"`

	// Configuration for a node step plugin to run as this command.
	NodeStepPlugin *JobPlugin `xml:"node-step-plugin"`
}

// JobCommandJobRef is a reference to another job that will run as one of the commands of a job.
type JobCommandJobRef struct {
	XMLName        xml.Name                  `xml:"jobref"`
	Name           string                    `xml:"name,attr"`
	GroupName      string                    `xml:"group,attr"`
	RunForEachNode bool                      `xml:"nodeStep,attr"`
	Arguments      JobCommandJobRefArguments `xml:"arg"`
}

// JobCommandJobRefArguments is a string representing the arguments in a JobCommandJobRef.
type JobCommandJobRefArguments string

// JobPlugin is a configuration for a plugin to run within a job.
type JobPlugin struct {
	XMLName xml.Name
	Type    string          `xml:"type,attr"`
	Config  JobPluginConfig `xml:"configuration"`
}

// JobPluginConfig is a specialization of map[string]string for job plugin configuration.
type JobPluginConfig map[string]string

// JobNodeFilter describes which nodes from the project's resource list will run the configured
// commands.
type JobNodeFilter struct {
	ExcludePrecedence bool   `xml:"excludeprecedence"`
	Query             string `xml:"filter,omitempty"`
}

type jobImportResults struct {
	Succeeded jobImportResultsCategory `xml:"succeeded"`
	Failed    jobImportResultsCategory `xml:"failed"`
	Skipped   jobImportResultsCategory `xml:"skipped"`
}

type jobImportResultsCategory struct {
	Count   int               `xml:"count,attr"`
	Results []jobImportResult `xml:"job"`
}

type jobImportResult struct {
	ID          string `xml:"id,omitempty"`
	Name        string `xml:"name"`
	GroupName   string `xml:"group,omitempty"`
	ProjectName string `xml:"context>project,omitempty"`
	Error       string `xml:"error"`
}

type JobDispatch struct {
	MaxThreadCount  int    `xml:"threadcount,omitempty"`
	ContinueOnError bool   `xml:"keepgoing"`
	RankAttribute   string `xml:"rankAttribute,omitempty"`
	RankOrder       string `xml:"rankOrder,omitempty"`
}

// GetJobSummariesForProject returns summaries of the jobs belonging to the named project.
func (c *Client) GetJobSummariesForProject(projectName string) ([]JobSummary, error) {
	jobList := &jobSummaryList{}
	err := c.get([]string{"project", projectName, "jobs"}, nil, jobList)
	return jobList.Jobs, err
}

// GetJobsForProject returns the full job details of the jobs belonging to the named project.
func (c *Client) GetJobsForProject(projectName string) ([]JobDetail, error) {
	jobList := &jobDetailList{}
	err := c.get([]string{"jobs", "export"}, map[string]string{"project": projectName}, jobList)
	if err != nil {
		return nil, err
	}
	return jobList.Jobs, nil
}

// GetJob returns the full job details of the job with the given id.
func (c *Client) GetJob(id string) (*JobDetail, error) {
	jobList := &jobDetailList{}
	err := c.get([]string{"job", id}, nil, jobList)
	if err != nil {
		return nil, err
	}
	return &jobList.Jobs[0], nil
}

// CreateJob creates a new job based on the provided structure.
func (c *Client) CreateJob(job *JobDetail) (*JobSummary, error) {
	return c.importJob(job, "create")
}

// CreateOrUpdateJob takes a job detail structure which has its ID set and either updates
// an existing job with the same id or creates a new job with that id.
func (c *Client) CreateOrUpdateJob(job *JobDetail) (*JobSummary, error) {
	return c.importJob(job, "update")
}

func (c *Client) importJob(job *JobDetail, dupeOption string) (*JobSummary, error) {
	jobList := &jobDetailList{
		Jobs: []JobDetail{*job},
	}
	args := map[string]string{
		"format":     "xml",
		"dupeOption": dupeOption,
		"uuidOption": "preserve",
	}
	result := &jobImportResults{}
	err := c.postXMLBatch([]string{"jobs", "import"}, args, jobList, result)
	if err != nil {
		return nil, err
	}

	if result.Failed.Count > 0 {
		errMsg := result.Failed.Results[0].Error
		return nil, fmt.Errorf(errMsg)
	}

	if result.Succeeded.Count != 1 {
		// Should never happen, since we send nothing in the request
		// that should cause a job to be skipped.
		return nil, fmt.Errorf("job was skipped")
	}

	return result.Succeeded.Results[0].JobSummary(), nil
}

// DeleteJob deletes the job with the given id.
func (c *Client) DeleteJob(id string) error {
	return c.delete([]string{"job", id})
}

func (c JobValueChoices) MarshalXMLAttr(name xml.Name) (xml.Attr, error) {
	if len(c) > 0 {
		return xml.Attr{name, strings.Join(c, ",")}, nil
	} else {
		return xml.Attr{}, nil
	}
}

func (c *JobValueChoices) UnmarshalXMLAttr(attr xml.Attr) error {
	values := strings.Split(attr.Value, ",")
	*c = values
	return nil
}

func (a JobCommandJobRefArguments) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	start.Attr = []xml.Attr{
		xml.Attr{xml.Name{Local: "line"}, string(a)},
	}
	e.EncodeToken(start)
	e.EncodeToken(xml.EndElement{start.Name})
	return nil
}

func (a *JobCommandJobRefArguments) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	type jobRefArgs struct {
		Line string `xml:"line,attr"`
	}
	args := jobRefArgs{}
	d.DecodeElement(&args, &start)

	*a = JobCommandJobRefArguments(args.Line)

	return nil
}

func (c JobPluginConfig) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	rc := map[string]string(c)
	return marshalMapToXML(&rc, e, start, "entry", "key", "value")
}

func (c *JobPluginConfig) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	rc := (*map[string]string)(c)
	return unmarshalMapFromXML(rc, d, start, "entry", "key", "value")
}

// JobSummary produces a JobSummary instance with values populated from the import result.
// The summary object won't have its Description populated, since import results do not
// include descriptions.
func (r *jobImportResult) JobSummary() *JobSummary {
	return &JobSummary{
		ID:          r.ID,
		Name:        r.Name,
		GroupName:   r.GroupName,
		ProjectName: r.ProjectName,
	}
}
