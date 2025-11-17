package producer

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/sawdustofmind/geniussports-event-processor/internal/log"
	"github.com/sawdustofmind/geniussports-event-processor/internal/models"
)

type ParsedMessage struct {
	LineNumber int
	Message    models.Message
	// TODO: consider renaming raw body avoiding lack of information into updates
	OriginalTimestamp time.Time
}

func ParseFile(filePath string) ([]ParsedMessage, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			log.Error("Failed to close file", zap.Error(closeErr))
		}
	}()

	scanner := bufio.NewScanner(file)

	// Increase buffer size for large lines
	const maxCapacity = 10 * 1024 * 1024 // 10MB
	buf := make([]byte, maxCapacity)
	scanner.Buffer(buf, maxCapacity)

	var messages []ParsedMessage
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		// Skip the first line which is just "extracted_data"
		if lineNum == 1 && line == `"extracted_data"` {
			continue
		}

		// Skip empty lines
		if len(line) == 0 {
			continue
		}

		line = strings.ReplaceAll(line, `""`, `"`)
		// Handle the special format: lines are wrapped in quotes with doubled internal quotes
		// Example: "{"Header": {"Retry": 0}}"
		// Strip outer quotes and replace " with "
		if len(line) >= 2 && line[0] == '"' && line[len(line)-1] == '"' {
			// Remove outer quotes
			line = line[1 : len(line)-1]
			// Replace doubled quotes with single quotes
		}

		// Parse the JSON object
		var msg models.Message
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			log.Warn("Failed to parse line as JSON",
				zap.Int("line_number", lineNum),
				zap.Error(err),
			)
			continue
		}

		// Skip messages that don't have either Fixture or MatchState
		if msg.Fixture == nil && msg.AmericanFootballMatchState == nil {
			continue
		}
		if msg.Header.MessageGuid == "" || msg.Header.TimeStampUtc.IsZero() {
			continue
		}

		messages = append(messages, ParsedMessage{
			LineNumber:        lineNum,
			OriginalTimestamp: msg.Header.TimeStampUtc,
			Message:           msg,
		})
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading file: %w", err)
	}

	// Sort messages by timestamp
	sort.Slice(messages, func(i, j int) bool {
		return messages[i].OriginalTimestamp.Before(messages[j].OriginalTimestamp)
	})

	log.Info("Successfully parsed messages", zap.Int("message_count", len(messages)))
	return messages, nil
}
