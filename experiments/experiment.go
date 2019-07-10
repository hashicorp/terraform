package experiments

// Experiment represents a particular experiment, which can be activated
// independently of all other experiments.
type Experiment string

// All active and defunct experiments must be represented by constants whose
// internal string values are unique.
//
// Each of these declarated constants must also be registered as either a
// current or a defunct experiment in the init() function below.
//
// Each experiment is represented by a string that must be a valid HCL
// identifier so that it can be specified in configuration.
const (
	ResourceForEach = Experiment("resource_for_each")
)

func init() {
	// Each experiment constant defined above must be registered here as either
	// a current or a defunct experiment.
	registerCurrentExperiment(ResourceForEach)
}

// GetCurrent takes an experiment name and returns the experiment value
// representing that expression if and only if it is a current experiment.
//
// If the selected experiment is considered defunct, GetCurrent will return an
// error of type DefunctError whose message hopefully includes some guidance
// for users of the experiment on how to migrate to a stable feature that
// succeeded it.
//
// If the selected experiment is not known at all, GetCurrent will return an
// error of type UnavailableError.
func GetCurrent(name string) (Experiment, error) {
	exp := Experiment(name)
	if currentExperiments.Has(exp) {
		return exp, nil
	}

	if msg, defunct := defunctExperiments[exp]; defunct {
		return Experiment(""), DefunctError{msg: msg}
	}

	return Experiment(""), UnavailableError{name: name}
}

// Keyword returns the keyword that would be used to activate this experiment
// in the configuration.
func (e Experiment) Keyword() string {
	return string(e)
}

// IsCurrent returns true if the receiver is considered a currently-selectable
// experiment.
func (e Experiment) IsCurrent() bool {
	return currentExperiments.Has(e)
}

// IsDefunct returns true if the receiver is considered to be a defunct
// experiment.
func (e Experiment) IsDefunct() bool {
	_, exists := defunctExperiments[e]
	return exists
}

// currentExperiments are those which are available to activate in the current
// version of Terraform.
//
// Members of this set are registered in the init function above.
var currentExperiments = make(Set)

// defunctExperiments are those which were available to activate in an earlier
// version of Terraform but are no longer available, either because the feature
// in question has been implemented or because the experiment failed and the
// feature was abandoned. Each experiment maps to a message describing the
// outcome, so we can give users feedback about what they might do in modules
// using defunct experiments.
//
// After an experiment has been defunct for a whole major release span it can
// be removed, since we expect users to perform upgrades one major release at
// at time without skipping and thus they will see the defunct-ness error
// message as they upgrade through a prior major version.
//
// Members of this map are registered in the init function above.
var defunctExperiments = make(map[Experiment]string)

func registerCurrentExperiment(exp Experiment) {
	currentExperiments.Add(exp)
}

func registerDefunctExperiment(exp Experiment, message string) {
	defunctExperiments[exp] = message
}
