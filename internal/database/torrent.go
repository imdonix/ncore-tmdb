package database

type Torrent struct {
	ID          string `json:"ID"`
	Title       string `json:"Title"`
	Key         string `json:"Key"`
	Type        string `json:"Type"`
	Date        string `json:"Date"`
	Seeders     int    `json:"Seeders"`
	Leechers    int    `json:"Leechers"`
	Completed   int    `json:"Completed"`
	DownloadURL string `json:"Download"`
	Provider    string `json:"Provider"`
	TMDBID      int    `json:"-"`
	ContentType string `json:"-"`
}

func CreateTorrentTable() error {
	query := `
	CREATE TABLE IF NOT EXISTS torrent (
		id TEXT NOT NULL,
		title TEXT,
		key TEXT,
		type TEXT,
		date TEXT,
		seeders INTEGER,
		leechers INTEGER,
		completed INTEGER,
		download_url TEXT,
		provider TEXT,
		tmdb_id INTEGER NOT NULL,
		content_type TEXT NOT NULL,
		PRIMARY KEY (id, tmdb_id, content_type)
	)`
	_, err := DB.Exec(query)
	return err
}

func InsertTorrent(t *Torrent) error {
	_, err := DB.Exec("INSERT OR REPLACE INTO torrent (id, title, key, type, date, seeders, leechers, completed, download_url, provider, tmdb_id, content_type) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		t.ID, t.Title, t.Key, t.Type, t.Date, t.Seeders, t.Leechers, t.Completed, t.DownloadURL, t.Provider, t.TMDBID, t.ContentType)
	return err
}

func GetTorrentsByContent(tmdbID int, contentType string) ([]Torrent, error) {
	rows, err := DB.Query("SELECT id, title, key, type, date, seeders, leechers, completed, download_url, provider, tmdb_id, content_type FROM torrent WHERE tmdb_id = ? AND content_type = ?", tmdbID, contentType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []Torrent
	for rows.Next() {
		var t Torrent
		if err := rows.Scan(&t.ID, &t.Title, &t.Key, &t.Type, &t.Date, &t.Seeders, &t.Leechers, &t.Completed, &t.DownloadURL, &t.Provider, &t.TMDBID, &t.ContentType); err != nil {
			return nil, err
		}
		list = append(list, t)
	}
	return list, nil
}
