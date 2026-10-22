package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()

	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_NAME"),
	)

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		log.Fatalf("Failed to open DB: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to connect to DB: %v", err)
	}

	for _, file := range []string{"migrations/001_init_schema.sql", "migrations/002_seed.sql"} {
		content, err := os.ReadFile(file)
		if err != nil {
			log.Printf("Could not read %s: %v", file, err)
			continue
		}

		// Split on semicolons for clean individual execution
		stmts := strings.Split(string(content), ";")
		for _, stmt := range stmts {
			trimmed := strings.TrimSpace(stmt)
			if trimmed == "" {
				continue
			}
			if _, err := db.Exec(trimmed); err != nil {
				log.Printf("Warning executing statement in %s: %v", file, err)
			}
		}
		log.Printf("Applied %s successfully", file)
	}

	log.Println("Database migration and seeding completed!")
}
