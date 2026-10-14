package stores

import (
	"database/sql"
	"encoding/json"
	"errors"
	"math"
	"time"

	"coachia-backend/models"
)

type FoodLogStore struct {
	db *sql.DB
}

func NewFoodLogStore(db *sql.DB) *FoodLogStore {
	return &FoodLogStore{db: db}
}

// Create logs one entry. loggedOn defaults to today (UTC) when empty — the
// common case is logging what you're eating right now.
func (s *FoodLogStore) Create(userID int64, req models.CreateFoodLogEntryRequest) (models.FoodLogEntry, error) {
	loggedOn := req.LoggedOn
	if loggedOn == "" {
		loggedOn = time.Now().UTC().Format("2006-01-02")
	}

	res, err := s.db.Exec(`
		INSERT INTO food_log_entries (user_id, food_id, meal_type, servings, logged_on)
		VALUES (?, ?, ?, ?, ?)
	`, userID, req.FoodID, req.MealType, req.Servings, loggedOn)
	if err != nil {
		return models.FoodLogEntry{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return models.FoodLogEntry{}, err
	}
	return s.GetByID(id)
}

func (s *FoodLogStore) GetByID(id int64) (models.FoodLogEntry, error) {
	entry, err := s.scanEntry(s.db.QueryRow(`
		SELECT fl.id, fl.user_id, fl.food_id, fl.meal_type, fl.servings, fl.logged_on, fl.created_at,
		       f.id, f.name, f.category, f.serving_desc, f.serving_grams, f.calories, f.protein_g, f.carbs_g, f.fat_g, f.tags
		FROM food_log_entries fl
		JOIN peruvian_foods f ON f.id = fl.food_id
		WHERE fl.id = ?
	`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return models.FoodLogEntry{}, ErrNotFound
	}
	return entry, err
}

// ListForDay returns every entry logged on that date, macros recomputed
// fresh from the current peruvian_foods row each time — never stored, so a
// correction to the catalog is reflected immediately rather than needing a
// backfill.
func (s *FoodLogStore) ListForDay(userID int64, loggedOn string) ([]models.FoodLogEntry, error) {
	rows, err := s.db.Query(`
		SELECT fl.id, fl.user_id, fl.food_id, fl.meal_type, fl.servings, fl.logged_on, fl.created_at,
		       f.id, f.name, f.category, f.serving_desc, f.serving_grams, f.calories, f.protein_g, f.carbs_g, f.fat_g, f.tags
		FROM food_log_entries fl
		JOIN peruvian_foods f ON f.id = fl.food_id
		WHERE fl.user_id = ? AND fl.logged_on = ?
		ORDER BY fl.created_at
	`, userID, loggedOn)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	entries := []models.FoodLogEntry{}
	for rows.Next() {
		entry, err := s.scanEntry(rows)
		if err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
	return entries, rows.Err()
}

// GetOwner is a lightweight ownership check for delete authorization.
func (s *FoodLogStore) GetOwner(id int64) (int64, error) {
	var userID int64
	err := s.db.QueryRow(`SELECT user_id FROM food_log_entries WHERE id = ?`, id).Scan(&userID)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, ErrNotFound
	}
	return userID, err
}

func (s *FoodLogStore) Delete(id int64) error {
	_, err := s.db.Exec(`DELETE FROM food_log_entries WHERE id = ?`, id)
	return err
}

type rowScanner interface {
	Scan(dest ...any) error
}

func (s *FoodLogStore) scanEntry(row rowScanner) (models.FoodLogEntry, error) {
	var e models.FoodLogEntry
	var f models.PeruvianFood
	var tags string
	if err := row.Scan(
		&e.ID, &e.UserID, &e.FoodID, &e.MealType, &e.Servings, &e.LoggedOn, &e.CreatedAt,
		&f.ID, &f.Name, &f.Category, &f.ServingDesc, &f.ServingGrams, &f.Calories, &f.ProteinG, &f.CarbsG, &f.FatG, &tags,
	); err != nil {
		return models.FoodLogEntry{}, err
	}
	if err := json.Unmarshal([]byte(tags), &f.Tags); err != nil {
		return models.FoodLogEntry{}, err
	}

	e.Food = &f
	e.TotalGrams = int(math.Round(f.ServingGrams * e.Servings))
	e.Calories = int(math.Round(f.Calories * e.Servings))
	e.ProteinG = int(math.Round(f.ProteinG * e.Servings))
	e.CarbsG = int(math.Round(f.CarbsG * e.Servings))
	e.FatG = int(math.Round(f.FatG * e.Servings))

	return e, nil
}
