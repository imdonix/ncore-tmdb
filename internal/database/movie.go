package database

import "database/sql"

// MovieDownload is the last movie release sent to qBittorrent for a TMDB movie.
type MovieDownload struct {
	ID             int64  `json:"id"`
	TMDBID         int    `json:"tmdbId"`
	NCoreTorrentID string `json:"ncoreTorrentId"`
	TorrentTitle   string `json:"torrentTitle"`
	Quality        string `json:"quality"`
	QbitHash       string `json:"qbitHash,omitempty"`
	Status         string `json:"status"` // downloading | completed | missing
	CreatedAt      string `json:"createdAt"`
	UpdatedAt      string `json:"updatedAt"`
}

func CreateMovieTables() error {
	_, err := DB.Exec(`
	CREATE TABLE IF NOT EXISTS movie_download (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		tmdb_id INTEGER NOT NULL UNIQUE,
		ncore_torrent_id TEXT NOT NULL,
		torrent_title TEXT,
		quality TEXT NOT NULL,
		qbit_hash TEXT,
		status TEXT NOT NULL DEFAULT 'downloading',
		created_at TEXT NOT NULL,
		updated_at TEXT NOT NULL
	);
	CREATE INDEX IF NOT EXISTS idx_movie_download_tmdb ON movie_download(tmdb_id);
	`)
	return err
}

func GetMovieDownload(tmdbID int) (*MovieDownload, error) {
	row := DB.QueryRow(`
		SELECT id, tmdb_id, ncore_torrent_id, COALESCE(torrent_title,''), quality,
		       COALESCE(qbit_hash,''), status, created_at, updated_at
		FROM movie_download WHERE tmdb_id=?`, tmdbID,
	)
	var m MovieDownload
	err := row.Scan(&m.ID, &m.TMDBID, &m.NCoreTorrentID, &m.TorrentTitle, &m.Quality,
		&m.QbitHash, &m.Status, &m.CreatedAt, &m.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &m, nil
}

func UpsertMovieDownload(m *MovieDownload) error {
	now := nowISO()
	if m.CreatedAt == "" {
		m.CreatedAt = now
	}
	m.UpdatedAt = now
	if m.Status == "" {
		m.Status = "downloading"
	}
	res, err := DB.Exec(`
		INSERT INTO movie_download (tmdb_id, ncore_torrent_id, torrent_title, quality, qbit_hash, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(tmdb_id) DO UPDATE SET
			ncore_torrent_id=excluded.ncore_torrent_id,
			torrent_title=excluded.torrent_title,
			quality=excluded.quality,
			qbit_hash=excluded.qbit_hash,
			status=excluded.status,
			updated_at=excluded.updated_at`,
		m.TMDBID, m.NCoreTorrentID, m.TorrentTitle, m.Quality, nullStr(m.QbitHash), m.Status, m.CreatedAt, m.UpdatedAt,
	)
	if err != nil {
		return err
	}
	if m.ID == 0 {
		m.ID, _ = res.LastInsertId()
	}
	return nil
}
