package pipeline

import (
	"context"
	"math"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.uber.org/zap"

	"github.com/sanspareilsmyn/featurelens/internal/config"
)

// Prometheus Metrics Definition
var (
	featureCount = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "featurelens_feature_window_count_total", // Follow Prometheus naming conventions
			Help: "Total number of messages processed for a feature in the last window.",
		},
		[]string{"feature_name"}, // Label: feature_name
	)
	featureNullCount = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "featurelens_feature_window_null_count_total",
			Help: "Total number of null values encountered for a feature in the last window.",
		},
		[]string{"feature_name"},
	)
	featureNullRate = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "featurelens_feature_window_null_rate",
			Help: "Null rate for a feature in the last window (NullCount / Count).",
		},
		[]string{"feature_name"},
	)
	featureMean = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "featurelens_feature_window_mean_value",
			Help: "Mean value for a feature in the last window.",
		},
		[]string{"feature_name"},
	)
	featureStdDev = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "featurelens_feature_window_stddev_value",
			Help: "Standard deviation for a feature in the last window.",
		},
		[]string{"feature_name"},
	)
	// Optional: Track violations
	featureThresholdViolations = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "featurelens_feature_threshold_violations_total",
			Help: "Total number of threshold violations detected for a feature and specific check.",
		},
		[]string{"feature_name", "check_type", "comparison"}, // Labels: feature_name, check_type (e.g., mean, null_rate), comparison (<, >)
	)
)

// Alerter receives aggregation results and checks them against configured thresholds.
type Alerter struct {
	features map[string]config.FeatureConfig
	input    <-chan AggregationResult
	logger   *zap.Logger
}

// NewAlerter creates a new Alerter instance.
func NewAlerter(features []config.FeatureConfig, input <-chan AggregationResult, logger *zap.Logger) *Alerter {
	featureMap := make(map[string]config.FeatureConfig)
	for _, f := range features {
		featureMap[f.Name] = f
	}

	logger.Debug("Alerter initialized", zap.Int("feature_count", len(featureMap)))

	return &Alerter{
		features: featureMap,
		input:    input,
		logger:   logger,
	}
}

// Run starts the alerter's processing loop, checking results against thresholds.
func (a *Alerter) Run(ctx context.Context) error {
	sugar := a.logger.Sugar()
	sugar.Info("Starting alerter loop...")
	defer sugar.Info("Alerter loop stopped.")

	for {
		select {
		case result, ok := <-a.input:
			if !ok {
				sugar.Info("Alerter input channel closed.")
				return nil
			}
			a.processResult(ctx, result)

		case <-ctx.Done():
			sugar.Info("Context cancelled, stopping alerter.")
			return ctx.Err()
		}
	}
}

// processResult checks thresholds, logs alerts, and updates Prometheus metrics.
func (a *Alerter) processResult(ctx context.Context, result AggregationResult) {
	sugar := a.logger.Sugar()
	featureName := result.FeatureName

	featureCfg, exists := a.features[featureName]
	if !exists {
		sugar.Warnw("Received result for unconfigured feature, skipping metric update",
			zap.String("feature_name", featureName),
			zap.Time("window_start", result.WindowStart),
			zap.Time("window_end", result.WindowEnd),
		)
		return
	}

	// Calculate Metrics
	nullRateVal := math.NaN()
	if result.Count > 0 {
		nullRateVal = float64(result.NullCount) / float64(result.Count)
	}

	stdDevVal := math.NaN()
	if !math.IsNaN(result.Variance) && result.Variance >= 0 {
		stdDevVal = math.Sqrt(result.Variance)
	}

	// Update Prometheus Gauges
	// Use .WithLabelValues(featureName) to get the specific gauge for this feature
	featureCount.WithLabelValues(featureName).Set(float64(result.Count))
	featureNullCount.WithLabelValues(featureName).Set(float64(result.NullCount))
	if !math.IsNaN(nullRateVal) {
		featureNullRate.WithLabelValues(featureName).Set(nullRateVal)
	} else {
		featureNullRate.WithLabelValues(featureName).Set(0)
	}
	if !math.IsNaN(result.Mean) {
		featureMean.WithLabelValues(featureName).Set(result.Mean)
	} else {
		featureMean.WithLabelValues(featureName).Set(0)
	}
	if !math.IsNaN(stdDevVal) {
		featureStdDev.WithLabelValues(featureName).Set(stdDevVal)
	} else {
		featureStdDev.WithLabelValues(featureName).Set(0)
	}

	// Perform Threshold Checks & Log
	thresholds := featureCfg.Thresholds
	a.checkNullRate(sugar, featureName, result.WindowEnd, nullRateVal, thresholds.NullRate)
	a.checkMean(sugar, featureName, result.WindowEnd, result.Mean, thresholds.MeanMin, thresholds.MeanMax)
	a.checkStdDev(sugar, featureName, result.WindowEnd, stdDevVal, thresholds.StdDevMin, thresholds.StdDevMax)

	// Log Statistics
	a.logStats(sugar, result, nullRateVal, stdDevVal)
}

// Helper function to check Null Rate threshold
func (a *Alerter) checkNullRate(sugar *zap.SugaredLogger, featureName string, windowEnd time.Time, actualRate float64, threshold *float64) {
	if threshold == nil || math.IsNaN(actualRate) {
		return
	}
	if actualRate > *threshold {
		sugar.Warnw("Null Rate violation",
			zap.String("feature_name", featureName),
			zap.Time("window_end", windowEnd),
			zap.Float64("actual", actualRate),
			zap.Float64("threshold", *threshold),
			zap.String("comparison", ">"),
		)
		// Increment violation counter
		featureThresholdViolations.WithLabelValues(featureName, "null_rate", ">").Inc()
	}
}

// Helper function to check Mean thresholds
func (a *Alerter) checkMean(sugar *zap.SugaredLogger, featureName string, windowEnd time.Time, actualMean float64, minThreshold, maxThreshold *float64) {
	if math.IsNaN(actualMean) {
		return
	}
	if minThreshold != nil && actualMean < *minThreshold {
		sugar.Warnw("Mean violation (Min)",
			zap.String("feature_name", featureName),
			zap.Time("window_end", windowEnd),
			zap.Float64("actual", actualMean),
			zap.Float64("threshold", *minThreshold),
			zap.String("comparison", "<"),
		)
		featureThresholdViolations.WithLabelValues(featureName, "mean", "<").Inc()
	}
	if maxThreshold != nil && actualMean > *maxThreshold {
		sugar.Warnw("Mean violation (Max)",
			zap.String("feature_name", featureName),
			zap.Time("window_end", windowEnd),
			zap.Float64("actual", actualMean),
			zap.Float64("threshold", *maxThreshold),
			zap.String("comparison", ">"),
		)
		featureThresholdViolations.WithLabelValues(featureName, "mean", ">").Inc()
	}
}

// Helper function to check Standard Deviation thresholds
func (a *Alerter) checkStdDev(sugar *zap.SugaredLogger, featureName string, windowEnd time.Time, actualStdDev float64, minThreshold, maxThreshold *float64) {
	if math.IsNaN(actualStdDev) {
		return
	}
	if minThreshold != nil && actualStdDev < *minThreshold {
		sugar.Warnw("StdDev violation (Min)",
			zap.String("feature_name", featureName),
			zap.Time("window_end", windowEnd),
			zap.Float64("actual", actualStdDev),
			zap.Float64("threshold", *minThreshold),
			zap.String("comparison", "<"),
		)
		featureThresholdViolations.WithLabelValues(featureName, "stddev", "<").Inc()
	}
	if maxThreshold != nil && actualStdDev > *maxThreshold {
		sugar.Warnw("StdDev violation (Max)",
			zap.String("feature_name", featureName),
			zap.Time("window_end", windowEnd),
			zap.Float64("actual", actualStdDev),
			zap.Float64("threshold", *maxThreshold),
			zap.String("comparison", ">"),
		)
		featureThresholdViolations.WithLabelValues(featureName, "stddev", ">").Inc()
	}
}

// Helper function to log calculated statistics
func (a *Alerter) logStats(sugar *zap.SugaredLogger, result AggregationResult, nullRate, stdDev float64) {
	fields := []interface{}{
		zap.String("feature_name", result.FeatureName),
		zap.Time("window_end", result.WindowEnd),
		zap.Int64("count", result.Count),
	}
	if !math.IsNaN(nullRate) {
		fields = append(fields, zap.Float64("null_rate", nullRate))
	}
	if !math.IsNaN(result.Mean) {
		fields = append(fields, zap.Float64("mean", result.Mean))
	}
	if !math.IsNaN(stdDev) {
		fields = append(fields, zap.Float64("stddev", stdDev))
	}

	sugar.Infow("Feature stats processed", fields...)
}
