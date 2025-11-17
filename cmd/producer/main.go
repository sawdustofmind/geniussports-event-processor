package main

import (
	"context"
	"errors"
	"flag"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"

	"github.com/sawdustofmind/geniussports-event-processor/internal/log"
	"github.com/sawdustofmind/geniussports-event-processor/internal/producer"
)

func run() int {
	filePath := flag.String("file", "PIT_LAC.txt", "Path to the data file")
	consumerURL := flag.String("consumer", "http://localhost:8080", "Consumer service URL")
	speed := flag.Duration("speed", 100*time.Millisecond, "delay between sending fixture messases")
	flag.Parse()

	log.Info("Starting Producer Service",
		zap.String("file", *filePath),
		zap.String("consumer_url", *consumerURL),
		zap.Duration("speed", *speed),
	)

	// Create sender
	sender := producer.NewSender(*consumerURL)

	// Setup context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		log.Info("Shutdown signal received, stopping")
		cancel()
	}()

	// Parse the file
	log.Info("Parsing file")
	messages, err := producer.ParseFile(*filePath)
	if err != nil {
		log.Error("Error parsing file", zap.Error(err))
		return 1
	}

	firstTimestamp := messages[0].OriginalTimestamp
	lastTimestamp := messages[len(messages)-1].OriginalTimestamp

	log.Info("File parsed successfully",
		zap.Int("message_count", len(messages)),
		zap.Time("first_timestamp", firstTimestamp),
		zap.Time("last_timestamp", lastTimestamp),
	)

	messagesCh := make(chan producer.ParsedMessage, len(messages))
	go func() {
		for _, msg := range messages {
			messagesCh <- msg
		}
	}()

	// Replay messages
	log.Info("Starting message replay")
	if err := sender.ReplayMessages(ctx, messagesCh, *speed); err != nil {
		if !errors.Is(err, context.Canceled) {
			log.Error("Error replaying messages", zap.Error(err))
			return 1
		}
	}

	log.Info("Producer finished successfully")
	return 0
}

func main() {
	// Initialize global logger
	if err := log.Init(true); err != nil {
		panic(err)
	}
	defer func() {
		_ = log.Sync()
	}()

	os.Exit(run())
}
