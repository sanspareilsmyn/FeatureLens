package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/sanspareilsmyn/featurelens/internal/config"
	"github.com/sanspareilsmyn/featurelens/internal/logging"
	"github.com/sanspareilsmyn/featurelens/internal/pipeline"
)

var (
	configFile = flag.String("config", "configs/config.dev.yaml", "Path to the configuration file")
	logger     *zap.Logger
)

func main() {
	// Initialize Configuration
	flag.Parse()

	cfg, err := config.Load(*configFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "FATAL: Failed to load configuration from %s: %v\n", *configFile, err)
		os.Exit(1)
	}

	// Initialize Logger
	var logErr error
	logger, logErr = logging.NewLogger(cfg.Log)
	if logErr != nil {
		fmt.Fprintf(os.Stderr, "FATAL: Failed to initialize logger: %v\n", logErr)
		os.Exit(1)
	}
	defer func() {
		_ = logger.Sync() // Flush buffered logs on exit
	}()

	sugar := logger.Sugar()
	sugar.Infow("Logger initialized",
		"level", cfg.Log.Level,
		"format", cfg.Log.Format,
	)
	sugar.Infow("Configuration loaded successfully", "path", *configFile)

	// Start Prometheus Metrics Server
	metricsAddr := ":8081"
	metricsSrv := &http.Server{Addr: metricsAddr}

	go func() {
		sugar.Infow("Starting Prometheus metrics server", "address", metricsAddr)
		http.Handle("/metrics", promhttp.Handler())
		if err := metricsSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			sugar.Errorw("Metrics server failed unexpectedly", "error", err)
		}
		sugar.Info("Metrics server stopped.")
	}()

	// Initialize Pipeline
	sugar.Info("Initializing pipeline...")
	pipe, err := pipeline.New(cfg, logger)
	if err != nil {
		sugar.Fatalw("Failed to initialize pipeline", "error", err)
	}
	sugar.Info("Monitoring pipeline initialized")

	// Handle Graceful Shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-signals
		sugar.Infow("Received signal, initiating shutdown...", "signal", sig.String())
		cancel()
	}()

	// Run Pipeline
	sugar.Info("Starting monitoring pipeline...")
	runErr := pipe.Run(ctx)

	// Graceful Shutdown of Metrics Server
	sugar.Info("Attempting to shut down metrics server gracefully...")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	if err := metricsSrv.Shutdown(shutdownCtx); err != nil {
		sugar.Warnw("Metrics server shutdown error", "error", err)
	} else {
		sugar.Info("Metrics server shutdown complete.")
	}

	// Evaluate Pipeline Result
	finalLogLevel := zapcore.InfoLevel
	shutdownReason := "gracefully"
	var finalErrorField = zap.Skip()

	switch {
	case runErr == nil:
		sugar.Info("Pipeline execution completed without error.")
	case errors.Is(runErr, context.Canceled):
		sugar.Info("Pipeline execution cancelled (expected on shutdown).")
		shutdownReason = "gracefully via signal"
	default: // Unexpected error
		shutdownReason = "due to pipeline error"
		finalLogLevel = zapcore.ErrorLevel
		finalErrorField = zap.Error(runErr)
		sugar.Errorw("Pipeline execution stopped unexpectedly", zap.Error(runErr))
	}

	finalMessage := fmt.Sprintf("Pipeline shutdown %s.", shutdownReason)
	logger.Log(finalLogLevel, finalMessage,
		zap.String("reason", shutdownReason),
		finalErrorField,
	)

	// Application Exit
	sugar.Info("Shutting down application...")
	sugar.Info("FeatureLens finished.")

	// Exit with appropriate code if there was an unexpected error from the pipeline
	if runErr != nil && !errors.Is(runErr, context.Canceled) {
		os.Exit(1)
	}
	os.Exit(0)
}
