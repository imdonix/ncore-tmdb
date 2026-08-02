package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// Follow is a TV series the user wants auto-downloaded when episodes appear.
type Follow struct {
	ID             int64  `json:"id"`
	TMDBID         int    `json:"tmdbId"`
	Name           string `json:"name"`
	Year           string `json:"year"`
	Quality        string `json:"quality"` // 720p | 1080p
	SearchPattern  string `json:"searchPattern"`
	Status         string `json:"status"` // active | paused
	PosterPath     string `json:"posterPath,omitempty"`
	SkippedSeasons []int  `json:"skippedSeasons"`
	LastCheckAt    string `json:"lastCheckAt,omitempty"`
	LastError      string `json:"lastError,omitempty"`
	CreatedAt      string `json:"createdAt"`
	// Computed
	Wanted    int `json:"wanted,omitempty"`
	Found     int `json:"found,omitempty"`
	Completed int `json:"completed,omitempty"`
}

// FollowItem is one season pack (episode=0) or single episode acquisition.
type FollowItem struct {
	ID             int64  `json:"id"`
	FollowID       int64  `json:"followId"`
	Season         int    `json:"season"`
	Episode        int    `json:"episode"` // 0 = full season pack
	Status         string `json:"status"`  // wanted | found | downloading | completed | failed | cannot_find | skipped
	NcoreTorrentID string `json:"ncoreTorrentId,omitempty"`
	TorrentTitle   string `json:"torrentTitle,omitempty"`
	QbitHash       string `json:"qbitHash,omitempty"`
	CoveredBy      int64  `json:"coveredBy,omitempty"`
	UpdatedAt      string `json:"updatedAt"`
	CreatedAt      string `json:"createdAt"`
}

func CreateFollowTables() error {
	_, err := DB.Exec(`
	CREATE TABLE IF NOT EXISTS follow (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		tmdb_id INTEGER NOT NULL UNIQUE,
		name TEXT NOT NULL,
		year TEXT,
		quality TEXT NOT NULL DEFAULT '1080p',
		search_pattern TEXT NOT NULL,
		status TEXT NOT NULL DEFAULT 'active',
		poster_path TEXT,
		skipped_seasons TEXT DEFAULT '[]',
		last_check_at TEXT,
		last_error TEXT,
		created_at TEXT NOT NULL
	);
	CREATE TABLE IF NOT EXISTS follow_item (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		follow_id INTEGER NOT NULL,
		season INTEGER NOT NULL,
		episode INTEGER NOT NULL DEFAULT 0,
		status TEXT NOT NULL DEFAULT 'wanted',
		ncore_torrent_id TEXT,
		torrent_title TEXT,
		qbit_hash TEXT,
		covered_by INTEGER,
		updated_at TEXT NOT NULL,
		created_at TEXT NOT NULL,
		UNIQUE(follow_id, season, episode),
		FOREIGN KEY(follow_id) REFERENCES follow(id) ON DELETE CASCADE
	);
	CREATE INDEX IF NOT EXISTS idx_follow_item_follow ON follow_item(follow_id);
	`)
	if err != nil {
		return err
	}
	// Migrations for existing DBs
	_, _ = DB.Exec(`ALTER TABLE follow ADD COLUMN skipped_seasons TEXT DEFAULT '[]'`)
	return nil
}

func nowISO() string {
	return time.Now().UTC().Format(time.RFC3339)
}

func encodeSkipped(seasons []int) string {
	if seasons == nil {
		seasons = []int{}
	}
	b, _ := json.Marshal(seasons)
	return string(b)
}

func decodeSkipped(s string) []int {
	if s == "" {
		return []int{}
	}
	var out []int
	if err := json.Unmarshal([]byte(s), &out); err != nil || out == nil {
		return []int{}
	}
	return out
}

func InsertFollow(f *Follow) error {
	if f.CreatedAt == "" {
		f.CreatedAt = nowISO()
	}
	if f.Status == "" {
		f.Status = "active"
	}
	if f.SkippedSeasons == nil {
		f.SkippedSeasons = []int{}
	}
	res, err := DB.Exec(
		`INSERT INTO follow (tmdb_id, name, year, quality, search_pattern, status, poster_path, skipped_seasons, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		f.TMDBID, f.Name, f.Year, f.Quality, f.SearchPattern, f.Status, f.PosterPath, encodeSkipped(f.SkippedSeasons), f.CreatedAt,
	)
	if err != nil {
		return err
	}
	f.ID, _ = res.LastInsertId()
	return nil
}

func UpdateFollow(f *Follow) error {
	if f.SkippedSeasons == nil {
		f.SkippedSeasons = []int{}
	}
	_, err := DB.Exec(
		`UPDATE follow SET quality=?, search_pattern=?, status=?, poster_path=?, skipped_seasons=?, last_check_at=?, last_error=?, name=?, year=? WHERE id=?`,
		f.Quality, f.SearchPattern, f.Status, f.PosterPath, encodeSkipped(f.SkippedSeasons), f.LastCheckAt, f.LastError, f.Name, f.Year, f.ID,
	)
	return err
}

func DeleteFollow(id int64) error {
	tx, err := DB.Begin()
	if err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM follow_item WHERE follow_id=?`, id); err != nil {
		_ = tx.Rollback()
		return err
	}
	if _, err := tx.Exec(`DELETE FROM follow WHERE id=?`, id); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

const followSelect = `id, tmdb_id, name, year, quality, search_pattern, status, COALESCE(poster_path,''), COALESCE(skipped_seasons,'[]'), COALESCE(last_check_at,''), COALESCE(last_error,''), created_at`

func GetFollowByTMDB(tmdbID int) (*Follow, error) {
	row := DB.QueryRow(`SELECT `+followSelect+` FROM follow WHERE tmdb_id=?`, tmdbID)
	return scanFollow(row)
}

func GetFollowByID(id int64) (*Follow, error) {
	row := DB.QueryRow(`SELECT `+followSelect+` FROM follow WHERE id=?`, id)
	return scanFollow(row)
}

func ListFollows() ([]Follow, error) {
	rows, err := DB.Query(`SELECT ` + followSelect + ` FROM follow ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []Follow
	for rows.Next() {
		f, err := scanFollowRow(rows)
		if err != nil {
			return nil, err
		}
		_ = attachFollowCounts(f)
		list = append(list, *f)
	}
	if list == nil {
		list = []Follow{}
	}
	return list, nil
}

func ListActiveFollows() ([]Follow, error) {
	rows, err := DB.Query(`SELECT ` + followSelect + ` FROM follow WHERE status='active' ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []Follow
	for rows.Next() {
		f, err := scanFollowRow(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, *f)
	}
	return list, nil
}

func scanFollow(row *sql.Row) (*Follow, error) {
	var f Follow
	var skipped string
	err := row.Scan(&f.ID, &f.TMDBID, &f.Name, &f.Year, &f.Quality, &f.SearchPattern, &f.Status, &f.PosterPath, &skipped, &f.LastCheckAt, &f.LastError, &f.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	f.SkippedSeasons = decodeSkipped(skipped)
	_ = attachFollowCounts(&f)
	return &f, nil
}

type scannable interface {
	Scan(dest ...any) error
}

func scanFollowRow(row scannable) (*Follow, error) {
	var f Follow
	var skipped string
	err := row.Scan(&f.ID, &f.TMDBID, &f.Name, &f.Year, &f.Quality, &f.SearchPattern, &f.Status, &f.PosterPath, &skipped, &f.LastCheckAt, &f.LastError, &f.CreatedAt)
	if err != nil {
		return nil, err
	}
	f.SkippedSeasons = decodeSkipped(skipped)
	return &f, nil
}

func attachFollowCounts(f *Follow) error {
	return DB.QueryRow(`
		SELECT
			COALESCE(SUM(CASE WHEN episode > 0 AND status IN ('wanted','cannot_find','failed') THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN status IN ('found','downloading','completed') THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN status='completed' OR status='skipped' THEN 1 ELSE 0 END), 0)
		FROM follow_item WHERE follow_id=?`, f.ID,
	).Scan(&f.Wanted, &f.Found, &f.Completed)
}

func UpsertFollowItem(item *FollowItem) error {
	now := nowISO()
	if item.CreatedAt == "" {
		item.CreatedAt = now
	}
	item.UpdatedAt = now
	res, err := DB.Exec(`
		INSERT INTO follow_item (follow_id, season, episode, status, ncore_torrent_id, torrent_title, qbit_hash, covered_by, updated_at, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(follow_id, season, episode) DO UPDATE SET
			status=excluded.status,
			ncore_torrent_id=excluded.ncore_torrent_id,
			torrent_title=excluded.torrent_title,
			qbit_hash=excluded.qbit_hash,
			covered_by=excluded.covered_by,
			updated_at=excluded.updated_at
	`, item.FollowID, item.Season, item.Episode, item.Status, nullStr(item.NcoreTorrentID), nullStr(item.TorrentTitle), nullStr(item.QbitHash), nullInt64(item.CoveredBy), item.UpdatedAt, item.CreatedAt)
	if err != nil {
		return err
	}
	if item.ID == 0 {
		item.ID, _ = res.LastInsertId()
	}
	return nil
}

func nullStr(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func nullInt64(n int64) any {
	if n == 0 {
		return nil
	}
	return n
}

func ListFollowItems(followID int64) ([]FollowItem, error) {
	rows, err := DB.Query(`
		SELECT id, follow_id, season, episode, status, COALESCE(ncore_torrent_id,''), COALESCE(torrent_title,''), COALESCE(qbit_hash,''), COALESCE(covered_by,0), updated_at, created_at
		FROM follow_item WHERE follow_id=? ORDER BY season, episode`, followID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []FollowItem
	for rows.Next() {
		var it FollowItem
		if err := rows.Scan(&it.ID, &it.FollowID, &it.Season, &it.Episode, &it.Status, &it.NcoreTorrentID, &it.TorrentTitle, &it.QbitHash, &it.CoveredBy, &it.UpdatedAt, &it.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, it)
	}
	if list == nil {
		list = []FollowItem{}
	}
	return list, nil
}

func GetFollowItem(followID int64, season, episode int) (*FollowItem, error) {
	row := DB.QueryRow(`
		SELECT id, follow_id, season, episode, status, COALESCE(ncore_torrent_id,''), COALESCE(torrent_title,''), COALESCE(qbit_hash,''), COALESCE(covered_by,0), updated_at, created_at
		FROM follow_item WHERE follow_id=? AND season=? AND episode=?`, followID, season, episode,
	)
	var it FollowItem
	err := row.Scan(&it.ID, &it.FollowID, &it.Season, &it.Episode, &it.Status, &it.NcoreTorrentID, &it.TorrentTitle, &it.QbitHash, &it.CoveredBy, &it.UpdatedAt, &it.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &it, nil
}

func MarkEpisodesCoveredBySeason(followID int64, season int, packItemID int64) error {
	now := nowISO()
	_, err := DB.Exec(`
		UPDATE follow_item SET status='completed', covered_by=?, updated_at=?
		WHERE follow_id=? AND season=? AND episode > 0 AND status NOT IN ('skipped')`,
		packItemID, now, followID, season,
	)
	return err
}

func EnsureWantedEpisodes(followID int64, season, episodeCount int) error {
	now := nowISO()
	for ep := 1; ep <= episodeCount; ep++ {
		_, err := DB.Exec(`
			INSERT OR IGNORE INTO follow_item (follow_id, season, episode, status, updated_at, created_at)
			VALUES (?, ?, ?, 'wanted', ?, ?)`,
			followID, season, ep, now, now,
		)
		if err != nil {
			return fmt.Errorf("ensure S%02dE%02d: %w", season, ep, err)
		}
	}
	return nil
}

// MarkSeasonSkipped marks every episode (and a season-pack row) as skipped —
// treated like already owned so the checker ignores them.
func MarkSeasonSkipped(followID int64, season, episodeCount int) error {
	now := nowISO()
	// Pack row
	if err := UpsertFollowItem(&FollowItem{
		FollowID: followID, Season: season, Episode: 0, Status: "skipped",
	}); err != nil {
		return err
	}
	for ep := 1; ep <= episodeCount; ep++ {
		// Only force skip if not already downloading/completed with a real torrent
		existing, _ := GetFollowItem(followID, season, ep)
		if existing != nil {
			switch existing.Status {
			case "downloading", "found", "completed":
				if existing.NcoreTorrentID != "" || existing.CoveredBy > 0 {
					continue // keep real acquisitions
				}
			}
		}
		if err := UpsertFollowItem(&FollowItem{
			FollowID: followID, Season: season, Episode: ep, Status: "skipped",
			UpdatedAt: now, CreatedAt: now,
		}); err != nil {
			return err
		}
	}
	return nil
}

// UnskipSeason resets skipped items back to wanted (does not touch real downloads).
func UnskipSeason(followID int64, season int) error {
	now := nowISO()
	_, err := DB.Exec(`
		UPDATE follow_item SET status='wanted', updated_at=?
		WHERE follow_id=? AND season=? AND status='skipped'`,
		now, followID, season,
	)
	return err
}

// IsSeasonSkipped reports whether season is in the skip list.
func IsSeasonSkipped(skipped []int, season int) bool {
	for _, s := range skipped {
		if s == season {
			return true
		}
	}
	return false
}
