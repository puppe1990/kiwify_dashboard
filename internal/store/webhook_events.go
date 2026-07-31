package store

import (
	"fmt"
	"time"
)

type WebhookEvent struct {
	ID          int64
	EventType   string
	PayloadJSON string
	HeadersJSON string
	ReceivedAt  time.Time
	ProcessedOK bool
}

func (s *SQLiteStore) InsertWebhookEvent(ev WebhookEvent) (int64, error) {
	processed := 0
	if ev.ProcessedOK {
		processed = 1
	}
	result, err := s.db.Exec(`
		INSERT INTO webhook_events (event_type, payload_json, headers_json, processed_ok)
		VALUES (?, ?, ?, ?)
	`, ev.EventType, ev.PayloadJSON, ev.HeadersJSON, processed)
	if err != nil {
		return 0, fmt.Errorf("insert webhook event: %w", err)
	}
	return result.LastInsertId()
}

func (s *SQLiteStore) ListWebhookEvents(limit, offset int) ([]WebhookEvent, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.db.Query(`
		SELECT id, event_type, payload_json, headers_json, received_at, processed_ok
		FROM webhook_events
		ORDER BY received_at DESC, id DESC
		LIMIT ? OFFSET ?
	`, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list webhook events: %w", err)
	}
	defer rows.Close()

	var events []WebhookEvent
	for rows.Next() {
		var ev WebhookEvent
		var processed int
		if err := rows.Scan(
			&ev.ID,
			&ev.EventType,
			&ev.PayloadJSON,
			&ev.HeadersJSON,
			&ev.ReceivedAt,
			&processed,
		); err != nil {
			return nil, fmt.Errorf("scan webhook event: %w", err)
		}
		ev.ProcessedOK = processed != 0
		events = append(events, ev)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list webhook events rows: %w", err)
	}
	if events == nil {
		events = []WebhookEvent{}
	}
	return events, nil
}
