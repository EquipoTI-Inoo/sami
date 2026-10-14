package stores

import (
	"database/sql"
	"errors"

	"coachia-backend/models"
	"coachia-backend/training"
)

type ProgramStore struct {
	db *sql.DB
}

func NewProgramStore(db *sql.DB) *ProgramStore {
	return &ProgramStore{db: db}
}

// Create persists a generated program — program + days + exercises — in one
// transaction, so a partial write (e.g. the process dying mid-insert) can
// never leave a program with some days missing their exercises.
func (s *ProgramStore) Create(userID int64, goal, location string, days []training.DayPlan) (models.Program, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return models.Program{}, err
	}
	defer tx.Rollback()

	res, err := tx.Exec(`INSERT INTO programs (user_id, goal, location) VALUES (?, ?, ?)`, userID, goal, location)
	if err != nil {
		return models.Program{}, err
	}
	programID, err := res.LastInsertId()
	if err != nil {
		return models.Program{}, err
	}

	for dayIndex, day := range days {
		dayRes, err := tx.Exec(`INSERT INTO program_days (program_id, day_index, name) VALUES (?, ?, ?)`, programID, dayIndex, day.Name)
		if err != nil {
			return models.Program{}, err
		}
		dayID, err := dayRes.LastInsertId()
		if err != nil {
			return models.Program{}, err
		}

		for orderIndex, pick := range day.Exercises {
			if _, err := tx.Exec(`
				INSERT INTO program_exercises (program_day_id, exercise_id, order_index, target_sets, target_reps_min, target_reps_max)
				VALUES (?, ?, ?, ?, ?, ?)
			`, dayID, pick.ExerciseID, orderIndex, pick.TargetSets, pick.TargetRepsMin, pick.TargetRepsMax); err != nil {
				return models.Program{}, err
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return models.Program{}, err
	}

	return s.GetByID(programID)
}

func (s *ProgramStore) GetLatest(userID int64) (models.Program, error) {
	var programID int64
	err := s.db.QueryRow(`SELECT id FROM programs WHERE user_id = ? ORDER BY created_at DESC LIMIT 1`, userID).Scan(&programID)
	if errors.Is(err, sql.ErrNoRows) {
		return models.Program{}, ErrNotFound
	}
	if err != nil {
		return models.Program{}, err
	}
	return s.GetByID(programID)
}

func (s *ProgramStore) GetByID(programID int64) (models.Program, error) {
	var p models.Program
	err := s.db.QueryRow(`SELECT id, user_id, goal, location, created_at FROM programs WHERE id = ?`, programID).
		Scan(&p.ID, &p.UserID, &p.Goal, &p.Location, &p.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return models.Program{}, ErrNotFound
	}
	if err != nil {
		return models.Program{}, err
	}

	dayRows, err := s.db.Query(`SELECT id, day_index, name FROM program_days WHERE program_id = ? ORDER BY day_index`, programID)
	if err != nil {
		return models.Program{}, err
	}
	defer dayRows.Close()

	p.Days = []models.ProgramDay{}
	for dayRows.Next() {
		var d models.ProgramDay
		if err := dayRows.Scan(&d.ID, &d.DayIndex, &d.Name); err != nil {
			return models.Program{}, err
		}
		d.Exercises = []models.ProgramExercise{}
		p.Days = append(p.Days, d)
	}
	if err := dayRows.Err(); err != nil {
		return models.Program{}, err
	}

	exerciseRows, err := s.db.Query(`
		SELECT pe.id, pe.program_day_id, pe.exercise_id, pe.order_index, pe.target_sets, pe.target_reps_min, pe.target_reps_max,
		       e.id, e.name, e.muscle_group, e.equipment, e.location
		FROM program_exercises pe
		JOIN program_days pd ON pd.id = pe.program_day_id
		JOIN exercises e ON e.id = pe.exercise_id
		WHERE pd.program_id = ?
		ORDER BY pe.program_day_id, pe.order_index
	`, programID)
	if err != nil {
		return models.Program{}, err
	}
	defer exerciseRows.Close()

	exercisesByDay := map[int64][]models.ProgramExercise{}
	for exerciseRows.Next() {
		var pe models.ProgramExercise
		var dayID int64
		var e models.Exercise
		var equipment string
		if err := exerciseRows.Scan(&pe.ID, &dayID, &pe.ExerciseID, &pe.OrderIndex, &pe.TargetSets, &pe.TargetRepsMin, &pe.TargetRepsMax,
			&e.ID, &e.Name, &e.MuscleGroup, &equipment, &e.Location); err != nil {
			return models.Program{}, err
		}
		if e.Equipment, err = unmarshalStrings(equipment); err != nil {
			return models.Program{}, err
		}
		pe.Exercise = &e
		exercisesByDay[dayID] = append(exercisesByDay[dayID], pe)
	}
	if err := exerciseRows.Err(); err != nil {
		return models.Program{}, err
	}

	for i := range p.Days {
		if exercises, ok := exercisesByDay[p.Days[i].ID]; ok {
			p.Days[i].Exercises = exercises
		}
	}

	return p, nil
}
