package config

import "errors"

var (
	ErrReadingConfigFile         = errors.New("failed to read config file")
	ErrUnmarshallingConfig       = errors.New("failed to unmarshal config")
	ErrEmptyKafkaBrokers         = errors.New("kafka brokers list cannot be empty")
	ErrEmptyKafkaTopic           = errors.New("kafka topic cannot be empty")
	ErrEmptyKafkaGroupID         = errors.New("kafka groupID cannot be empty")
	ErrInvalidPipelineWindowSize = errors.New("pipeline windowSize must be positive")
	ErrConfigFileMissing         = errors.New("config file not found")
)
