package pipeline

import (
	"context"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/sanspareilsmyn/featurelens/internal/config"
	"github.com/sanspareilsmyn/featurelens/internal/message"
)

// Calculator processes messages and calculates statistics based on configuration.
// It uses windowInfo to manage state.
type Calculator struct {
	config        config.PipelineConfig
	featuresToRun []config.FeatureConfig
	input         <-chan message.DynamicMessage
	output        chan<- AggregationResult
	logger        *zap.Logger

	mu           sync.Mutex
	windowStates map[time.Time]*windowInfo
}

// NewCalculator creates a new Calculator instance.
func NewCalculator(cfg config.PipelineConfig, features []config.FeatureConfig, input <-chan message.DynamicMessage, output chan<- AggregationResult, logger *zap.Logger) *Calculator {
	c := &Calculator{
		config:        cfg,
		featuresToRun: features,
		input:         input,
		output:        output,
		logger:        logger,
		windowStates:  make(map[time.Time]*windowInfo),
	}
	logger.Info("Calculator initialized",
		zap.Duration("window_size", cfg.WindowSize),
		zap.Int("configured_features", len(features)),
	)
	return c
}

// Run starts the calculator's processing loop.
func (c *Calculator) Run(ctx context.Context) error {
	sugar := c.logger.Sugar() // Use sugared logger for convenience
	sugar.Info("Starting calculator loop...")
	defer sugar.Info("Calculator loop stopped.")

	ticker := time.NewTicker(c.config.WindowSize) // Ticker to trigger window processing based on config.WindowSize
	defer ticker.Stop()

	for {
		select {
		case msg, ok := <-c.input:
			if !ok {
				sugar.Info("Calculator input channel closed. Processing final windows...")
				c.flushWindows(time.Now())
				return nil
			}
			c.processMessage(msg)

		case tickTime := <-ticker.C:
			// Time to process completed windows based on the ticker fire time
			sugar.Debugw("Ticker fired, processing completed windows", zap.Time("tick_time", tickTime))
			c.flushWindows(tickTime)

		case <-ctx.Done():
			sugar.Info("Context cancelled, stopping calculator. Processing final windows...")
			c.flushWindows(time.Now())
			return ctx.Err()
		}
	}
}

// processMessage determines the window and delegates feature processing.
func (c *Calculator) processMessage(msg message.DynamicMessage) {
	now := time.Now() // Determine window end time based on processing time
	windowDuration := c.config.WindowSize
	windowEnd := now.Truncate(windowDuration).Add(windowDuration)

	for _, featureCfg := range c.featuresToRun {
		c.updateFeatureStats(msg, featureCfg, windowEnd)
	}
}

// updateFeatureStats handles stats update for a single feature within its window.
// It gets the stats struct, updates basic counts, and delegates specific processing.
func (c *Calculator) updateFeatureStats(msg message.DynamicMessage, featureCfg config.FeatureConfig, windowEnd time.Time) {
	featureName := featureCfg.Name

	// Check if the feature is present in the message
	stats := c.getOrCreateFeatureStats(windowEnd, featureName)

	// Update basic stats
	stats.count++

	// Check for null value first
	if !msg.HasNonNull(featureName) {
		stats.nullCount++
		return
	}

	// Process non-null value based on metric type
	processed := c.processNonNullValue(stats, msg, featureCfg)

	// Log a warning if a non-null value couldn't be processed according to its type
	if !processed {
		c.logger.Sugar().Warnw("Non-null value could not be processed for feature",
			zap.String("feature_name", featureName),
			zap.String("metric_type", featureCfg.MetricType),
			zap.Any("value_snippet", msg.GetFieldSnippet(featureName, 50)),
			zap.Time("window_end", windowEnd),
		)
	}
}

// getOrCreateFeatureStats retrieves or initializes the stats struct for a given window/feature.
// It acquires and releases the lock internally.
func (c *Calculator) getOrCreateFeatureStats(windowEnd time.Time, featureName string) *FeatureStats {
	c.mu.Lock()
	defer c.mu.Unlock()

	windowState, exists := c.windowStates[windowEnd]
	if !exists {
		windowStart := windowEnd.Add(-c.config.WindowSize)
		windowState = newWindowInfo(windowStart, windowEnd)
		c.windowStates[windowEnd] = windowState
		c.logger.Debug("Created new state for window", zap.Time("window_end", windowEnd))
	}

	stats, exists := windowState.features[featureName]
	if !exists {
		stats = &FeatureStats{}
		windowState.features[featureName] = stats
	}
	return stats
}

// flushWindows finds windows completed by 'cutoffTime', calculates their stats,
// sends results downstream, and removes them from the state.
func (c *Calculator) flushWindows(cutoffTime time.Time) {
	completedWindows := c.collectAndRemoveCompletedWindows(cutoffTime)

	if len(completedWindows) == 0 {
		return
	}

	c.logger.Debug("Processing completed windows",
		zap.Time("cutoff_time", cutoffTime),
		zap.Int("window_count", len(completedWindows)),
	)

	// Process each completed window outside the main lock for calculations/sending
	for windowEnd, windowState := range completedWindows {
		c.processAndSendWindowResults(windowEnd, windowState)
	}
}

// collectAndRemoveCompletedWindows identifies completed windows and removes them from internal state.
// Returns a map of windowInfo pointers to process. MUST be called with the mutex held.
func (c *Calculator) collectAndRemoveCompletedWindows(cutoffTime time.Time) map[time.Time]*windowInfo {
	c.mu.Lock()
	defer c.mu.Unlock()

	windowsToProcess := make(map[time.Time]*windowInfo)
	for windowEnd, windowState := range c.windowStates {
		// A window is complete if its end time is less than or equal to the cutoff
		if !windowEnd.After(cutoffTime) {
			windowsToProcess[windowEnd] = windowState
			delete(c.windowStates, windowEnd)
		}
	}
	return windowsToProcess
}

// processAndSendWindowResults calculates final stats and sends them downstream.
// Accepts windowInfo struct.
func (c *Calculator) processAndSendWindowResults(windowEnd time.Time, windowState *windowInfo) {
	sugar := c.logger.Sugar()
	sugar.Debugw("Flushing window",
		zap.Time("window_end", windowEnd),
		zap.Int("feature_count", len(windowState.features)), // Use features map from windowInfo
	)

	for featureName, stats := range windowState.features {
		if stats.count == 0 {
			continue
		}

		mean, variance := c.calculateMeanVariance(stats, featureName, windowState.windowStart)

		result := AggregationResult{
			FeatureName: featureName,
			WindowStart: windowState.windowStart,
			WindowEnd:   windowEnd,
			Count:       stats.count,
			NullCount:   stats.nullCount,
			Mean:        mean,
			Variance:    variance,
		}

		select {
		case c.output <- result:
			sugar.Debugw("Sent aggregation result", zap.String("feature_name", featureName), zap.Time("window_end", windowEnd))
		default:
			sugar.Warnw("Calculator output channel full, dropping result",
				zap.String("feature_name", featureName),
				zap.Time("window_end", windowEnd),
			)
		}
	}
}
