package config

// ProvisionerWhen is an enum for valid values for when to run provisioners.
type ProvisionerWhen int

const (
	ProvisionerWhenInvalid ProvisionerWhen = iota
	ProvisionerWhenCreate
	ProvisionerWhenDestroy
)

var provisionerWhenStrs = map[ProvisionerWhen]string{
	ProvisionerWhenInvalid: "invalid",
	ProvisionerWhenCreate:  "create",
	ProvisionerWhenDestroy: "destroy",
}

func (v ProvisionerWhen) String() string {
	return provisionerWhenStrs[v]
}

// ProvisionerOnFailure is an enum for valid values for on_failure options
// for provisioners.
type ProvisionerOnFailure int

const (
	ProvisionerOnFailureInvalid ProvisionerOnFailure = iota
	ProvisionerOnFailureContinue
	ProvisionerOnFailureFail
)

var provisionerOnFailureStrs = map[ProvisionerOnFailure]string{
	ProvisionerOnFailureInvalid:  "invalid",
	ProvisionerOnFailureContinue: "continue",
	ProvisionerOnFailureFail:     "fail",
}

func (v ProvisionerOnFailure) String() string {
	return provisionerOnFailureStrs[v]
}
