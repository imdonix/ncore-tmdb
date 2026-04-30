package database

import (
	"database/sql"
)

type Content struct {
	TMDBID int
	Name   string
}

func CreateContentTable() error {
	query := `
	CREATE TABLE IF NOT EXISTS content (
		tmdb_id INTEGER PRIMARY KEY,
		name TEXT
	)`
	_, err := DB.Exec(query)
	return err
}

func InsertContent(tmdbID int, name string) error {
	_, err := DB.Exec("INSERT OR REPLACE INTO content (tmdb_id, name) VALUES (?, ?)", tmdbID, name)
	return err
}

func GetContent(tmdbID int) (*Content, error) {
	row := DB.QueryRow("SELECT tmdb_id, name FROM content WHERE tmdb_id = ?", tmdbID)
	c := &Content{}
	err := row.Scan(&c.TMDBID, &c.Name)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return c, err
}

func ListContent() ([]Content, error) {
	rows, err := DB.Query("SELECT tmdb_id, name FROM content")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []Content
	for rows.Next() {
		var c Content
		if err := rows.Scan(&c.TMDBID, &c.Name); err != nil {
			return nil, err
		}
		list = append(list, c)
	}
	return list, nil
}
