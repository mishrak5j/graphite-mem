package governor

import (
	"math"
	"testing"
	"time"
)

func TestApplyTemporalDecay(t *testing.T) {
	lambda := 0.01
	now := time.Now()

	score := applyTemporalDecay(1.0, now, lambda)
	if math.Abs(score-1.0) > 0.01 {
		t.Fatalf("score for just-now memory should be ~1.0, got %f", score)
	}

	oneDay := now.Add(-24 * time.Hour)
	score24 := applyTemporalDecay(1.0, oneDay, lambda)
	expected := math.Exp(-lambda * 24)
	if math.Abs(score24-expected) > 0.01 {
		t.Fatalf("expected ~%f for 24h old memory, got %f", expected, score24)
	}

	oneWeek := now.Add(-168 * time.Hour)
	scoreWeek := applyTemporalDecay(1.0, oneWeek, lambda)
	if scoreWeek >= score24 {
		t.Fatal("week-old memory should score lower than day-old")
	}
}

func TestDecayWithZeroLambda(t *testing.T) {
	old := time.Now().Add(-1000 * time.Hour)
	score := applyTemporalDecay(0.9, old, 0.0)
	if score != 0.9 {
		t.Fatalf("with lambda=0, score should be unchanged: %f", score)
	}
}

func TestFrequencySuppressor(t *testing.T) {
	fs := NewFrequencySuppressor(3, 5)
	sid := "session-1"
	mid := "memory-1"

	if fs.ShouldSuppress(sid, mid, 0) {
		t.Fatal("should not suppress on first encounter")
	}

	fs.RecordInjection(sid, mid, 1)
	fs.RecordInjection(sid, mid, 2)
	fs.RecordInjection(sid, mid, 3)

	if !fs.ShouldSuppress(sid, mid, 4) {
		t.Fatal("should suppress after 3 injections within cooldown")
	}

	if fs.ShouldSuppress(sid, mid, 10) {
		t.Fatal("should not suppress after cooldown period")
	}
}

func TestFrequencySuppressorCrossSessions(t *testing.T) {
	fs := NewFrequencySuppressor(2, 3)

	fs.RecordInjection("s1", "m1", 1)
	fs.RecordInjection("s1", "m1", 2)

	if fs.ShouldSuppress("s2", "m1", 1) {
		t.Fatal("suppression should be per-session")
	}
}

func TestLambdaFromHalfLifeDays(t *testing.T) {
	// 3-day half-life → λ = ln(2) / (3 * 24) per hour
	lambda := LambdaFromHalfLifeDays(3)
	expected := math.Log(2) / (3 * 24)
	if math.Abs(lambda-expected) > 1e-9 {
		t.Fatalf("LambdaFromHalfLifeDays(3): got %v want %v", lambda, expected)
	}
	if LambdaFromHalfLifeDays(0) != 0 {
		t.Fatal("LambdaFromHalfLifeDays(0) should be 0")
	}
}

func TestDecayPreservesOrdering(t *testing.T) {
	lambda := 0.01
	recent := time.Now().Add(-1 * time.Hour)
	old := time.Now().Add(-100 * time.Hour)

	recentScore := applyTemporalDecay(0.8, recent, lambda)
	oldScore := applyTemporalDecay(0.85, old, lambda)

	if recentScore <= oldScore {
		t.Fatalf("recent memory (score=%f) should beat old memory (score=%f) despite lower raw similarity",
			recentScore, oldScore)
	}
}
