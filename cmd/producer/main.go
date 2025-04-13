package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/segmentio/kafka-go"
)

const (
	kafkaBroker = "localhost:9092"
	topic       = "feature-stream"
)

// Example Feature Message Structure (matches what FeatureLens expects)
type FeatureMessage struct {
	Timestamp   time.Time `json:"timestamp"`
	UserID      string    `json:"user_id"`
	FeatureA    *float64  `json:"feature_a"`
	FeatureB    *float64  `json:"feature_b"`
	FeatureC    *string   `json:"feature_c"`
	ProcessTime int       `json:"process_time_ms"`
}

func main() {
	writer := &kafka.Writer{
		Addr:     kafka.TCP(kafkaBroker),
		Topic:    topic,
		Balancer: &kafka.LeastBytes{},
	}
	defer func() {
		if err := writer.Close(); err != nil {
			log.Fatalf("Error closing kafka writer: %v", err)
		}
	}()
	log.Printf("Starting sample producer for topic: %s on broker: %s", topic, kafkaBroker)

	// Handle graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-signals
		log.Println("Shutdown signal received, stopping producer...")
		cancel()
	}()

	// Produce messages periodically
	ticker := time.NewTicker(1 * time.Second) // Produce message every second
	defer ticker.Stop()

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	for {
		select {
		case <-ticker.C:
			msg := generateSampleMessage(rng)
			msgBytes, err := json.Marshal(msg)
			if err != nil {
				log.Printf("Error marshalling message: %v", err)
				continue
			}

			err = writer.WriteMessages(ctx, kafka.Message{Value: msgBytes})
			if err != nil {
				if ctx.Err() != nil { // Check if context was cancelled (shutdown)
					log.Println("Context cancelled, exiting message loop.")
					return
				}
				log.Printf("Error writing message: %v", err)
			} else {
				log.Printf("Produced message: %s", string(msgBytes))
			}

		case <-ctx.Done():
			log.Println("Producer loop stopped.")
			return
		}
	}
}

// Generates a sample message with some randomness and potential nulls/outliers
func generateSampleMessage(rng *rand.Rand) FeatureMessage {
	now := time.Now()
	userID := fmt.Sprintf("user_%d", rng.Intn(1000))

	var featureA *float64
	// ~10% chance of being null
	if rng.Float64() > 0.1 {
		// Normal distribution around 10, stddev 2, plus occasional outlier
		val := 10.0 + rng.NormFloat64()*2.0
		if rng.Float64() < 0.02 { // 2% chance of outlier
			val += rng.Float64() * 30.0 // Add large positive offset
		}
		featureA = &val
	}

	var featureB *float64
	// ~5% chance of being null
	if rng.Float64() > 0.05 {
		val := 50.0 + rng.Float64()*10.0 // Uniform between 50 and 60
		featureB = &val
	}

	var featureC *string
	categories := []string{"A", "B", "C", "D"}
	// ~15% chance of being null
	if rng.Float64() > 0.15 {
		cat := categories[rng.Intn(len(categories))]
		featureC = &cat
	}

	processTime := 10 + rng.Intn(40) // 10-49 ms

	return FeatureMessage{
		Timestamp:   now,
		UserID:      userID,
		FeatureA:    featureA,
		FeatureB:    featureB,
		FeatureC:    featureC,
		ProcessTime: processTime,
	}
}
