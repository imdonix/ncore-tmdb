package database

import (
	"database/sql"
)

type Content struct {
	TMDBID      int
	Type        string
	Name        string
	ReleaseDate string
	Progress    string
}

func CreateContentTable() error {
	query := `
	CREATE TABLE IF NOT EXISTS content (
		tmdb_id INTEGER NOT NULL,
		type TEXT NOT NULL,
		name TEXT,
		release_date TEXT,
		progress TEXT,
		PRIMARY KEY (tmdb_id, type)
	)`
	_, err := DB.Exec(query)
	return err
}

func InsertContent(c *Content) error {
	_, err := DB.Exec("INSERT OR REPLACE INTO content (tmdb_id, type, name, release_date, progress) VALUES (?, ?, ?, ?, ?)",
		c.TMDBID, c.Type, c.Name, c.ReleaseDate, c.Progress)
	return err
}

func GetContent(tmdbID int, contentType string) (*Content, error) {
	row := DB.QueryRow("SELECT tmdb_id, type, name, release_date, progress FROM content WHERE tmdb_id = ? AND type = ?", tmdbID, contentType)
	c := &Content{}
	err := row.Scan(&c.TMDBID, &c.Type, &c.Name, &c.ReleaseDate, &c.Progress)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return c, err
}

func ListContent() ([]Content, error) {
	rows, err := DB.Query("SELECT tmdb_id, type, name, release_date, progress FROM content")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []Content
	for rows.Next() {
		var c Content
		if err := rows.Scan(&c.TMDBID, &c.Type, &c.Name, &c.ReleaseDate, &c.Progress); err != nil {
			return nil, err
		}
		list = append(list, c)
	}
	return list, nil
}
