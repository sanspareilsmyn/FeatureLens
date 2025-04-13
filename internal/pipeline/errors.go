package pipeline

import "errors"

var (
	ErrInvalidKafkaConfig     = errors.New("invalid Kafka configuration provided")
	ErrKafkaFetchFailed       = errors.New("failed to fetch message from Kafka")
	ErrConsumerCreationFailed = errors.New("failed to create consumer")
	ErrConsumerRunFailed      = errors.New("consumer component failed")
	ErrCalculatorRunFailed    = errors.New("calculator component failed")
	ErrAlerterRunFailed       = errors.New("alerter component failed")
)
