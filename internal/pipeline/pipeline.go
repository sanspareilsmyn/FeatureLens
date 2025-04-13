// internal/pipeline/pipeline.go
package pipeline

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"go.uber.org/zap"

	"github.com/sanspareilsmyn/featurelens/internal/config"
	"github.com/sanspareilsmyn/featurelens/internal/message"
)

// Pipeline orchestrates the different stages: consumer, parsing, calculation, alerting.
type Pipeline struct {
	cfg        *config.Config
	consumer   *Consumer
	calculator *Calculator
	alerter    *Alerter
	logger     *zap.Logger

	rawMessages    chan []byte
	parsedMessages chan message.DynamicMessage
	aggResults     chan AggregationResult
}

// New creates and wires up a new monitoring pipeline.
func New(cfg *config.Config, logger *zap.Logger) (*Pipeline, error) {
	initLogger := logger.Named("pipeline.init")
	initLogger.Debug("Creating pipeline components...")

	// Create Channels
	const channelBufferSize = 100
	rawMessages := make(chan []byte, channelBufferSize)
	parsedMessages := make(chan message.DynamicMessage, channelBufferSize)
	aggResults := make(chan AggregationResult, channelBufferSize)
	initLogger.Debug("Channels created", zap.Int("bufferSize", channelBufferSize))

	// Initialize Components
	consumerLogger := logger.Named("consumer")
	consumerInstance, err := NewConsumer(cfg.Kafka, rawMessages, consumerLogger)
	if err != nil {
		initLogger.Error("Failed to create consumer", zap.Error(err))
		return nil, fmt.Errorf("%w: %w", ErrConsumerCreationFailed, err) // Use specific error
	}
	initLogger.Debug("Consumer created")

	calculatorLogger := logger.Named("calculator")
	calculatorInstance := NewCalculator(cfg.Pipeline, cfg.Features, parsedMessages, aggResults, calculatorLogger)
	initLogger.Debug("Calculator created")

	alerterLogger := logger.Named("alerter")
	alerterInstance := NewAlerter(cfg.Features, aggResults, alerterLogger)
	initLogger.Debug("Alerter created")

	// Create Pipeline
	p := &Pipeline{
		cfg:            cfg,
		consumer:       consumerInstance,
		calculator:     calculatorInstance,
		alerter:        alerterInstance,
		logger:         logger.Named("pipeline"),
		rawMessages:    rawMessages,
		parsedMessages: parsedMessages,
		aggResults:     aggResults,
	}

	initLogger.Info("Pipeline instance created successfully")
	return p, nil
}

// Run starts all pipeline components and waits for them to complete or context cancellation.
func (p *Pipeline) Run(ctx context.Context) error {
	sugar := p.logger.Sugar()
	var wg sync.WaitGroup
	pipelineErr := make(chan error, 4) // consumer, parser, calculator, alerter

	sugar.Info("Pipeline Run: Starting components...")

	// Start components as goroutines
	wg.Add(4)
	go p.runConsumer(ctx, &wg, pipelineErr)
	go p.runParser(ctx, &wg)
	go p.runCalculator(ctx, &wg, pipelineErr)
	go p.runAlerter(ctx, &wg, pipelineErr)

	// Wait for context cancellation or the first error from any component
	var firstErr error
	select {
	case <-ctx.Done():
		sugar.Info("Pipeline Run: Context cancelled. Waiting for components to finish...")
		firstErr = ctx.Err()
	case err := <-pipelineErr:
		sugar.Errorw("Pipeline Run: Received error from a component, initiating shutdown...", zap.Error(err))
		firstErr = err
	}

	// Wait for all component goroutines to complete their shutdown sequence
	sugar.Debug("Pipeline Run: Waiting on WaitGroup...")
	wg.Wait()
	sugar.Info("Pipeline Run: All components finished.")

	if firstErr != nil && !errors.Is(firstErr, context.Canceled) {
		return firstErr
	}
	return nil
}

// runConsumer executes the consumer component logic in a goroutine.
func (p *Pipeline) runConsumer(ctx context.Context, wg *sync.WaitGroup, errCh chan<- error) {
	defer wg.Done()
	defer func() {
		close(p.rawMessages)
		p.logger.Debug("Raw messages channel closed")
	}()

	p.logger.Debug("Starting consumer goroutine...")
	if err := p.consumer.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
		p.logger.Error("Consumer component exited with error", zap.Error(err))
		errCh <- fmt.Errorf("%w: %w", ErrConsumerRunFailed, err)
	} else if err == nil {
		p.logger.Debug("Consumer goroutine finished normally")
	} else {
		p.logger.Debug("Consumer goroutine cancelled gracefully")
	}
}

// runParser executes the parsing logic in a goroutine.
func (p *Pipeline) runParser(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()
	defer func() {
		close(p.parsedMessages)
		p.logger.Debug("Parsed messages channel closed")
	}()

	parserLogger := p.logger.Named("parser").Sugar()
	parserLogger.Debug("Starting parser goroutine...")

	for {
		select {
		case rawMsg, ok := <-p.rawMessages:
			if !ok {
				parserLogger.Debug("Parser finished (raw message channel closed).")
				return
			}

			parsedMsg, err := message.ParseDynamicJSON(rawMsg)
			if err != nil {
				parserLogger.Warnw("Failed to parse message, skipping", zap.Error(err))
				continue
			}

			// Send parsed message downstream or handle context cancellation
			select {
			case p.parsedMessages <- parsedMsg:

			case <-ctx.Done():
				parserLogger.Debug("Parser context cancelled during send.", zap.Error(ctx.Err()))
				return
			}

		case <-ctx.Done():
			parserLogger.Debug("Parser context cancelled while waiting for raw message.", zap.Error(ctx.Err()))
			return
		}
	}
}

// runCalculator executes the calculator component logic in a goroutine.
func (p *Pipeline) runCalculator(ctx context.Context, wg *sync.WaitGroup, errCh chan<- error) {
	defer wg.Done()
	defer func() {
		close(p.aggResults)
		p.logger.Debug("Aggregation results channel closed")
	}()

	p.logger.Debug("Starting calculator goroutine...")
	if err := p.calculator.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
		p.logger.Error("Calculator component exited with error", zap.Error(err))
		errCh <- fmt.Errorf("%w: %w", ErrCalculatorRunFailed, err)
	} else if err == nil {
		p.logger.Debug("Calculator goroutine finished normally")
	} else { // errors.Is(err, context.Canceled)
		p.logger.Debug("Calculator goroutine cancelled gracefully")
	}
}

// runAlerter executes the alerter component logic in a goroutine.
func (p *Pipeline) runAlerter(ctx context.Context, wg *sync.WaitGroup, errCh chan<- error) {
	defer wg.Done()

	p.logger.Debug("Starting alerter goroutine...")
	if err := p.alerter.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
		p.logger.Error("Alerter component exited with error", zap.Error(err))
		errCh <- fmt.Errorf("%w: %w", ErrAlerterRunFailed, err)
	} else if err == nil {
		p.logger.Debug("Alerter goroutine finished normally")
	} else { // errors.Is(err, context.Canceled)
		p.logger.Debug("Alerter goroutine cancelled gracefully")
	}
}

// Close is kept for potential future explicit cleanup needs outside the Run cycle.
func (p *Pipeline) Close() error {
	p.logger.Debug("Pipeline Close called (most cleanup handled by Run/context).")
	return nil
}
