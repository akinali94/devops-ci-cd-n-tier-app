package repository

import (
	"database/sql"
	"fmt"

	"task-manager/auth-service/internal/model"
)

type UserRepository interface {
	Create(email, passwordHash string) (*model.User, error)
	GetByEmail(email string) (*model.User, error)
}

type postgresUserRepo struct {
	db *sql.DB
}

func NewPostgresUserRepo(db *sql.DB) UserRepository {
	return &postgresUserRepo{db: db}
}

func (r *postgresUserRepo) Create(email, passwordHash string) (*model.User, error) {
	const q = `
		INSERT INTO users (email, password_hash)
		VALUES ($1, $2)
		RETURNING id, email, created_at`

	var u model.User
	err := r.db.QueryRow(q, email, passwordHash).Scan(&u.ID, &u.Email, &u.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}
	return &u, nil
}

func (r *postgresUserRepo) GetByEmail(email string) (*model.User, error) {
	const q = `
		SELECT id, email, password_hash, created_at
		FROM users
		WHERE email = $1`

	var u model.User
	err := r.db.QueryRow(q, email).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get user by email: %w", err)
	}
	return &u, nil
}
