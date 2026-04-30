package database

import (
	"database/sql"
)

func CreateKVTable() error {
	query := `
	CREATE TABLE IF NOT EXISTS kv (
		tmdb_id INTEGER NOT NULL,
		type TEXT NOT NULL,
		key TEXT NOT NULL,
		value TEXT,
		PRIMARY KEY (tmdb_id, type, key)
	)`
	_, err := DB.Exec(query)
	return err
}

func SetContentKV(tmdbID int, contentType, key, value string) error {
	_, err := DB.Exec("INSERT OR REPLACE INTO kv (tmdb_id, type, key, value) VALUES (?, ?, ?, ?)", tmdbID, contentType, key, value)
	return err
}

func GetContentKV(tmdbID int, contentType, key string) (string, error) {
	var value string
	err := DB.QueryRow("SELECT value FROM kv WHERE tmdb_id = ? AND type = ? AND key = ?", tmdbID, contentType, key).Scan(&value)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return value, err
}

func GetAllContentKV(tmdbID int, contentType string) (map[string]string, error) {
	rows, err := DB.Query("SELECT key, value FROM kv WHERE tmdb_id = ? AND type = ?", tmdbID, contentType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]string)
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			return nil, err
		}
		result[k] = v
	}
	return result, nil
}
