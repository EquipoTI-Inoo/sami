package stores

import (
	"database/sql"
	"errors"
	"strings"

	"coachia-backend/models"
)

var ErrNotFound = errors.New("not found")
var ErrEmailTaken = errors.New("email already registered")

type UserStore struct {
	db *sql.DB
}

func NewUserStore(db *sql.DB) *UserStore {
	return &UserStore{db: db}
}

func (s *UserStore) Create(email, passwordHash string) (models.User, error) {
	res, err := s.db.Exec(`INSERT INTO users (email, password_hash) VALUES (?, ?)`, email, passwordHash)
	if err != nil {
		if isUniqueConstraintErr(err) {
			return models.User{}, ErrEmailTaken
		}
		return models.User{}, err
	}

	id, err := res.LastInsertId()
	if err != nil {
		return models.User{}, err
	}

	return s.FindByID(id)
}

func (s *UserStore) FindByID(id int64) (models.User, error) {
	var u models.User
	err := s.db.QueryRow(`SELECT id, email, created_at FROM users WHERE id = ?`, id).
		Scan(&u.ID, &u.Email, &u.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return models.User{}, ErrNotFound
	}
	return u, err
}

func (s *UserStore) FindByEmail(email string) (models.User, error) {
	var u models.User
	err := s.db.QueryRow(`SELECT id, email, created_at FROM users WHERE email = ?`, email).
		Scan(&u.ID, &u.Email, &u.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return models.User{}, ErrNotFound
	}
	return u, err
}

func (s *UserStore) FindByEmailWithPassword(email string) (models.User, string, error) {
	var u models.User
	var hash string
	err := s.db.QueryRow(`SELECT id, email, created_at, password_hash FROM users WHERE email = ?`, email).
		Scan(&u.ID, &u.Email, &u.CreatedAt, &hash)
	if errors.Is(err, sql.ErrNoRows) {
		return models.User{}, "", ErrNotFound
	}
	return u, hash, err
}

func isUniqueConstraintErr(err error) bool {
	return err != nil && strings.Contains(strings.ToLower(err.Error()), "unique constraint")
}
