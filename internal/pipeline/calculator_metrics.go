package pipeline

import (
	"github.com/sanspareilsmyn/featurelens/internal/config"
	"github.com/sanspareilsmyn/featurelens/internal/message"
	"go.uber.org/zap"
	"math"
	"time"
)

// processNonNullValue attempts to process a non-null value based on the feature's metric type.
// Returns true if processing was successful according to the type, false otherwise.
func (c *Calculator) processNonNullValue(stats *FeatureStats, msg message.DynamicMessage, featureCfg config.FeatureConfig) bool {
	switch featureCfg.MetricType {
	case "numerical":
		return c.processNumericalValue(stats, msg, featureCfg.Name)

	// TODO: add categorical!
	// case "categorical": // Future extension point
	//     return c.processCategoricalValue(stats, msg, featureCfg.Name)
	default:
		c.logger.Debug("Skipping feature update due to unsupported metric type",
			zap.String("feature_name", featureCfg.Name),
			zap.String("metric_type", featureCfg.MetricType),
		)
		return false
	}
}

// processNumericalValue attempts to parse a float64 value and update numerical stats.
// Returns true on success, false on failure (e.g., parsing error).
func (c *Calculator) processNumericalValue(stats *FeatureStats, msg message.DynamicMessage, featureName string) bool {
	floatValPtr, ok := msg.GetFloat64(featureName)
	if !ok {
		// GetFloat64 failed to parse the value as a number (value exists, is not null)
		return false
	}
	floatVal := *floatValPtr
	stats.sum += floatVal
	stats.sumSq += floatVal * floatVal
	return true
}

// calculateMeanVariance computes mean and variance from FeatureStats.
// Added featureName and windowStart for better context in logs.
func (c *Calculator) calculateMeanVariance(stats *FeatureStats, featureName string, windowStart time.Time) (mean, variance float64) {
	validCount := stats.count - stats.nullCount
	if validCount <= 0 {
		return math.NaN(), math.NaN()
	}

	mean = stats.sum / float64(validCount)

	// Variance = E[X^2] - (E[X])^2 = (SumSq / N) - Mean^2
	meanSq := mean * mean
	sumSqAvg := stats.sumSq / float64(validCount)
	variance = sumSqAvg - meanSq

	// Correct for potential floating point inaccuracies yielding small negative variance
	if variance < 0 && variance > -1e-9 { // Allow for tiny floating point errors
		variance = 0
	} else if variance < 0 {
		c.logger.Warn("Negative variance calculated, setting to 0",
			zap.String("feature_name", featureName),
			zap.Time("window_start", windowStart),
			zap.Float64("calculated_variance", variance),
			zap.Int64("valid_count", validCount),
			zap.Float64("sum_sq", stats.sumSq),
			zap.Float64("mean", mean),
		)
		variance = 0
	}
	return mean, variance
}
