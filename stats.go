package fixedts

import (
	"fmt"
	"math"
	"time"
)

type Stats struct {
	From          time.Time
	Steps         uint64
	TimeAccum     float64
	StepRate      float64
	StepAverage   float64
	GapAccum      float64
	GapAbsAccum   float64
	GapAbsAverage float64
	GapMax        float64
	GapMin        float64
}

func (ts *FixedTimestep) newStats() *Stats {
	return &Stats{From: ts.Now, GapMax: math.Inf(-1), GapMin: math.Inf(1)}
}

func (ts *FixedTimestep) updateStats(stats *Stats) {
	stats.Steps++
	stats.TimeAccum += ts.Delta
	stats.GapAccum += ts.Gap
	stats.GapAbsAccum += math.Abs(ts.Gap)

	stats.StepRate = float64(stats.Steps) / stats.TimeAccum
	stats.StepAverage = stats.TimeAccum / float64(stats.Steps)

	if ts.Gap > stats.GapMax {
		stats.GapMax = ts.Gap
	}
	if ts.Gap < stats.GapMin {
		stats.GapMin = ts.Gap
	}
	stats.GapAbsAverage = stats.GapAbsAccum / float64(stats.Steps)
}

func (stats *Stats) String() string {
	return fmt.Sprintf("elapsed: %s from: %s\nsteps: %d avg: %s rate: %0.3f/s\ngap_accum: %s gap_abs_accum: %s gap_abs_avg: %s gap_min: %s gap_max: %s", secondsToDuration(stats.TimeAccum), stats.From, stats.Steps, secondsToDuration(stats.StepAverage), stats.StepRate, secondsToDuration(stats.GapAccum), secondsToDuration(stats.GapAbsAccum), secondsToDuration(stats.GapAbsAverage), secondsToDuration(stats.GapMin), secondsToDuration(stats.GapMax))
}

func secondsToDuration(seconds float64) time.Duration {
	return time.Duration(seconds * float64(time.Second))
}
