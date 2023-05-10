// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package experiments

// Experiment represents a particular experiment, which can be activated
// independently of all other experiments.
type Experiment string

// All active and defunct experiments must be represented by constants whose
// internal string values are unique.
//
// Each of these declared constants must also be registered as either a
// current or a defunct experiment in the init() function below.
//
// Each experiment is represented by a string that must be a valid HCL
// identifier so that it can be specified in configuration.
const (
	VariableValidation             = Experiment("variable_validation")
	ModuleVariableOptionalAttrs    = Experiment("module_variable_optional_attrs")
	SuppressProviderSensitiveAttrs = Experiment("provider_sensitive_attrs")
	ConfigDrivenMove               = Experiment("config_driven_move")
	PreconditionsPostconditions    = Experiment("preconditions_postconditions")
)

func init() {
	// Each experiment constant defined above must be registered here as either
	// a current or a concluded experiment.
	registerConcludedExperiment(VariableValidation, "Custom variable validation can now be used by default, without enabling an experiment.")
	registerConcludedExperiment(SuppressProviderSensitiveAttrs, "Provider-defined sensitive attributes are now redacted by default, without enabling an experiment.")
	registerConcludedExperiment(ConfigDrivenMove, "Declarations of moved resource instances using \"moved\" blocks can now be used by default, without enabling an experiment.")
	registerConcludedExperiment(PreconditionsPostconditions, "Condition blocks can now be used by default, without enabling an experiment.")
	registerConcludedExperiment(ModuleVariableOptionalAttrs, "The final feature corresponding to this experiment differs from the experimental form and is available in the Terraform language from Terraform v1.3.0 onwards.")
}

// GetCurrent takes an experiment name and returns the experiment value
// representing that expression if and only if it is a current experiment.
//
// If the selected experiment is concluded, GetCurrent will return an
// error of type ConcludedError whose message hopefully includes some guidance
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

	if msg, concluded := concludedExperiments[exp]; concluded {
		return Experiment(""), ConcludedError{ExperimentName: name, Message: msg}
	}

	return Experiment(""), UnavailableError{ExperimentName: name}
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

// IsConcluded returns true if the receiver is a concluded experiment.
func (e Experiment) IsConcluded() bool {
	_, exists := concludedExperiments[e]
	return exists
}

// currentExperiments are those which are available to activate in the current
// version of Terraform.
//
// Members of this set are registered in the init function above.
var currentExperiments = make(Set)

// concludedExperiments are those which were available to activate in an earlier
// version of Terraform but are no longer available, either because the feature
// in question has been implemented or because the experiment failed and the
// feature was abandoned. Each experiment maps to a message describing the
// outcome, so we can give users feedback about what they might do in modules
// using concluded experiments.
//
// After an experiment has been concluded for a whole major release span it can
// be removed, since we expect users to perform upgrades one major release at
// at time without skipping and thus they will see the concludedness error
// message as they upgrade through a prior major version.
//
// Members of this map are registered in the init function above.
var concludedExperiments = make(map[Experiment]string)

//lint:ignore U1000 No experiments are active
func registerCurrentExperiment(exp Experiment) {
	currentExperiments.Add(exp)
}

func registerConcludedExperiment(exp Experiment, message string) {
	concludedExperiments[exp] = message
}
