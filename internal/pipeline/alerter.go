package pipeline

import (
	"context"
	"math"
	"time"

	"go.uber.org/zap"

	"github.com/sanspareilsmyn/featurelens/internal/config"
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
			a.checkThresholds(ctx, result)

		case <-ctx.Done():
			sugar.Info("Context cancelled, stopping alerter.")
			return ctx.Err()
		}
	}
}

// checkThresholds compares the result against configured thresholds and logs alerts.
func (a *Alerter) checkThresholds(ctx context.Context, result AggregationResult) {
	sugar := a.logger.Sugar()

	featureCfg, exists := a.features[result.FeatureName]
	if !exists {
		sugar.Warnw("Received result for unconfigured feature",
			zap.String("feature_name", result.FeatureName),
			zap.Time("window_start", result.WindowStart),
			zap.Time("window_end", result.WindowEnd),
		)
		return
	}

	thresholds := featureCfg.Thresholds
	featureName := result.FeatureName

	// Pre-calculate metrics to avoid redundant checks/calculations
	nullRate := math.NaN()
	if result.Count > 0 {
		nullRate = float64(result.NullCount) / float64(result.Count)
	}

	stdDev := math.NaN()
	if !math.IsNaN(result.Variance) && result.Variance >= 0 {
		stdDev = math.Sqrt(result.Variance)
	}

	// --- Perform Threshold Checks ---
	a.checkNullRate(sugar, featureName, result.WindowEnd, nullRate, thresholds.NullRate)
	a.checkMean(sugar, featureName, result.WindowEnd, result.Mean, thresholds.MeanMin, thresholds.MeanMax)
	a.checkStdDev(sugar, featureName, result.WindowEnd, stdDev, thresholds.StdDevMin, thresholds.StdDevMax)

	// --- Log Statistics ---
	a.logStats(sugar, result, nullRate, stdDev)
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
	}
}

// Helper function to check Mean thresholds
func (a *Alerter) checkMean(sugar *zap.SugaredLogger, featureName string, windowEnd time.Time, actualMean float64, minThreshold, maxThreshold *float64) {
	if math.IsNaN(actualMean) {
		return // Mean not calculable
	}

	if minThreshold != nil && actualMean < *minThreshold {
		sugar.Warnw("Mean violation (Min)",
			zap.String("feature_name", featureName),
			zap.Time("window_end", windowEnd),
			zap.Float64("actual", actualMean),
			zap.Float64("threshold", *minThreshold),
			zap.String("comparison", "<"),
		)
	}
	if maxThreshold != nil && actualMean > *maxThreshold {
		sugar.Warnw("Mean violation (Max)",
			zap.String("feature_name", featureName),
			zap.Time("window_end", windowEnd),
			zap.Float64("actual", actualMean),
			zap.Float64("threshold", *maxThreshold),
			zap.String("comparison", ">"),
		)
	}
}

// Helper function to check Standard Deviation thresholds
func (a *Alerter) checkStdDev(sugar *zap.SugaredLogger, featureName string, windowEnd time.Time, actualStdDev float64, minThreshold, maxThreshold *float64) {
	if math.IsNaN(actualStdDev) {
		return // StdDev not calculable
	}

	if minThreshold != nil && actualStdDev < *minThreshold {
		sugar.Warnw("StdDev violation (Min)",
			zap.String("feature_name", featureName),
			zap.Time("window_end", windowEnd),
			zap.Float64("actual", actualStdDev),
			zap.Float64("threshold", *minThreshold),
			zap.String("comparison", "<"),
		)
	}
	if maxThreshold != nil && actualStdDev > *maxThreshold {
		sugar.Warnw("StdDev violation (Max)",
			zap.String("feature_name", featureName),
			zap.Time("window_end", windowEnd),
			zap.Float64("actual", actualStdDev),
			zap.Float64("threshold", *maxThreshold),
			zap.String("comparison", ">"),
		)
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
