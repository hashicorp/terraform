package complete

// PredictSet expects specific set of terms, given in the options argument.
func PredictSet(options ...string) Predictor {
	return predictSet(options)
}

type predictSet []string

func (p predictSet) Predict(a Args) []string {
	return p
}
