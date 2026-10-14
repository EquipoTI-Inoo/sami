package stores

import (
	"database/sql"
	"errors"

	"coachia-backend/models"
)

type WorkoutStore struct {
	db *sql.DB
}

func NewWorkoutStore(db *sql.DB) *WorkoutStore {
	return &WorkoutStore{db: db}
}

// CountCompletedSince counts completed workouts started on or after `since`
// (a "2006-01-02T..." timestamp) — used for both the friend-progress view
// and shared_goals of type workouts_per_week.
func (s *WorkoutStore) CountCompletedSince(userID int64, since string) (int, error) {
	var count int
	err := s.db.QueryRow(`
		SELECT COUNT(1) FROM workouts WHERE user_id = ? AND completed_at IS NOT NULL AND started_at >= ?
	`, userID, since).Scan(&count)
	return count, err
}

func (s *WorkoutStore) CountCompleted(userID int64) (int, error) {
	var count int
	err := s.db.QueryRow(`
		SELECT COUNT(1) FROM workouts WHERE user_id = ? AND completed_at IS NOT NULL
	`, userID).Scan(&count)
	return count, err
}

func (s *WorkoutStore) Create(userID int64) (models.Workout, error) {
	res, err := s.db.Exec(`INSERT INTO workouts (user_id) VALUES (?)`, userID)
	if err != nil {
		return models.Workout{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return models.Workout{}, err
	}
	return s.GetByID(id)
}

// GetWorkoutOwner is a lightweight ownership check used before any write to
// a workout, so one user's request can never touch another's session.
func (s *WorkoutStore) GetWorkoutOwner(workoutID int64) (int64, error) {
	var userID int64
	err := s.db.QueryRow(`SELECT user_id FROM workouts WHERE id = ?`, workoutID).Scan(&userID)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, ErrNotFound
	}
	return userID, err
}

// GetWorkoutExerciseOwner resolves straight through to the owning user, for
// authorizing a set log without a round trip through the workout first.
func (s *WorkoutStore) GetWorkoutExerciseOwner(workoutExerciseID int64) (userID int64, workoutID int64, err error) {
	err = s.db.QueryRow(`
		SELECT w.user_id, w.id
		FROM workout_exercises we
		JOIN workouts w ON w.id = we.workout_id
		WHERE we.id = ?
	`, workoutExerciseID).Scan(&userID, &workoutID)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, 0, ErrNotFound
	}
	return userID, workoutID, err
}

func (s *WorkoutStore) AddExercise(workoutID, exerciseID int64) (models.WorkoutExercise, error) {
	var nextOrder int
	if err := s.db.QueryRow(`SELECT COUNT(1) FROM workout_exercises WHERE workout_id = ?`, workoutID).Scan(&nextOrder); err != nil {
		return models.WorkoutExercise{}, err
	}

	res, err := s.db.Exec(`
		INSERT INTO workout_exercises (workout_id, exercise_id, order_index) VALUES (?, ?, ?)
	`, workoutID, exerciseID, nextOrder)
	if err != nil {
		return models.WorkoutExercise{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return models.WorkoutExercise{}, err
	}
	return s.getWorkoutExerciseByID(id)
}

func (s *WorkoutStore) getWorkoutExerciseByID(id int64) (models.WorkoutExercise, error) {
	var we models.WorkoutExercise
	err := s.db.QueryRow(`SELECT id, workout_id, exercise_id, order_index FROM workout_exercises WHERE id = ?`, id).
		Scan(&we.ID, &we.WorkoutID, &we.ExerciseID, &we.OrderIndex)
	if errors.Is(err, sql.ErrNoRows) {
		return models.WorkoutExercise{}, ErrNotFound
	}
	return we, err
}

func (s *WorkoutStore) LogSet(workoutExerciseID int64, reps int, weightKg *float64) (models.WorkoutSet, error) {
	var nextSetNumber int
	if err := s.db.QueryRow(`SELECT COUNT(1) FROM workout_sets WHERE workout_exercise_id = ?`, workoutExerciseID).Scan(&nextSetNumber); err != nil {
		return models.WorkoutSet{}, err
	}
	nextSetNumber++

	res, err := s.db.Exec(`
		INSERT INTO workout_sets (workout_exercise_id, set_number, reps, weight_kg) VALUES (?, ?, ?, ?)
	`, workoutExerciseID, nextSetNumber, reps, weightKg)
	if err != nil {
		return models.WorkoutSet{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return models.WorkoutSet{}, err
	}

	var set models.WorkoutSet
	err = s.db.QueryRow(`
		SELECT id, workout_exercise_id, set_number, reps, weight_kg, completed_at FROM workout_sets WHERE id = ?
	`, id).Scan(&set.ID, &set.WorkoutExerciseID, &set.SetNumber, &set.Reps, &set.WeightKg, &set.CompletedAt)
	return set, err
}

func (s *WorkoutStore) Complete(workoutID int64) (models.Workout, error) {
	if _, err := s.db.Exec(`
		UPDATE workouts SET completed_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now') WHERE id = ?
	`, workoutID); err != nil {
		return models.Workout{}, err
	}
	return s.GetByID(workoutID)
}

// GetByID loads a workout with its exercises (including catalog details)
// and sets fully nested — the shape the Gym Mode screen and workout detail
// view both need in one call.
func (s *WorkoutStore) GetByID(workoutID int64) (models.Workout, error) {
	var w models.Workout
	err := s.db.QueryRow(`
		SELECT id, user_id, started_at, completed_at, notes FROM workouts WHERE id = ?
	`, workoutID).Scan(&w.ID, &w.UserID, &w.StartedAt, &w.CompletedAt, &w.Notes)
	if errors.Is(err, sql.ErrNoRows) {
		return models.Workout{}, ErrNotFound
	}
	if err != nil {
		return models.Workout{}, err
	}

	rows, err := s.db.Query(`
		SELECT we.id, we.workout_id, we.exercise_id, we.order_index,
		       e.id, e.name, e.muscle_group, e.equipment, e.location
		FROM workout_exercises we
		JOIN exercises e ON e.id = we.exercise_id
		WHERE we.workout_id = ?
		ORDER BY we.order_index
	`, workoutID)
	if err != nil {
		return models.Workout{}, err
	}
	defer rows.Close()

	w.Exercises = []models.WorkoutExercise{}
	for rows.Next() {
		var we models.WorkoutExercise
		var e models.Exercise
		var equipment string
		if err := rows.Scan(&we.ID, &we.WorkoutID, &we.ExerciseID, &we.OrderIndex, &e.ID, &e.Name, &e.MuscleGroup, &equipment, &e.Location); err != nil {
			return models.Workout{}, err
		}
		if e.Equipment, err = unmarshalStrings(equipment); err != nil {
			return models.Workout{}, err
		}
		we.Exercise = &e
		we.Sets = []models.WorkoutSet{}
		w.Exercises = append(w.Exercises, we)
	}
	if err := rows.Err(); err != nil {
		return models.Workout{}, err
	}

	setRows, err := s.db.Query(`
		SELECT ws.id, ws.workout_exercise_id, ws.set_number, ws.reps, ws.weight_kg, ws.completed_at
		FROM workout_sets ws
		JOIN workout_exercises we ON we.id = ws.workout_exercise_id
		WHERE we.workout_id = ?
		ORDER BY ws.workout_exercise_id, ws.set_number
	`, workoutID)
	if err != nil {
		return models.Workout{}, err
	}
	defer setRows.Close()

	setsByExercise := map[int64][]models.WorkoutSet{}
	for setRows.Next() {
		var set models.WorkoutSet
		if err := setRows.Scan(&set.ID, &set.WorkoutExerciseID, &set.SetNumber, &set.Reps, &set.WeightKg, &set.CompletedAt); err != nil {
			return models.Workout{}, err
		}
		setsByExercise[set.WorkoutExerciseID] = append(setsByExercise[set.WorkoutExerciseID], set)
	}
	if err := setRows.Err(); err != nil {
		return models.Workout{}, err
	}

	for i := range w.Exercises {
		if sets, ok := setsByExercise[w.Exercises[i].ID]; ok {
			w.Exercises[i].Sets = sets
		}
	}
	w.ExerciseCount = len(w.Exercises)

	return w, nil
}

// ListForUser returns workout summaries (no nested exercises/sets, just a
// count) newest first, for a history list.
func (s *WorkoutStore) ListForUser(userID int64) ([]models.Workout, error) {
	rows, err := s.db.Query(`
		SELECT w.id, w.user_id, w.started_at, w.completed_at, w.notes,
		       (SELECT COUNT(1) FROM workout_exercises we WHERE we.workout_id = w.id)
		FROM workouts w WHERE w.user_id = ? ORDER BY w.started_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	workouts := []models.Workout{}
	for rows.Next() {
		var w models.Workout
		if err := rows.Scan(&w.ID, &w.UserID, &w.StartedAt, &w.CompletedAt, &w.Notes, &w.ExerciseCount); err != nil {
			return nil, err
		}
		workouts = append(workouts, w)
	}
	return workouts, rows.Err()
}
