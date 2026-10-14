package stores

import (
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"coachia-backend/models"
)

type ProfileStore struct {
	db *sql.DB
}

func NewProfileStore(db *sql.DB) *ProfileStore {
	return &ProfileStore{db: db}
}

func (s *ProfileStore) UpsertProfile(userID int64, req models.OnboardingRequest) error {
	equipment, err := marshalStrings(req.Equipment)
	if err != nil {
		return err
	}
	likes, err := marshalStrings(req.Likes)
	if err != nil {
		return err
	}
	dislikes, err := marshalStrings(req.Dislikes)
	if err != nil {
		return err
	}
	restrictions, err := marshalStrings(req.Restrictions)
	if err != nil {
		return err
	}

	_, err = s.db.Exec(`
		INSERT INTO user_profile
			(user_id, age, sex, height_cm, body_fat_pct, activity_level, goal, training_location, equipment, likes, dislikes, restrictions, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
		ON CONFLICT(user_id) DO UPDATE SET
			age = excluded.age,
			sex = excluded.sex,
			height_cm = excluded.height_cm,
			body_fat_pct = excluded.body_fat_pct,
			activity_level = excluded.activity_level,
			goal = excluded.goal,
			training_location = excluded.training_location,
			equipment = excluded.equipment,
			likes = excluded.likes,
			dislikes = excluded.dislikes,
			restrictions = excluded.restrictions,
			updated_at = excluded.updated_at
	`, userID, req.Age, req.Sex, req.HeightCm, req.BodyFatPct, req.ActivityLevel, req.Goal, req.TrainingLocation, equipment, likes, dislikes, restrictions)

	return err
}

func (s *ProfileStore) GetProfile(userID int64) (models.UserProfile, error) {
	var p models.UserProfile
	var equipment, likes, dislikes, restrictions string

	err := s.db.QueryRow(`
		SELECT user_id, age, sex, height_cm, body_fat_pct, activity_level, goal, training_location, equipment, likes, dislikes, restrictions, created_at, updated_at
		FROM user_profile WHERE user_id = ?
	`, userID).Scan(&p.UserID, &p.Age, &p.Sex, &p.HeightCm, &p.BodyFatPct, &p.ActivityLevel, &p.Goal, &p.TrainingLocation, &equipment, &likes, &dislikes, &restrictions, &p.CreatedAt, &p.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return models.UserProfile{}, ErrNotFound
	}
	if err != nil {
		return models.UserProfile{}, err
	}

	if p.Equipment, err = unmarshalStrings(equipment); err != nil {
		return models.UserProfile{}, err
	}
	if p.Likes, err = unmarshalStrings(likes); err != nil {
		return models.UserProfile{}, err
	}
	if p.Dislikes, err = unmarshalStrings(dislikes); err != nil {
		return models.UserProfile{}, err
	}
	if p.Restrictions, err = unmarshalStrings(restrictions); err != nil {
		return models.UserProfile{}, err
	}

	return p, nil
}

type BodyMeasurementInput struct {
	WeightKg   float64
	WaistCm    *float64
	HipCm      *float64
	ArmCm      *float64
	BodyFatPct *float64
	Note       *string
}

func (s *ProfileStore) InsertBodyMeasurement(userID int64, in BodyMeasurementInput) (models.BodyMeasurement, error) {
	today := time.Now().UTC().Format("2006-01-02")
	res, err := s.db.Exec(`
		INSERT INTO body_measurements (user_id, logged_on, weight_kg, waist_cm, hip_cm, arm_cm, body_fat_pct, note)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, userID, today, in.WeightKg, in.WaistCm, in.HipCm, in.ArmCm, in.BodyFatPct, in.Note)
	if err != nil {
		return models.BodyMeasurement{}, err
	}

	id, err := res.LastInsertId()
	if err != nil {
		return models.BodyMeasurement{}, err
	}

	return s.GetBodyMeasurementByID(id)
}

func (s *ProfileStore) GetBodyMeasurementByID(id int64) (models.BodyMeasurement, error) {
	var m models.BodyMeasurement
	err := s.db.QueryRow(`
		SELECT id, user_id, logged_on, weight_kg, waist_cm, hip_cm, arm_cm, body_fat_pct, note, created_at
		FROM body_measurements WHERE id = ?
	`, id).Scan(&m.ID, &m.UserID, &m.LoggedOn, &m.WeightKg, &m.WaistCm, &m.HipCm, &m.ArmCm, &m.BodyFatPct, &m.Note, &m.CreatedAt)
	return m, err
}

func (s *ProfileStore) GetLatestBodyMeasurement(userID int64) (models.BodyMeasurement, error) {
	var m models.BodyMeasurement
	err := s.db.QueryRow(`
		SELECT id, user_id, logged_on, weight_kg, waist_cm, hip_cm, arm_cm, body_fat_pct, note, created_at
		FROM body_measurements WHERE user_id = ? ORDER BY logged_on DESC, id DESC LIMIT 1
	`, userID).Scan(&m.ID, &m.UserID, &m.LoggedOn, &m.WeightKg, &m.WaistCm, &m.HipCm, &m.ArmCm, &m.BodyFatPct, &m.Note, &m.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return models.BodyMeasurement{}, ErrNotFound
	}
	return m, err
}

// ListRecentBodyMeasurements returns the most recent check-ins, newest
// first — used by the friend-progress view to show a short weight trend.
func (s *ProfileStore) ListRecentBodyMeasurements(userID int64, limit int) ([]models.BodyMeasurement, error) {
	rows, err := s.db.Query(`
		SELECT id, user_id, logged_on, weight_kg, waist_cm, hip_cm, arm_cm, body_fat_pct, note, created_at
		FROM body_measurements WHERE user_id = ? ORDER BY logged_on DESC, id DESC LIMIT ?
	`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	measurements := []models.BodyMeasurement{}
	for rows.Next() {
		var m models.BodyMeasurement
		if err := rows.Scan(&m.ID, &m.UserID, &m.LoggedOn, &m.WeightKg, &m.WaistCm, &m.HipCm, &m.ArmCm, &m.BodyFatPct, &m.Note, &m.CreatedAt); err != nil {
			return nil, err
		}
		measurements = append(measurements, m)
	}
	return measurements, rows.Err()
}

func (s *ProfileStore) InsertNutritionTargets(userID int64, t NutritionTargetsInput) (models.NutritionTargets, error) {
	res, err := s.db.Exec(`
		INSERT INTO nutrition_targets (user_id, calories, protein_g, carbs_g, fat_g, updated_by, reason)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, userID, t.Calories, t.ProteinG, t.CarbsG, t.FatG, t.UpdatedBy, t.Reason)
	if err != nil {
		return models.NutritionTargets{}, err
	}

	id, err := res.LastInsertId()
	if err != nil {
		return models.NutritionTargets{}, err
	}

	return s.GetNutritionTargetsByID(id)
}

func (s *ProfileStore) GetNutritionTargetsByID(id int64) (models.NutritionTargets, error) {
	var t models.NutritionTargets
	err := s.db.QueryRow(`
		SELECT id, user_id, calories, protein_g, carbs_g, fat_g, updated_by, reason, created_at
		FROM nutrition_targets WHERE id = ?
	`, id).Scan(&t.ID, &t.UserID, &t.Calories, &t.ProteinG, &t.CarbsG, &t.FatG, &t.UpdatedBy, &t.Reason, &t.CreatedAt)
	return t, err
}

func (s *ProfileStore) GetLatestNutritionTargets(userID int64) (models.NutritionTargets, error) {
	var t models.NutritionTargets
	err := s.db.QueryRow(`
		SELECT id, user_id, calories, protein_g, carbs_g, fat_g, updated_by, reason, created_at
		FROM nutrition_targets WHERE user_id = ? ORDER BY created_at DESC LIMIT 1
	`, userID).Scan(&t.ID, &t.UserID, &t.Calories, &t.ProteinG, &t.CarbsG, &t.FatG, &t.UpdatedBy, &t.Reason, &t.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return models.NutritionTargets{}, ErrNotFound
	}
	return t, err
}

func (s *ProfileStore) GetNutritionTargetsHistory(userID int64) ([]models.NutritionTargets, error) {
	rows, err := s.db.Query(`
		SELECT id, user_id, calories, protein_g, carbs_g, fat_g, updated_by, reason, created_at
		FROM nutrition_targets WHERE user_id = ? ORDER BY created_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	history := []models.NutritionTargets{}
	for rows.Next() {
		var t models.NutritionTargets
		if err := rows.Scan(&t.ID, &t.UserID, &t.Calories, &t.ProteinG, &t.CarbsG, &t.FatG, &t.UpdatedBy, &t.Reason, &t.CreatedAt); err != nil {
			return nil, err
		}
		history = append(history, t)
	}
	return history, rows.Err()
}

type NutritionTargetsInput struct {
	Calories  int
	ProteinG  int
	CarbsG    int
	FatG      int
	UpdatedBy string
	Reason    string
}

func marshalStrings(v []string) (string, error) {
	if v == nil {
		v = []string{}
	}
	b, err := json.Marshal(v)
	return string(b), err
}

func unmarshalStrings(s string) ([]string, error) {
	var v []string
	if s == "" {
		return []string{}, nil
	}
	err := json.Unmarshal([]byte(s), &v)
	return v, err
}
