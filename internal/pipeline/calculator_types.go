package pipeline

import "time"

// AggregationResult holds the calculated statistics for a feature in a window.
type AggregationResult struct {
	FeatureName string
	WindowStart time.Time
	WindowEnd   time.Time
	Count       int64
	NullCount   int64
	Mean        float64
	Variance    float64
}

// FeatureStats holds the running aggregates for a single feature within a window.
type FeatureStats struct {
	count     int64
	nullCount int64
	sum       float64
	sumSq     float64
}

// windowInfo holds information about a single time window and the state of all features within it.
type windowInfo struct {
	windowStart time.Time
	windowEnd   time.Time
	features    map[string]*FeatureStats // Map FeatureName to its stats within this window
}

// newWindowInfo creates a new windowInfo instance.
func newWindowInfo(start, end time.Time) *windowInfo {
	return &windowInfo{
		windowStart: start,
		windowEnd:   end,
		features:    make(map[string]*FeatureStats),
	}
}
