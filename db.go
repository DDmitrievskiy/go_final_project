package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"
)

func InitDB() (*sqlx.DB, error) {

	dbFile := os.Getenv("TODO_DBFILE")

	if dbFile == "" {
		exePath, err := os.Executable()
		if err != nil {
			return nil, fmt.Errorf("ошибка получения пути: %w", err)
		}
		dbFile = filepath.Join(filepath.Dir(exePath), "scheduler.db")
	}

	install := false
	if _, err := os.Stat(dbFile); os.IsNotExist(err) {
		install = true
		log.Println("Инициализация новой БД...")
	}

	db, err := sqlx.Open("sqlite", dbFile)
	if err != nil {
		return nil, fmt.Errorf("ошибка подключения: %w", err)
	}

	if install {
		if err := createSchema(db); err != nil {
			return nil, fmt.Errorf("ошибка инициализации схемы: %w", err)
		}
	}

	return db, nil
}

func createSchema(db *sqlx.DB) error {

	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS scheduler (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			date TEXT NOT NULL,
			title TEXT NOT NULL,
			comment TEXT,
			repeat TEXT CHECK(LENGTH(repeat) <= 128))
			`); err != nil {
		return fmt.Errorf("ошибка создания таблицы: %w", err)
	}

	if _, err := db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_date 
		ON scheduler(date)
	`); err != nil {
		return fmt.Errorf("ошибка создания индекса: %w", err)
	}

	return nil
}
