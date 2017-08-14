package complete

import "github.com/posener/complete/match"

// PredictSet expects specific set of terms, given in the options argument.
func PredictSet(options ...string) Predictor {
	return predictSet(options)
}

type predictSet []string

func (p predictSet) Predict(a Args) (prediction []string) {
	for _, m := range p {
		if match.Prefix(m, a.Last) {
			prediction = append(prediction, m)
		}
	}
	return
}
