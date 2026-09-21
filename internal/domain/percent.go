package domain

import "math"

// ValidatePercent reports whether v is an acceptable provider-reported
// percentage: finite and within [0,100]. Values outside the range are a
// protocol/semantic mismatch and must not be clamped or substituted.
func ValidatePercent(v float64) bool {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return false
	}
	return v >= 0 && v <= 100
}

// NormalizePercentSignedZero maps -0 to 0 so formatting never emits "-0.0%".
func NormalizePercentSignedZero(v float64) float64 {
	if v == 0 {
		return 0
	}
	return v
}
