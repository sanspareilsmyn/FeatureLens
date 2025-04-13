package logging

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/sanspareilsmyn/featurelens/internal/config"
	"gopkg.in/natefinch/lumberjack.v2"
)

// NewLogger initializes a zap logger based on the provided configuration,
// supporting both console and rotating file output.
func NewLogger(cfg config.LogConfig) (*zap.Logger, error) {
	level, err := parseLevel(cfg.Level)
	if err != nil {
		fmt.Fprintf(os.Stderr, "WARN: %v, defaulting to INFO level\n", err)
		level = zapcore.InfoLevel
	}

	isConsole := strings.ToLower(cfg.Format) == "console"
	isDevelopment := (level == zapcore.DebugLevel) || isConsole

	cores := []zapcore.Core{}

	// Configure Console Output
	if isConsole {
		consoleEncoder := buildEncoder(true)
		consoleDebugging := zapcore.Lock(os.Stdout)
		consoleErrors := zapcore.Lock(os.Stderr)
		// Filter levels for different console outputs
		coreConsoleInfo := zapcore.NewCore(consoleEncoder, consoleDebugging, zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
			return lvl >= level && lvl < zapcore.ErrorLevel // Log configured level up to Warn on stdout
		}))
		coreConsoleError := zapcore.NewCore(consoleEncoder, consoleErrors, zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
			return lvl >= level && lvl >= zapcore.ErrorLevel // Log Error and above on stderr
		}))
		cores = append(cores, coreConsoleInfo, coreConsoleError)
	}

	// Configure File Output
	if cfg.FileLoggingEnabled {
		if err := os.MkdirAll(cfg.Directory, 0755); err != nil {
			return nil, fmt.Errorf("failed to create log directory '%s': %w", cfg.Directory, err)
		}

		// Configure lumberjack
		ljack := &lumberjack.Logger{
			Filename:   filepath.Join(cfg.Directory, cfg.Filename),
			MaxSize:    cfg.MaxSize,    // megabytes
			MaxBackups: cfg.MaxBackups, // files
			MaxAge:     cfg.MaxAge,     // days
			Compress:   cfg.Compress,   // disabled by default
		}
		fileEncoder := buildEncoder(false)
		fileSyncer := zapcore.AddSync(ljack)

		coreFile := zapcore.NewCore(fileEncoder, fileSyncer, level)
		cores = append(cores, coreFile)
	}

	// Combine cores if multiple outputs are configured
	var combinedCore zapcore.Core
	if len(cores) == 0 {
		return nil, fmt.Errorf("no logging outputs configured (neither console nor file enabled)")
	} else if len(cores) == 1 {
		combinedCore = cores[0]
	} else {
		combinedCore = zapcore.NewTee(cores...)
	}

	// --- Build Logger Options ---
	loggerOptions := []zap.Option{
		zap.AddCaller(),
		zap.AddCallerSkip(1),
	}
	if isDevelopment {
		loggerOptions = append(loggerOptions, zap.Development(), zap.AddStacktrace(zapcore.WarnLevel))
	} else {
		loggerOptions = append(loggerOptions, zap.AddStacktrace(zapcore.ErrorLevel))
	}

	logger := zap.New(combinedCore, loggerOptions...)

	logger.Debug("Zap logger constructed",
		zap.String("final_level", level.String()),
		zap.String("console_format", cfg.Format),
		zap.Bool("file_logging_enabled", cfg.FileLoggingEnabled),
		zap.String("file_path", filepath.Join(cfg.Directory, cfg.Filename)),
		zap.Bool("development_mode", isDevelopment),
	)

	return logger, nil
}

func parseLevel(levelStr string) (zapcore.Level, error) {
	var level zapcore.Level
	err := level.UnmarshalText([]byte(strings.ToLower(levelStr)))
	if err != nil {
		return zapcore.InfoLevel, fmt.Errorf("invalid log level '%s'", levelStr)
	}
	return level, nil
}

func buildEncoder(useConsoleStyle bool) zapcore.Encoder {
	var encoderConfig zapcore.EncoderConfig
	if useConsoleStyle {
		encoderConfig = zap.NewDevelopmentEncoderConfig()
		encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
		encoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder // Color for console
		return zapcore.NewConsoleEncoder(encoderConfig)
	} else { // Typically for production / file output (JSON)
		encoderConfig = zap.NewProductionEncoderConfig()
		encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder // ISO8601 is standard
		encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
		return zapcore.NewJSONEncoder(encoderConfig)
	}
}
