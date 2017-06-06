package terraform

type CallbackUIOutput struct {
	OutputFn func(string)
}

func (o *CallbackUIOutput) Output(v string) {
	o.OutputFn(v)
}
