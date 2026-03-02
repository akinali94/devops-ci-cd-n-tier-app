package service

import (
	"database/sql"
	"errors"
	"fmt"

	"task-manager/api-service/internal/model"
	"task-manager/api-service/internal/repository"
)

var ErrNotFound = errors.New("task not found")

type TaskService interface {
	ListTasks(userID string) ([]model.Task, error)
	GetTask(id, userID string) (*model.Task, error)
	CreateTask(userID string, req model.CreateTaskRequest) (*model.Task, error)
	UpdateTask(id, userID string, req model.UpdateTaskRequest) (*model.Task, error)
	DeleteTask(id, userID string) error
}

type taskService struct {
	repo repository.TaskRepository
}

func NewTaskService(repo repository.TaskRepository) TaskService {
	return &taskService{repo: repo}
}

func (s *taskService) ListTasks(userID string) ([]model.Task, error) {
	tasks, err := s.repo.GetAllByUser(userID)
	if err != nil {
		return nil, fmt.Errorf("list tasks: %w", err)
	}
	if tasks == nil {
		tasks = []model.Task{}
	}
	return tasks, nil
}

func (s *taskService) GetTask(id, userID string) (*model.Task, error) {
	task, err := s.repo.GetByID(id, userID)
	if err != nil {
		return nil, fmt.Errorf("get task: %w", err)
	}
	if task == nil {
		return nil, ErrNotFound
	}
	return task, nil
}

func (s *taskService) CreateTask(userID string, req model.CreateTaskRequest) (*model.Task, error) {
	if req.Title == "" {
		return nil, errors.New("title is required")
	}
	task, err := s.repo.Create(userID, req)
	if err != nil {
		return nil, fmt.Errorf("create task: %w", err)
	}
	return task, nil
}

func (s *taskService) UpdateTask(id, userID string, req model.UpdateTaskRequest) (*model.Task, error) {
	if req.Status != nil {
		switch *req.Status {
		case model.StatusPending, model.StatusInProgress, model.StatusDone:
		default:
			return nil, fmt.Errorf("invalid status %q: must be pending, in_progress, or done", *req.Status)
		}
	}
	task, err := s.repo.Update(id, userID, req)
	if err != nil {
		return nil, fmt.Errorf("update task: %w", err)
	}
	if task == nil {
		return nil, ErrNotFound
	}
	return task, nil
}

func (s *taskService) DeleteTask(id, userID string) error {
	err := s.repo.Delete(id, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	return err
}
