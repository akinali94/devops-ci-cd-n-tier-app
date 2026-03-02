package repository

import (
	"database/sql"
	"fmt"

	"task-manager/api-service/internal/model"
)

type TaskRepository interface {
	GetAllByUser(userID string) ([]model.Task, error)
	GetByID(id, userID string) (*model.Task, error)
	Create(userID string, req model.CreateTaskRequest) (*model.Task, error)
	Update(id, userID string, req model.UpdateTaskRequest) (*model.Task, error)
	Delete(id, userID string) error
}

type postgresTaskRepo struct {
	db *sql.DB
}

func NewPostgresTaskRepo(db *sql.DB) TaskRepository {
	return &postgresTaskRepo{db: db}
}

func (r *postgresTaskRepo) GetAllByUser(userID string) ([]model.Task, error) {
	const q = `
		SELECT id, user_id, title, description, status, created_at, updated_at
		FROM tasks
		WHERE user_id = $1
		ORDER BY created_at DESC`

	rows, err := r.db.Query(q, userID)
	if err != nil {
		return nil, fmt.Errorf("query tasks: %w", err)
	}
	defer rows.Close()

	var tasks []model.Task
	for rows.Next() {
		var t model.Task
		if err := rows.Scan(&t.ID, &t.UserID, &t.Title, &t.Description, &t.Status, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan task: %w", err)
		}
		tasks = append(tasks, t)
	}
	return tasks, rows.Err()
}

func (r *postgresTaskRepo) GetByID(id, userID string) (*model.Task, error) {
	const q = `
		SELECT id, user_id, title, description, status, created_at, updated_at
		FROM tasks
		WHERE id = $1 AND user_id = $2`

	var t model.Task
	err := r.db.QueryRow(q, id, userID).Scan(
		&t.ID, &t.UserID, &t.Title, &t.Description, &t.Status, &t.CreatedAt, &t.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get task by id: %w", err)
	}
	return &t, nil
}

func (r *postgresTaskRepo) Create(userID string, req model.CreateTaskRequest) (*model.Task, error) {
	const q = `
		INSERT INTO tasks (user_id, title, description, status)
		VALUES ($1, $2, $3, $4)
		RETURNING id, user_id, title, description, status, created_at, updated_at`

	var t model.Task
	err := r.db.QueryRow(q, userID, req.Title, req.Description, model.StatusPending).Scan(
		&t.ID, &t.UserID, &t.Title, &t.Description, &t.Status, &t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("create task: %w", err)
	}
	return &t, nil
}

func (r *postgresTaskRepo) Update(id, userID string, req model.UpdateTaskRequest) (*model.Task, error) {
	const q = `
		UPDATE tasks
		SET
			title       = COALESCE($1, title),
			description = COALESCE($2, description),
			status      = COALESCE($3, status),
			updated_at  = now()
		WHERE id = $4 AND user_id = $5
		RETURNING id, user_id, title, description, status, created_at, updated_at`

	var t model.Task
	err := r.db.QueryRow(q, req.Title, req.Description, req.Status, id, userID).Scan(
		&t.ID, &t.UserID, &t.Title, &t.Description, &t.Status, &t.CreatedAt, &t.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("update task: %w", err)
	}
	return &t, nil
}

func (r *postgresTaskRepo) Delete(id, userID string) error {
	const q = `DELETE FROM tasks WHERE id = $1 AND user_id = $2`
	result, err := r.db.Exec(q, id, userID)
	if err != nil {
		return fmt.Errorf("delete task: %w", err)
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}
