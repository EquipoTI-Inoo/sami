package stores

import (
	"database/sql"
	"errors"

	"coachia-backend/models"
)

type ExerciseStore struct {
	db *sql.DB
}

func NewExerciseStore(db *sql.DB) *ExerciseStore {
	return &ExerciseStore{db: db}
}

// ListAll returns the exercise catalog. A location of "gym" or "home" also
// includes "both"; an empty location returns the whole catalog — this is
// the filter the PRD calls out as new versus Lyftr ("solo tengo esto en casa").
func (s *ExerciseStore) ListAll(location string) ([]models.Exercise, error) {
	query := `SELECT id, name, muscle_group, equipment, location FROM exercises`
	var args []any
	if location != "" {
		query += ` WHERE location = ? OR location = 'both'`
		args = append(args, location)
	}
	query += ` ORDER BY muscle_group, name`

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	exercises := []models.Exercise{}
	for rows.Next() {
		var e models.Exercise
		var equipment string
		if err := rows.Scan(&e.ID, &e.Name, &e.MuscleGroup, &equipment, &e.Location); err != nil {
			return nil, err
		}
		if e.Equipment, err = unmarshalStrings(equipment); err != nil {
			return nil, err
		}
		exercises = append(exercises, e)
	}
	return exercises, rows.Err()
}

func (s *ExerciseStore) GetByID(id int64) (models.Exercise, error) {
	var e models.Exercise
	var equipment string
	err := s.db.QueryRow(`SELECT id, name, muscle_group, equipment, location FROM exercises WHERE id = ?`, id).
		Scan(&e.ID, &e.Name, &e.MuscleGroup, &equipment, &e.Location)
	if errors.Is(err, sql.ErrNoRows) {
		return models.Exercise{}, ErrNotFound
	}
	if err != nil {
		return models.Exercise{}, err
	}
	e.Equipment, err = unmarshalStrings(equipment)
	return e, err
}
