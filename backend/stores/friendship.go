package stores

import (
	"database/sql"
	"errors"

	"coachia-backend/models"
)

type FriendshipStore struct {
	db *sql.DB
}

func NewFriendshipStore(db *sql.DB) *FriendshipStore {
	return &FriendshipStore{db: db}
}

func (s *FriendshipStore) Create(requesterID, recipientID int64) (models.Friendship, error) {
	res, err := s.db.Exec(`INSERT INTO friendships (requester_id, recipient_id) VALUES (?, ?)`, requesterID, recipientID)
	if err != nil {
		return models.Friendship{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return models.Friendship{}, err
	}
	return s.GetByID(id)
}

func (s *FriendshipStore) GetByID(id int64) (models.Friendship, error) {
	var f models.Friendship
	err := s.db.QueryRow(`
		SELECT id, requester_id, recipient_id, status, created_at, resolved_at FROM friendships WHERE id = ?
	`, id).Scan(&f.ID, &f.RequesterID, &f.RecipientID, &f.Status, &f.CreatedAt, &f.ResolvedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return models.Friendship{}, ErrNotFound
	}
	return f, err
}

// FindBetween returns any existing friendship (any status) between the two
// users, regardless of who requested — used to block duplicate/overlapping
// requests between the same pair.
func (s *FriendshipStore) FindBetween(userA, userB int64) (models.Friendship, error) {
	var f models.Friendship
	err := s.db.QueryRow(`
		SELECT id, requester_id, recipient_id, status, created_at, resolved_at
		FROM friendships
		WHERE (requester_id = ? AND recipient_id = ?) OR (requester_id = ? AND recipient_id = ?)
		ORDER BY created_at DESC LIMIT 1
	`, userA, userB, userB, userA).Scan(&f.ID, &f.RequesterID, &f.RecipientID, &f.Status, &f.CreatedAt, &f.ResolvedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return models.Friendship{}, ErrNotFound
	}
	return f, err
}

// ListIncomingPending returns pending requests sent TO userID, enriched with
// the requester's email so the frontend can show "X quiere agregarte" with
// no second lookup.
func (s *FriendshipStore) ListIncomingPending(userID int64) ([]models.Friendship, error) {
	rows, err := s.db.Query(`
		SELECT f.id, f.requester_id, f.recipient_id, f.status, f.created_at, f.resolved_at, u.id, u.email
		FROM friendships f
		JOIN users u ON u.id = f.requester_id
		WHERE f.recipient_id = ? AND f.status = 'pending'
		ORDER BY f.created_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	requests := []models.Friendship{}
	for rows.Next() {
		var f models.Friendship
		if err := rows.Scan(&f.ID, &f.RequesterID, &f.RecipientID, &f.Status, &f.CreatedAt, &f.ResolvedAt, &f.FriendID, &f.FriendEmail); err != nil {
			return nil, err
		}
		requests = append(requests, f)
	}
	return requests, rows.Err()
}

// ListAccepted returns accepted friendships for userID, enriched with the
// OTHER participant's id/email regardless of which side sent the request.
func (s *FriendshipStore) ListAccepted(userID int64) ([]models.Friendship, error) {
	rows, err := s.db.Query(`
		SELECT f.id, f.requester_id, f.recipient_id, f.status, f.created_at, f.resolved_at,
		       u.id, u.email
		FROM friendships f
		JOIN users u ON u.id = CASE WHEN f.requester_id = ? THEN f.recipient_id ELSE f.requester_id END
		WHERE f.status = 'accepted' AND (f.requester_id = ? OR f.recipient_id = ?)
		ORDER BY f.resolved_at DESC
	`, userID, userID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	friends := []models.Friendship{}
	for rows.Next() {
		var f models.Friendship
		if err := rows.Scan(&f.ID, &f.RequesterID, &f.RecipientID, &f.Status, &f.CreatedAt, &f.ResolvedAt, &f.FriendID, &f.FriendEmail); err != nil {
			return nil, err
		}
		friends = append(friends, f)
	}
	return friends, rows.Err()
}

func (s *FriendshipStore) Resolve(id int64, status string) (models.Friendship, error) {
	if _, err := s.db.Exec(`
		UPDATE friendships SET status = ?, resolved_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now') WHERE id = ?
	`, status, id); err != nil {
		return models.Friendship{}, err
	}
	return s.GetByID(id)
}
