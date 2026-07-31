package store

import (
	"fmt"
	"time"
)

type AuditLog struct {
	ID             int64
	UserID         int64
	Action         string
	ResourceType   string
	ResourceID     string
	RequestSummary string
	ResponseStatus int
	ResponseBody   string
	IP             string
	CreatedAt      time.Time
}

func (s *SQLiteStore) InsertAuditLog(log AuditLog) (int64, error) {
	result, err := s.db.Exec(`
		INSERT INTO audit_logs (
			user_id, action, resource_type, resource_id,
			request_summary, response_status, response_body, ip
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`,
		log.UserID,
		log.Action,
		log.ResourceType,
		log.ResourceID,
		log.RequestSummary,
		log.ResponseStatus,
		log.ResponseBody,
		log.IP,
	)
	if err != nil {
		return 0, fmt.Errorf("insert audit log: %w", err)
	}
	return result.LastInsertId()
}

func (s *SQLiteStore) ListAuditLogs(limit, offset int) ([]AuditLog, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.db.Query(`
		SELECT id, user_id, action, resource_type, resource_id,
		       request_summary, response_status, response_body, ip, created_at
		FROM audit_logs
		ORDER BY created_at DESC, id DESC
		LIMIT ? OFFSET ?
	`, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list audit logs: %w", err)
	}
	defer rows.Close()

	var logs []AuditLog
	for rows.Next() {
		var l AuditLog
		if err := rows.Scan(
			&l.ID,
			&l.UserID,
			&l.Action,
			&l.ResourceType,
			&l.ResourceID,
			&l.RequestSummary,
			&l.ResponseStatus,
			&l.ResponseBody,
			&l.IP,
			&l.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan audit log: %w", err)
		}
		logs = append(logs, l)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list audit logs rows: %w", err)
	}
	if logs == nil {
		logs = []AuditLog{}
	}
	return logs, nil
}
