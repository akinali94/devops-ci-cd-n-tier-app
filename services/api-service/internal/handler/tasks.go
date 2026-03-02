package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"task-manager/api-service/internal/middleware"
	"task-manager/api-service/internal/model"
	"task-manager/api-service/internal/service"
)

type TaskHandler struct {
	svc service.TaskService
}

func NewTaskHandler(svc service.TaskService) *TaskHandler {
	return &TaskHandler{svc: svc}
}

// RegisterRoutes registers all task routes onto the provided mux.
// The auth middleware is applied to the entire sub-mux.
func (h *TaskHandler) RegisterRoutes(mux *http.ServeMux, auth *middleware.AuthClient) {
	mux.HandleFunc("GET /health", h.health)

	// Authenticated routes
	mux.Handle("GET /tasks", auth.Authenticate(http.HandlerFunc(h.listTasks)))
	mux.Handle("POST /tasks", auth.Authenticate(http.HandlerFunc(h.createTask)))
	mux.Handle("GET /tasks/{id}", auth.Authenticate(http.HandlerFunc(h.getTask)))
	mux.Handle("PUT /tasks/{id}", auth.Authenticate(http.HandlerFunc(h.updateTask)))
	mux.Handle("DELETE /tasks/{id}", auth.Authenticate(http.HandlerFunc(h.deleteTask)))
}

func (h *TaskHandler) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *TaskHandler) listTasks(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.UserIDFromContext(r.Context())
	tasks, err := h.svc.ListTasks(userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch tasks")
		return
	}
	writeJSON(w, http.StatusOK, tasks)
}

func (h *TaskHandler) getTask(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.UserIDFromContext(r.Context())
	id := r.PathValue("id")

	task, err := h.svc.GetTask(id, userID)
	if errors.Is(err, service.ErrNotFound) {
		writeError(w, http.StatusNotFound, "task not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch task")
		return
	}
	writeJSON(w, http.StatusOK, task)
}

func (h *TaskHandler) createTask(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.UserIDFromContext(r.Context())

	var req model.CreateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	task, err := h.svc.CreateTask(userID, req)
	if err != nil {
		if err.Error() == "title is required" {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to create task")
		return
	}
	writeJSON(w, http.StatusCreated, task)
}

func (h *TaskHandler) updateTask(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.UserIDFromContext(r.Context())
	id := r.PathValue("id")

	var req model.UpdateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	task, err := h.svc.UpdateTask(id, userID, req)
	if errors.Is(err, service.ErrNotFound) {
		writeError(w, http.StatusNotFound, "task not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, task)
}

func (h *TaskHandler) deleteTask(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.UserIDFromContext(r.Context())
	id := r.PathValue("id")

	err := h.svc.DeleteTask(id, userID)
	if errors.Is(err, service.ErrNotFound) {
		writeError(w, http.StatusNotFound, "task not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete task")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
