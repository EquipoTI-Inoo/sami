package stores

import (
	"database/sql"
	"encoding/json"
	"errors"

	"coachia-backend/models"
)

type FoodStore struct {
	db *sql.DB
}

func NewFoodStore(db *sql.DB) *FoodStore {
	return &FoodStore{db: db}
}

func (s *FoodStore) ListAll() ([]models.PeruvianFood, error) {
	rows, err := s.db.Query(`SELECT id, name, category, serving_desc, serving_grams, calories, protein_g, carbs_g, fat_g, tags FROM peruvian_foods`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var foods []models.PeruvianFood
	for rows.Next() {
		var f models.PeruvianFood
		var tags string
		if err := rows.Scan(&f.ID, &f.Name, &f.Category, &f.ServingDesc, &f.ServingGrams, &f.Calories, &f.ProteinG, &f.CarbsG, &f.FatG, &tags); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(tags), &f.Tags); err != nil {
			return nil, err
		}
		foods = append(foods, f)
	}
	return foods, rows.Err()
}

func (s *FoodStore) GetByID(id int64) (models.PeruvianFood, error) {
	var f models.PeruvianFood
	var tags string
	err := s.db.QueryRow(`SELECT id, name, category, serving_desc, serving_grams, calories, protein_g, carbs_g, fat_g, tags FROM peruvian_foods WHERE id = ?`, id).
		Scan(&f.ID, &f.Name, &f.Category, &f.ServingDesc, &f.ServingGrams, &f.Calories, &f.ProteinG, &f.CarbsG, &f.FatG, &tags)
	if errors.Is(err, sql.ErrNoRows) {
		return models.PeruvianFood{}, ErrNotFound
	}
	if err != nil {
		return models.PeruvianFood{}, err
	}
	err = json.Unmarshal([]byte(tags), &f.Tags)
	return f, err
}
