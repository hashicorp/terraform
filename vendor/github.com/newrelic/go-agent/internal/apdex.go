package internal

import "time"

// ApdexZone is a transaction classification.
type ApdexZone int

// https://en.wikipedia.org/wiki/Apdex
const (
	ApdexNone ApdexZone = iota
	ApdexSatisfying
	ApdexTolerating
	ApdexFailing
)

// ApdexFailingThreshold calculates the threshold at which the transaction is
// considered a failure.
func ApdexFailingThreshold(threshold time.Duration) time.Duration {
	return 4 * threshold
}

// CalculateApdexZone calculates the apdex based on the transaction duration and
// threshold.
//
// Note that this does not take into account whether or not the transaction
// had an error.  That is expected to be done by the caller.
func CalculateApdexZone(threshold, duration time.Duration) ApdexZone {
	if duration <= threshold {
		return ApdexSatisfying
	}
	if duration <= ApdexFailingThreshold(threshold) {
		return ApdexTolerating
	}
	return ApdexFailing
}

func (zone ApdexZone) label() string {
	switch zone {
	case ApdexSatisfying:
		return "S"
	case ApdexTolerating:
		return "T"
	case ApdexFailing:
		return "F"
	default:
		return ""
	}
}
