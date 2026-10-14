package stores

import (
	"database/sql"
	"encoding/json"
	"errors"

	"coachia-backend/models"
)

type PendingChangeStore struct {
	db *sql.DB
}

func NewPendingChangeStore(db *sql.DB) *PendingChangeStore {
	return &PendingChangeStore{db: db}
}

func (s *PendingChangeStore) Create(userID int64, changeType string, payload json.RawMessage, reason string) (models.PendingChange, error) {
	res, err := s.db.Exec(`
		INSERT INTO pending_changes (user_id, change_type, payload, reason)
		VALUES (?, ?, ?, ?)
	`, userID, changeType, string(payload), reason)
	if err != nil {
		return models.PendingChange{}, err
	}

	id, err := res.LastInsertId()
	if err != nil {
		return models.PendingChange{}, err
	}

	return s.GetByID(id)
}

func (s *PendingChangeStore) GetByID(id int64) (models.PendingChange, error) {
	var c models.PendingChange
	var payload string
	err := s.db.QueryRow(`
		SELECT id, user_id, change_type, payload, reason, status, created_at, resolved_at
		FROM pending_changes WHERE id = ?
	`, id).Scan(&c.ID, &c.UserID, &c.ChangeType, &payload, &c.Reason, &c.Status, &c.CreatedAt, &c.ResolvedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return models.PendingChange{}, ErrNotFound
	}
	if err != nil {
		return models.PendingChange{}, err
	}
	c.Payload = json.RawMessage(payload)
	return c, nil
}

func (s *PendingChangeStore) ListPending(userID int64) ([]models.PendingChange, error) {
	rows, err := s.db.Query(`
		SELECT id, user_id, change_type, payload, reason, status, created_at, resolved_at
		FROM pending_changes WHERE user_id = ? AND status = 'pending' ORDER BY created_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []models.PendingChange{}
	for rows.Next() {
		var c models.PendingChange
		var payload string
		if err := rows.Scan(&c.ID, &c.UserID, &c.ChangeType, &payload, &c.Reason, &c.Status, &c.CreatedAt, &c.ResolvedAt); err != nil {
			return nil, err
		}
		c.Payload = json.RawMessage(payload)
		out = append(out, c)
	}
	return out, rows.Err()
}

func (s *PendingChangeStore) Resolve(id int64, status string) (models.PendingChange, error) {
	if _, err := s.db.Exec(`
		UPDATE pending_changes SET status = ?, resolved_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now') WHERE id = ?
	`, status, id); err != nil {
		return models.PendingChange{}, err
	}
	return s.GetByID(id)
}
