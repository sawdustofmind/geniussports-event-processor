package consumer

import (
	"context"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"

	"github.com/sawdustofmind/geniussports-event-processor/internal/log"
	"github.com/sawdustofmind/geniussports-event-processor/internal/models"
)

type Handler struct {
	redisClient *redis.Client
}

func NewHandler(redisAddr string) (*Handler, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: "", // no password set
		DB:       0,  // use default DB
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	log.Info("Successfully connected to Redis", zap.String("address", redisAddr))
	return &Handler{
		redisClient: client,
	}, nil
}

func (h *Handler) ProcessMessage(ctx context.Context, msg models.Message) error {
	// Process Fixture message
	if msg.Fixture != nil {
		return h.processFixture(ctx, msg.Fixture, msg.Header.TimeStampUtc)
	}

	// Process MatchState message
	if msg.AmericanFootballMatchState != nil {
		return h.processMatchState(ctx, msg.AmericanFootballMatchState, msg.Header.TimeStampUtc)
	}

	// Log the message structure to help debug
	log.Error("Unknown message type received",
		zap.String("message_guid", msg.Header.MessageGuid),
		zap.Bool("has_fixture", msg.Fixture != nil),
		zap.Bool("has_match_state", msg.AmericanFootballMatchState != nil),
	)
	return fmt.Errorf("unknown message type")
}

const (
	Home = "Home"
	Away = "Away"
)

func (h *Handler) processFixture(ctx context.Context, fixture *models.Fixture, timestamp time.Time) error {
	homeTeam := ""
	awayTeam := ""

	for _, comp := range fixture.Competitors {
		if comp.HomeAway == Home {
			homeTeam = comp.Name
		} else if comp.HomeAway == Away {
			awayTeam = comp.Name
		}
	}
	if homeTeam == "" || awayTeam == "" {
		return fmt.Errorf("fixture %d missing home or away team", fixture.ID)
	}

	key := fmt.Sprintf("fixture:%d", fixture.ID)
	scoreData := map[string]interface{}{
		"away_team":      awayTeam,
		"home_team":      homeTeam,
		"fixture_status": fixture.Status,
		"start_time":     fixture.StartTimeUtc.Format(time.RFC3339),
		"timestamp":      timestamp.Format(time.RFC3339),
	}

	if err := h.redisClient.HSet(ctx, key, scoreData).Err(); err != nil {
		return fmt.Errorf("failed to store latest score: %w", err)
	}

	log.Info("Stored match state", zap.Any("state", scoreData))
	return nil
}

func (h *Handler) processMatchState(ctx context.Context, state *models.AmericanFootballMatchState, timestamp time.Time) error {
	// Add to sorted set with timestamp as score for chronological ordering
	key := fmt.Sprintf("fixture:%s", state.FixtureId)
	scoreData := map[string]interface{}{
		"away":         state.Score.Away,
		"home":         state.Score.Home,
		"is_confirmed": state.Score.IsConfirmed,
		"period_num":   state.Period.Number,
		"period_type":  state.Period.Type,
		"is_running":   state.GameTime.IsRunning,
		"timestamp":    timestamp.Format(time.RFC3339),
	}

	if err := h.redisClient.HSet(ctx, key, scoreData).Err(); err != nil {
		return fmt.Errorf("failed to store latest score: %w", err)
	}

	log.Info("Stored match state", zap.Any("state", state))
	return nil
}

func (h *Handler) Close() error {
	return h.redisClient.Close()
}
