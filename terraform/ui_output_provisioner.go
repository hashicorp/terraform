package terraform

// ProvisionerUIOutput is an implementation of UIOutput that calls a hook
// for the output so that the hooks can handle it.
type ProvisionerUIOutput struct {
	Info  *InstanceInfo
	Type  string
	Hooks []Hook
}

func (o *ProvisionerUIOutput) Output(msg string) {
	for _, h := range o.Hooks {
		h.ProvisionOutput(o.Info, o.Type, msg)
	}
}
