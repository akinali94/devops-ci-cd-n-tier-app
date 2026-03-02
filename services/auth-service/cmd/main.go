package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"

	"task-manager/auth-service/internal/handler"
	"task-manager/auth-service/internal/repository"
	"task-manager/auth-service/internal/service"
)

func main() {
	_ = godotenv.Load()

	dbURL := mustEnv("DB_URL")
	jwtSecret := mustEnv("JWT_SECRET")
	port := getEnv("PORT", "8081")

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("ping db: %v", err)
	}
	log.Println("connected to database")

	repo := repository.NewPostgresUserRepo(db)
	svc := service.NewAuthService(repo, jwtSecret)
	h := handler.NewAuthHandler(svc)

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	addr := fmt.Sprintf(":%s", port)
	log.Printf("auth-service listening on %s", addr)
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
