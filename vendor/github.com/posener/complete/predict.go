package complete

// Predictor implements a predict method, in which given
// command line arguments returns a list of options it predicts.
type Predictor interface {
	Predict(Args) []string
}

// PredictOr unions two predicate functions, so that the result predicate
// returns the union of their predication
func PredictOr(predictors ...Predictor) Predictor {
	return PredictFunc(func(a Args) (prediction []string) {
		for _, p := range predictors {
			if p == nil {
				continue
			}
			prediction = append(prediction, p.Predict(a)...)
		}
		return
	})
}

// PredictFunc determines what terms can follow a command or a flag
// It is used for auto completion, given last - the last word in the already
// in the command line, what words can complete it.
type PredictFunc func(Args) []string

// Predict invokes the predict function and implements the Predictor interface
func (p PredictFunc) Predict(a Args) []string {
	if p == nil {
		return nil
	}
	return p(a)
}

// PredictNothing does not expect anything after.
var PredictNothing Predictor

// PredictAnything expects something, but nothing particular, such as a number
// or arbitrary name.
var PredictAnything = PredictFunc(func(Args) []string { return nil })
