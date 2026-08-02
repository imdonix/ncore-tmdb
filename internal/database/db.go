package database

import (
	"database/sql"
	"fmt"
	"log"
	"net/url"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

var DB *sql.DB

func Init(dbPath string) error {
	// WAL: concurrent readers + one writer.
	// busy_timeout: wait/retry instead of failing immediately on lock.
	dsn := fmt.Sprintf("file:%s?%s", dbPath, url.Values{
		"_pragma": {
			"busy_timeout(5000)",
			"journal_mode(WAL)",
			"synchronous(NORMAL)",
			"foreign_keys(ON)",
		},
	}.Encode())

	var err error
	DB, err = sql.Open("sqlite", dsn)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	// Do NOT use MaxOpenConns(1) — the follow checker holds the DB across many
	// queries and would block every HTTP handler for the whole check duration.
	// With WAL, a small pool is safe for this single-user app.
	DB.SetMaxOpenConns(8)
	DB.SetMaxIdleConns(4)
	DB.SetConnMaxLifetime(time.Hour)

	if err := DB.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	for _, pragma := range []string{
		`PRAGMA busy_timeout=5000`,
		`PRAGMA journal_mode=WAL`,
		`PRAGMA synchronous=NORMAL`,
		`PRAGMA foreign_keys=ON`,
	} {
		if _, err := DB.Exec(pragma); err != nil {
			log.Printf("database: pragma %q: %v", pragma, err)
		}
	}

	log.Println("Database connected:", dbPath, "(WAL, pool=8, busy_timeout=5s)")
	return nil
}

func Close() error {
	if DB != nil {
		return DB.Close()
	}
	return nil
}

// WithRetry runs fn, retrying a few times on SQLITE_BUSY.
func WithRetry(fn func() error) error {
	var err error
	for i := 0; i < 8; i++ {
		err = fn()
		if err == nil {
			return nil
		}
		msg := err.Error()
		if !strings.Contains(msg, "SQLITE_BUSY") && !strings.Contains(msg, "database is locked") {
			return err
		}
		time.Sleep(time.Duration(40*(i+1)) * time.Millisecond)
	}
	return err
}
