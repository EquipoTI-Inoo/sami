package stores

import (
	"database/sql"
	"errors"

	"coachia-backend/models"
)

type SharedGoalStore struct {
	db *sql.DB
}

func NewSharedGoalStore(db *sql.DB) *SharedGoalStore {
	return &SharedGoalStore{db: db}
}

func (s *SharedGoalStore) Create(friendshipID, createdBy int64, req models.CreateSharedGoalRequest) (models.SharedGoal, error) {
	res, err := s.db.Exec(`
		INSERT INTO shared_goals (friendship_id, created_by, goal_type, description, target_value, target_date)
		VALUES (?, ?, ?, ?, ?, ?)
	`, friendshipID, createdBy, req.GoalType, req.Description, req.TargetValue, req.TargetDate)
	if err != nil {
		return models.SharedGoal{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return models.SharedGoal{}, err
	}
	return s.GetByID(id)
}

func (s *SharedGoalStore) GetByID(id int64) (models.SharedGoal, error) {
	var g models.SharedGoal
	err := s.db.QueryRow(`
		SELECT id, friendship_id, created_by, goal_type, description, target_value, target_date, status, created_at, completed_at
		FROM shared_goals WHERE id = ?
	`, id).Scan(&g.ID, &g.FriendshipID, &g.CreatedBy, &g.GoalType, &g.Description, &g.TargetValue, &g.TargetDate, &g.Status, &g.CreatedAt, &g.CompletedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return models.SharedGoal{}, ErrNotFound
	}
	return g, err
}

func (s *SharedGoalStore) ListForFriendship(friendshipID int64) ([]models.SharedGoal, error) {
	rows, err := s.db.Query(`
		SELECT id, friendship_id, created_by, goal_type, description, target_value, target_date, status, created_at, completed_at
		FROM shared_goals WHERE friendship_id = ? ORDER BY created_at DESC
	`, friendshipID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	goals := []models.SharedGoal{}
	for rows.Next() {
		var g models.SharedGoal
		if err := rows.Scan(&g.ID, &g.FriendshipID, &g.CreatedBy, &g.GoalType, &g.Description, &g.TargetValue, &g.TargetDate, &g.Status, &g.CreatedAt, &g.CompletedAt); err != nil {
			return nil, err
		}
		goals = append(goals, g)
	}
	return goals, rows.Err()
}

func (s *SharedGoalStore) Complete(id int64) (models.SharedGoal, error) {
	if _, err := s.db.Exec(`
		UPDATE shared_goals SET status = 'completed', completed_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now') WHERE id = ?
	`, id); err != nil {
		return models.SharedGoal{}, err
	}
	return s.GetByID(id)
}
