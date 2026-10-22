package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/joho/godotenv"

	"stockflow/backend/internal/handler"
	"stockflow/backend/internal/repository"
)

func main() {
	log.Println("Starting Stockflow API service...")

	// Load .env (non-fatal if not found — docker sets env vars directly)
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, reading env vars from environment")
	}

	// Build DSN from environment with sensible local defaults
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		getEnv("DB_HOST", "localhost"),
		getEnv("DB_PORT", "5432"),
		getEnv("DB_USER", "postgres"),
		getEnv("DB_PASSWORD", "1234"),
		getEnv("DB_NAME", "stockflow_db"),
	)

	// Open DB connection
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		log.Fatalf("Failed to open DB: %v", err)
	}
	defer db.Close()

	// Connection pool settings
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Retry ping (docker compose: DB may not be ready immediately)
	for i := range 10 {
		if err := db.Ping(); err == nil {
			break
		} else if i == 9 {
			log.Fatalf("Failed to connect to PostgreSQL after 10 attempts: %v", err)
		} else {
			log.Printf("DB not ready, retrying... (%d/10)", i+1)
			time.Sleep(2 * time.Second)
		}
	}
	log.Println("Connected to PostgreSQL successfully!")

	// Auto-apply migrations and seed data if needed
	runMigrations(db)

	// Init repositories
	productRepo := repository.NewProductRepository(db)
	orderRepo := repository.NewOrderRepository(db)

	// Init handlers
	productHandler := handler.NewProductHandler(productRepo)
	orderHandler := handler.NewOrderHandler(orderRepo)

	// Setup router
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)
	r.Use(corsMiddleware)

	r.Route("/api", func(r chi.Router) {
		// Products
		r.Get("/products", productHandler.List)
		r.Post("/products", productHandler.Create)
		r.Put("/products/{id}", productHandler.Update)
		r.Delete("/products/{id}", productHandler.Delete)
		r.Post("/products/{id}/stock", productHandler.AdjustStock)
		r.Get("/products/{id}/movements", productHandler.ListMovements)

		// Orders
		r.Get("/orders", orderHandler.List)
		r.Post("/orders", orderHandler.Create)
		r.Get("/orders/{id}", orderHandler.Get)
		r.Post("/orders/{id}/cancel", orderHandler.Cancel)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server listening on :%s", port)
	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

// corsMiddleware allows requests from the Vite dev server.
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// runMigrations automatically initializes the schema and seed data if the products table does not exist.
func runMigrations(db *sql.DB) {
	var exists bool
	_ = db.QueryRow("SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_schema = 'public' AND table_name = 'products')").Scan(&exists)
	if !exists {
		log.Println("Database tables not found. Applying migrations...")
		schemaCandidates := []string{"migrations/001_init_schema.sql", "../migrations/001_init_schema.sql", "backend/migrations/001_init_schema.sql"}
		for _, p := range schemaCandidates {
			if content, err := os.ReadFile(p); err == nil {
				if _, err := db.Exec(string(content)); err != nil {
					log.Printf("Failed to execute %s: %v", p, err)
				} else {
					log.Printf("Executed %s successfully", p)
				}
				break
			}
		}

		seedCandidates := []string{"migrations/002_seed.sql", "../migrations/002_seed.sql", "backend/migrations/002_seed.sql"}
		for _, p := range seedCandidates {
			if content, err := os.ReadFile(p); err == nil {
				if _, err := db.Exec(string(content)); err != nil {
					log.Printf("Failed to execute %s: %v", p, err)
				} else {
					log.Printf("Executed %s successfully", p)
				}
				break
			}
		}
	}
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}