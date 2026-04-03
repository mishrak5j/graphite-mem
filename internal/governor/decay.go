package governor

import (
	"math"
	"time"
)

func applyTemporalDecay(score float64, createdAt time.Time, lambda float64) float64 {
	hours := time.Since(createdAt).Hours()
	if hours < 0 {
		hours = 0
	}
	return score * math.Exp(-lambda*hours)
}
