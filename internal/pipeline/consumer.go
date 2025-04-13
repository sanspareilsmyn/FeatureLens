// internal/pipeline/consumer.go
package pipeline

import (
	"context"
	"errors"
	"fmt"
	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"

	"github.com/sanspareilsmyn/featurelens/internal/config"
)

type kafkaZapLogger struct {
	log *zap.Logger
}

func (l kafkaZapLogger) Printf(msg string, args ...interface{}) {
	l.log.Info(fmt.Sprintf(msg, args...))
}

type kafkaZapErrorLogger struct {
	log *zap.Logger
}

func (l kafkaZapErrorLogger) Printf(msg string, args ...interface{}) {
	l.log.Error(fmt.Sprintf(msg, args...))
}

// Consumer reads messages from a Kafka topic using kafka-go library.
type Consumer struct {
	reader *kafka.Reader
	output chan<- []byte
	cfg    config.KafkaConfig
	logger *zap.Logger
}

// NewConsumer creates and configures a new Kafka consumer instance.
func NewConsumer(cfg config.KafkaConfig, output chan<- []byte, logger *zap.Logger) (*Consumer, error) {
	if len(cfg.Brokers) == 0 || cfg.Topic == "" || cfg.GroupID == "" {
		logger.Error("Kafka configuration validation failed",
			zap.Strings("brokers", cfg.Brokers),
			zap.String("topic", cfg.Topic),
			zap.String("group_id", cfg.GroupID),
		)
		return nil, ErrInvalidKafkaConfig
	}

	readerCfg := kafka.ReaderConfig{
		Brokers:     cfg.Brokers,
		GroupID:     cfg.GroupID,
		Topic:       cfg.Topic,
		Logger:      kafkaZapLogger{logger.Named("kafka-reader").WithOptions(zap.AddCallerSkip(1))},
		ErrorLogger: kafkaZapErrorLogger{logger.Named("kafka-reader-error").WithOptions(zap.AddCallerSkip(1))},
	}
	r := kafka.NewReader(readerCfg)

	logger.Info("Kafka consumer created",
		zap.String("topic", cfg.Topic),
		zap.String("group_id", cfg.GroupID),
		zap.Strings("brokers", cfg.Brokers),
		zap.Duration("commit_interval", readerCfg.CommitInterval),
		zap.Duration("max_wait", readerCfg.MaxWait),
		zap.Int("min_bytes", readerCfg.MinBytes),
		zap.Int("max_bytes", readerCfg.MaxBytes),
	)

	return &Consumer{
		reader: r,
		output: output,
		cfg:    cfg,
		logger: logger,
	}, nil
}

// Run starts the consumer message reading loop.
// It blocks until the context is cancelled or an unrecoverable error occurs.
func (c *Consumer) Run(ctx context.Context) error {
	sugar := c.logger.Sugar()
	sugar.Info("Starting Kafka consumer loop...")

	defer func() {
		sugar.Info("Closing Kafka consumer reader...")
		if err := c.reader.Close(); err != nil {
			sugar.Errorw("Failed to close Kafka reader cleanly", zap.Error(err))
		} else {
			sugar.Info("Kafka consumer reader closed successfully.")
		}
		sugar.Info("Kafka consumer loop stopped.")
	}()

	for {
		// FetchMessage blocks until a message is available or context is cancelled/deadline exceeded.
		m, err := c.reader.FetchMessage(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				c.logger.Debug("Context cancelled or deadline exceeded, stopping consumer fetch loop.", zap.Error(err))
				return context.Canceled
			}
			c.logger.Error("Error fetching message from Kafka", zap.Error(err))
			return fmt.Errorf("%w: %w", ErrKafkaFetchFailed, err)
		}

		select {
		case c.output <- m.Value:
			continue

		case <-ctx.Done():
			c.logger.Debug("Context cancelled while sending message downstream.", zap.Error(ctx.Err()))
			return context.Canceled
		}
	}
}

// Close cleans up the consumer resources. Provided for potential explicit cleanup needs,
// although Run()'s defer handles the primary reader closing.
func (c *Consumer) Close() error {
	c.logger.Info("Explicit Close() called on Kafka consumer...")
	return nil
}
