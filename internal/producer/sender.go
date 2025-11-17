package producer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"go.uber.org/zap"

	"github.com/sawdustofmind/geniussports-event-processor/internal/log"
	"github.com/sawdustofmind/geniussports-event-processor/internal/models"
)

type Sender struct {
	httpClient  *http.Client
	consumerURL string
}

func NewSender(consumerURL string) *Sender {
	return &Sender{
		consumerURL: consumerURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (s *Sender) SendHeartbeat(ctx context.Context) error {
	url := fmt.Sprintf("%s/heartbeat", s.consumerURL)

	req, err := http.NewRequestWithContext(ctx, "POST", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create heartbeat request: %w", err)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send heartbeat: %w", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			log.Error("Failed to close heartbeat response body", zap.Error(closeErr))
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("heartbeat returned status %d", resp.StatusCode)
	}

	return nil
}

func (s *Sender) SendMessage(ctx context.Context, msg models.Message) error {
	url := fmt.Sprintf("%s/process-msg", s.consumerURL)

	// Update timestamp to current time
	msg.Header.TimeStampUtc = time.Now().UTC()

	jsonData, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			log.Error("Failed to close message response body", zap.Error(closeErr))
		}
	}()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("message processing returned status %d and failed to read body", resp.StatusCode)
		}
		return fmt.Errorf("message processing returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

func (s *Sender) ReplayMessages(ctx context.Context, messages <-chan ParsedMessage, speed time.Duration) error {
	// Send initial heartbeat
	if err := s.SendHeartbeat(ctx); err != nil {
		return err
	}

	heartbeatTicker := time.NewTicker(60 * time.Second)
	defer heartbeatTicker.Stop()
	lastSent := time.Time{}

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-heartbeatTicker.C:
			if err := s.SendHeartbeat(ctx); err != nil {
				log.Error("Heartbeat failed", zap.Error(err))
			} else {
				log.Info("Heartbeat sent successfully")
			}
		case msg := <-messages:
			now := time.Now()
			if !lastSent.IsZero() && now.Sub(lastSent) <= speed {
				select {
				case <-ctx.Done():
					return nil
				case <-time.After(speed - now.Sub(lastSent)):
				}
			}

			if err := s.SendMessage(ctx, msg.Message); err != nil {
				log.Error("Failed to send message",
					zap.Int("message_number", msg.LineNumber),
					zap.String("message_guid", msg.Message.Header.MessageGuid),
					zap.Error(err),
				)
			} else {
				msgType := "MatchState"
				if msg.Message.Fixture != nil {
					msgType = "Fixture"
				}
				log.Info("Sent message",
					zap.Int("message_number", msg.LineNumber),
					zap.Int("total_messages", len(messages)),
					zap.String("message_type", msgType),
					zap.Time("original_timestamp", msg.OriginalTimestamp),
				)
				lastSent = now
			}
		}
	}
}
