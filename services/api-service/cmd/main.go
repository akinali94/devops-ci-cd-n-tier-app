package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"

	"task-manager/api-service/internal/handler"
	"task-manager/api-service/internal/middleware"
	"task-manager/api-service/internal/repository"
	"task-manager/api-service/internal/service"
)

func main() {
	// Load .env in development (ignored in production where env vars are set directly)
	_ = godotenv.Load()

	dbURL := mustEnv("DB_URL")
	authServiceURL := mustEnv("AUTH_SERVICE_URL")
	port := getEnv("PORT", "8090")

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("ping db: %v", err)
	}
	log.Println("connected to database")

	repo := repository.NewPostgresTaskRepo(db)
	svc := service.NewTaskService(repo)
	auth := middleware.NewAuthClient(authServiceURL)
	h := handler.NewTaskHandler(svc)

	mux := http.NewServeMux()
	h.RegisterRoutes(mux, auth)

	addr := fmt.Sprintf(":%s", port)
	log.Printf("api-service listening on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("required environment variable %q is not set", key)
	}
	return v
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
