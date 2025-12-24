package main

import (
	"fmt"

	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"
)

type DatabaseService struct {
	db *sqlx.DB
}

type PreviousGif struct {
	ID         int64  `db:"id" json:"id"`
	URL        string `db:"url" json:"url"`
	PreviewURL string `db:"preview_url" json:"previewUrl"`
	Timestamp  string `db:"timestamp" json:"timestamp"`
}

func NewDatabaseService(dbPath string) (*DatabaseService, error) {
	db, err := sqlx.Connect("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %v", err)
	}

	service := &DatabaseService{db: db}

	// Initialize tables
	if err := service.initTables(); err != nil {
		return nil, fmt.Errorf("failed to initialize tables: %v", err)
	}

	return service, nil
}

func (ds *DatabaseService) initTables() error {
	schema := `
    CREATE TABLE IF NOT EXISTS previous_gifs (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        url TEXT NOT NULL,
        preview_url TEXT,
        timestamp DATETIME DEFAULT CURRENT_TIMESTAMP
    );

    CREATE UNIQUE INDEX IF NOT EXISTS idx_previous_gifs_url ON previous_gifs(url);
    CREATE INDEX IF NOT EXISTS idx_previous_gifs_timestamp ON previous_gifs(timestamp);
    `

	_, err := ds.db.Exec(schema)
	if err != nil {
		return fmt.Errorf("failed to create schema: %v", err)
	}

	return nil
}

func (ds *DatabaseService) SavePreviousGif(url, previewURL string) error {
	// Start a transaction to ensure consistency
	tx, err := ds.db.Beginx()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %v", err)
	}
	defer tx.Rollback()

	// Check if we already have this URL (to avoid counting it in the limit)
	var existingCount int
	err = tx.Get(&existingCount, "SELECT COUNT(*) FROM previous_gifs WHERE url = ?", url)
	if err != nil {
		return fmt.Errorf("failed to check existing URL: %v", err)
	}

	// If this is a new URL, check if we need to remove old entries
	if existingCount == 0 {
		var totalCount int
		err = tx.Get(&totalCount, "SELECT COUNT(*) FROM previous_gifs")
		if err != nil {
			return fmt.Errorf("failed to count total gifs: %v", err)
		}

		// If we have 30 or more, delete the oldest entries to make room for 1 new entry
		if totalCount >= 30 {
			_, err = tx.Exec(`
				DELETE FROM previous_gifs 
				WHERE id IN (
					SELECT id FROM previous_gifs 
					ORDER BY timestamp ASC 
					LIMIT ?
				)`, totalCount-29) // Keep 29, delete the rest to make room for 1 new
			if err != nil {
				return fmt.Errorf("failed to delete old entries: %v", err)
			}
		}
	}

	// Insert new or update existing
	_, err = tx.Exec(`
        INSERT INTO previous_gifs (url, preview_url, timestamp) 
        VALUES (?, ?, CURRENT_TIMESTAMP)
        ON CONFLICT(url) DO UPDATE SET 
            preview_url = excluded.preview_url,
            timestamp = CURRENT_TIMESTAMP`,
		url, previewURL)

	if err != nil {
		return fmt.Errorf("failed to save previous gif: %v", err)
	}

	// Commit the transaction
	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("failed to commit transaction: %v", err)
	}

	return nil
}

func (ds *DatabaseService) GetPreviousGifs(limit int) ([]PreviousGif, error) {
	var gifs []PreviousGif
	err := ds.db.Select(&gifs, `
        SELECT id, url, preview_url, timestamp 
        FROM previous_gifs 
        ORDER BY timestamp DESC 
        LIMIT ?`, limit)

	return gifs, err
}

func (ds *DatabaseService) Close() error {
	return ds.db.Close()
}
