package repository

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/joho/godotenv"
)

func getTestDB(t *testing.T) *sql.DB {
	t.Helper()

	// Try loading .env from backend directory
	_ = godotenv.Load("../../.env")
	_ = godotenv.Load("../.env")
	_ = godotenv.Load(".env")

	host := os.Getenv("DB_HOST")
	if host == "" {
		host = "localhost"
	}
	port := os.Getenv("DB_PORT")
	if port == "" {
		port = "5432"
	}
	user := os.Getenv("DB_USER")
	if user == "" {
		user = "postgres"
	}
	pass := os.Getenv("DB_PASSWORD")
	if pass == "" {
		pass = "1234"
	}
	dbname := os.Getenv("DB_NAME")
	if dbname == "" {
		dbname = "stockflow_db"
	}

	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable", host, port, user, pass, dbname)
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Skipf("Skipping DB test: unable to open DB: %v", err)
	}

	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(1 * time.Minute)

	if err := db.Ping(); err != nil {
		t.Skipf("Skipping DB test: PostgreSQL not reachable at %s:%s: %v", host, port, err)
	}

	// Apply schema migrations if tables don't exist
	migrationPaths := []string{
		"../../migrations/001_init_schema.sql",
		"../migrations/001_init_schema.sql",
		"migrations/001_init_schema.sql",
	}

	for _, mp := range migrationPaths {
		if abs, err := filepath.Abs(mp); err == nil {
			if content, err := os.ReadFile(abs); err == nil {
				_, _ = db.Exec(string(content))
				break
			}
		}
	}

	// Apply seed data if not seeded yet
	var seeded bool
	_ = db.QueryRow("SELECT EXISTS (SELECT 1 FROM products WHERE sku = 'SKU-001')").Scan(&seeded)
	if !seeded {
		seedPaths := []string{
			"../../migrations/002_seed.sql",
			"../migrations/002_seed.sql",
			"migrations/002_seed.sql",
		}
		for _, sp := range seedPaths {
			if abs, err := filepath.Abs(sp); err == nil {
				if content, err := os.ReadFile(abs); err == nil {
					_, _ = db.Exec(string(content))
					break
				}
			}
		}
	}

	return db
}
