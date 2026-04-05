package governor

import (
	"math"
	"time"
)

// LambdaFromHalfLifeDays returns decay constant λ (per hour) for exponential decay
// with the given half-life in days: score(t) = score(0) * exp(-λt) with t in hours.
func LambdaFromHalfLifeDays(halfLifeDays float64) float64 {
	if halfLifeDays <= 0 {
		return 0
	}
	return math.Log(2) / (halfLifeDays * 24)
}

// applyTemporalDecay applies exponential decay in hours: score * exp(-lambdaPerHour * hours).
// If lambdaPerHour <= 0, the score is unchanged.
func applyTemporalDecay(score float64, createdAt time.Time, lambdaPerHour float64) float64 {
	if lambdaPerHour <= 0 {
		return score
	}
	hours := time.Since(createdAt).Hours()
	if hours < 0 {
		hours = 0
	}
	return score * math.Exp(-lambdaPerHour*hours)
}
