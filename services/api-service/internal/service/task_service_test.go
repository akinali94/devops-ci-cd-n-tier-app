package service_test

import (
	"database/sql"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"task-manager/api-service/internal/model"
	"task-manager/api-service/internal/service"
)

// mockTaskRepo is a testify mock for repository.TaskRepository.
type mockTaskRepo struct {
	mock.Mock
}

func (m *mockTaskRepo) GetAllByUser(userID string) ([]model.Task, error) {
	args := m.Called(userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]model.Task), args.Error(1)
}

func (m *mockTaskRepo) GetByID(id, userID string) (*model.Task, error) {
	args := m.Called(id, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Task), args.Error(1)
}

func (m *mockTaskRepo) Create(userID string, req model.CreateTaskRequest) (*model.Task, error) {
	args := m.Called(userID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Task), args.Error(1)
}

func (m *mockTaskRepo) Update(id, userID string, req model.UpdateTaskRequest) (*model.Task, error) {
	args := m.Called(id, userID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Task), args.Error(1)
}

func (m *mockTaskRepo) Delete(id, userID string) error {
	args := m.Called(id, userID)
	return args.Error(0)
}

// --- ListTasks ---

func TestListTasks_ReturnsTasks(t *testing.T) {
	repo := &mockTaskRepo{}
	svc := service.NewTaskService(repo)

	expected := []model.Task{{ID: "1", UserID: "u1", Title: "Test"}}
	repo.On("GetAllByUser", "u1").Return(expected, nil)

	tasks, err := svc.ListTasks("u1")
	assert.NoError(t, err)
	assert.Equal(t, expected, tasks)
	repo.AssertExpectations(t)
}

func TestListTasks_ReturnsEmptySliceWhenNone(t *testing.T) {
	repo := &mockTaskRepo{}
	svc := service.NewTaskService(repo)

	repo.On("GetAllByUser", "u1").Return(nil, nil)

	tasks, err := svc.ListTasks("u1")
	assert.NoError(t, err)
	assert.Empty(t, tasks)
	repo.AssertExpectations(t)
}

func TestListTasks_ReturnsErrorOnRepoFailure(t *testing.T) {
	repo := &mockTaskRepo{}
	svc := service.NewTaskService(repo)

	repo.On("GetAllByUser", "u1").Return(nil, errors.New("db error"))

	_, err := svc.ListTasks("u1")
	assert.Error(t, err)
	repo.AssertExpectations(t)
}

// --- GetTask ---

func TestGetTask_ReturnsTask(t *testing.T) {
	repo := &mockTaskRepo{}
	svc := service.NewTaskService(repo)

	expected := &model.Task{ID: "t1", UserID: "u1", Title: "Do laundry"}
	repo.On("GetByID", "t1", "u1").Return(expected, nil)

	task, err := svc.GetTask("t1", "u1")
	assert.NoError(t, err)
	assert.Equal(t, expected, task)
	repo.AssertExpectations(t)
}

func TestGetTask_ReturnsErrNotFoundWhenMissing(t *testing.T) {
	repo := &mockTaskRepo{}
	svc := service.NewTaskService(repo)

	repo.On("GetByID", "t99", "u1").Return(nil, nil)

	_, err := svc.GetTask("t99", "u1")
	assert.ErrorIs(t, err, service.ErrNotFound)
	repo.AssertExpectations(t)
}

// --- CreateTask ---

func TestCreateTask_Success(t *testing.T) {
	repo := &mockTaskRepo{}
	svc := service.NewTaskService(repo)

	req := model.CreateTaskRequest{Title: "Buy milk", Description: "2%"}
	expected := &model.Task{ID: "t2", UserID: "u1", Title: "Buy milk", Status: model.StatusPending}
	repo.On("Create", "u1", req).Return(expected, nil)

	task, err := svc.CreateTask("u1", req)
	assert.NoError(t, err)
	assert.Equal(t, expected, task)
	repo.AssertExpectations(t)
}

func TestCreateTask_RejectsEmptyTitle(t *testing.T) {
	repo := &mockTaskRepo{}
	svc := service.NewTaskService(repo)

	_, err := svc.CreateTask("u1", model.CreateTaskRequest{Title: ""})
	assert.Error(t, err)
	repo.AssertNotCalled(t, "Create")
}

// --- UpdateTask ---

func TestUpdateTask_RejectsInvalidStatus(t *testing.T) {
	repo := &mockTaskRepo{}
	svc := service.NewTaskService(repo)

	bad := "invalid"
	_, err := svc.UpdateTask("t1", "u1", model.UpdateTaskRequest{Status: &bad})
	assert.Error(t, err)
	repo.AssertNotCalled(t, "Update")
}

func TestUpdateTask_ReturnsErrNotFoundWhenMissing(t *testing.T) {
	repo := &mockTaskRepo{}
	svc := service.NewTaskService(repo)

	done := model.StatusDone
	repo.On("Update", "t99", "u1", model.UpdateTaskRequest{Status: &done}).Return(nil, nil)

	_, err := svc.UpdateTask("t99", "u1", model.UpdateTaskRequest{Status: &done})
	assert.ErrorIs(t, err, service.ErrNotFound)
	repo.AssertExpectations(t)
}

// --- DeleteTask ---

func TestDeleteTask_Success(t *testing.T) {
	repo := &mockTaskRepo{}
	svc := service.NewTaskService(repo)

	repo.On("Delete", "t1", "u1").Return(nil)

	err := svc.DeleteTask("t1", "u1")
	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestDeleteTask_ReturnsErrNotFoundWhenMissing(t *testing.T) {
	repo := &mockTaskRepo{}
	svc := service.NewTaskService(repo)

	repo.On("Delete", "t99", "u1").Return(sql.ErrNoRows)

	err := svc.DeleteTask("t99", "u1")
	assert.ErrorIs(t, err, service.ErrNotFound)
	repo.AssertExpectations(t)
}
