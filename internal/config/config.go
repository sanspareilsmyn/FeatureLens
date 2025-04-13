package config

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

const (
	defaultKafkaGroupID   = "featurelens-default-group"
	defaultPipelineWindow = 1 * time.Minute
	defaultLogLevel       = "info"
	defaultLogFormat      = "console"
	defaultLogFileEnabled = false
	defaultLogDirectory   = "log"
	defaultLogFilename    = "app.log"
	defaultLogMaxSizeMB   = 100
	defaultLogMaxBackups  = 3
	defaultLogMaxAgeDays  = 7
	defaultLogCompress    = false

	// Environment variable prefix
	envPrefix = "FEATURELENS"
)

type Config struct {
	Kafka    KafkaConfig     `mapstructure:"kafka"`
	Pipeline PipelineConfig  `mapstructure:"pipeline"`
	Features []FeatureConfig `mapstructure:"features"`
	Log      LogConfig       `mapstructure:"log"`
}

type KafkaConfig struct {
	Brokers []string `mapstructure:"brokers"`
	Topic   string   `mapstructure:"topic"`
	GroupID string   `mapstructure:"groupID"`
}

type PipelineConfig struct {
	WindowSize time.Duration `mapstructure:"windowSize"`
}

type FeatureConfig struct {
	Name       string     `mapstructure:"name"`
	MetricType string     `mapstructure:"metricType"` // e.g., "numerical", "categorical"
	Thresholds Thresholds `mapstructure:"thresholds"`
}

type LogConfig struct {
	Level              string `mapstructure:"level"`
	Format             string `mapstructure:"format"`
	FileLoggingEnabled bool   `mapstructure:"fileLoggingEnabled"`
	Directory          string `mapstructure:"directory"`
	Filename           string `mapstructure:"filename"`
	MaxSize            int    `mapstructure:"maxSize"`    // Max size in MB
	MaxBackups         int    `mapstructure:"maxBackups"` // Max backup files
	MaxAge             int    `mapstructure:"maxAge"`     // Max days to retain
	Compress           bool   `mapstructure:"compress"`   // Compress rotated files?
}

type Thresholds struct {
	NullRate  *float64 `mapstructure:"nullRate"`
	MeanMin   *float64 `mapstructure:"meanMin"`
	MeanMax   *float64 `mapstructure:"meanMax"`
	StdDevMin *float64 `mapstructure:"stdDevMin"`
	StdDevMax *float64 `mapstructure:"stdDevMax"`
}

// Load initializes viper, reads config, applies defaults, unmarshals, and validates.
func Load(configPath string) (*Config, error) {
	v := viper.New()
	configureViper(v, configPath)

	// Set default values before reading config source .yaml
	setDefaults(v)

	// Read configuration from file (error if mandatory file is missing)
	if err := readConfigFile(v); err != nil {
		return nil, err
	}

	// Unmarshal the configuration
	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrUnmarshallingConfig, err)
	}

	if err := validateConfig(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// configureViper sets up viper instance for file and environment variables.
func configureViper(v *viper.Viper, configPath string) {
	if configPath != "" {
		v.SetConfigFile(configPath)
	}

	v.SetEnvPrefix(envPrefix)
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
}

// setDefaults applies default configuration values using Viper.
func setDefaults(v *viper.Viper) {
	v.SetDefault("kafka.groupID", defaultKafkaGroupID)
	v.SetDefault("pipeline.windowSize", defaultPipelineWindow)
	v.SetDefault("log.level", defaultLogLevel)
	v.SetDefault("log.format", defaultLogFormat)
	v.SetDefault("log.fileLoggingEnabled", defaultLogFileEnabled)
	v.SetDefault("log.directory", defaultLogDirectory)
	v.SetDefault("log.filename", defaultLogFilename)
	v.SetDefault("log.maxSize", defaultLogMaxSizeMB)
	v.SetDefault("log.maxBackups", defaultLogMaxBackups)
	v.SetDefault("log.maxAge", defaultLogMaxAgeDays)
	v.SetDefault("log.compress", defaultLogCompress)
}

// readConfigFile attempts to read the configuration file specified in viper.
func readConfigFile(v *viper.Viper) error {
	err := v.ReadInConfig()
	if err != nil {
		var configFileNotFoundError viper.ConfigFileNotFoundError
		if errors.As(err, &configFileNotFoundError) {
			return ErrConfigFileMissing
		}
		return fmt.Errorf("%w: %w", ErrReadingConfigFile, err)
	}
	return nil
}

func validateConfig(cfg *Config) error {
	if len(cfg.Kafka.Brokers) == 0 {
		return ErrEmptyKafkaBrokers
	}
	if cfg.Kafka.Topic == "" {
		return ErrEmptyKafkaTopic
	}
	if cfg.Kafka.GroupID == "" {
		return ErrEmptyKafkaGroupID
	}
	if cfg.Pipeline.WindowSize <= 0 {
		return ErrInvalidPipelineWindowSize
	}
	return nil
}
