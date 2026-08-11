package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"strings"
	"time"
)

var (
	qbitHost   string
	qbitUser   string
	qbitPass   string
	qbitClient *http.Client
)

const ncoreTagPrefix = "ncore:"

func InitQbit() {
	qbitHost = strings.TrimRight(os.Getenv("QBIT_HOST"), "/")
	qbitUser = os.Getenv("QBIT_USER")
	qbitPass = os.Getenv("QBIT_PASS")

	if qbitHost == "" {
		panic("QBIT_HOST must be set")
	}

	jar, _ := cookiejar.New(nil)
	qbitClient = &http.Client{
		Jar:     jar,
		Timeout: 30 * time.Second,
	}
}

func qbitOK(code int) bool {
	return code >= 200 && code < 300
}

func qbitLogin() error {
	loginURL := fmt.Sprintf("%s/api/v2/auth/login", qbitHost)
	data := url.Values{}

	if qbitUser != "" {
		data.Set("username", qbitUser)
	}
	if qbitPass != "" {
		data.Set("password", qbitPass)
	}

	req, err := http.NewRequest(http.MethodPost, loginURL, strings.NewReader(data.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Referer", qbitHost)

	resp, err := qbitClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	bodyStr := strings.TrimSpace(string(body))

	if !qbitOK(resp.StatusCode) {
		return fmt.Errorf("qbit login failed with status: %s (%s)", resp.Status, bodyStr)
	}
	if strings.EqualFold(bodyStr, "Fails.") || strings.Contains(bodyStr, "Fails") {
		return fmt.Errorf("qbit login failed: %s", bodyStr)
	}

	return nil
}

// NcoreTag builds the qBittorrent tag used to link back to an nCore torrent id.
func NcoreTag(ncoreID string) string {
	return ncoreTagPrefix + ncoreID
}

// ParseNcoreIDFromTags extracts ncore id from comma-separated qBittorrent tags.
func ParseNcoreIDFromTags(tags string) string {
	for _, part := range strings.Split(tags, ",") {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(part, ncoreTagPrefix) {
			return strings.TrimPrefix(part, ncoreTagPrefix)
		}
	}
	return ""
}

// AddTorrentOpts controls optional qBittorrent add behavior.
type AddTorrentOpts struct {
	// NcoreID is stored as tag ncore:{id} for linking back to the SPA.
	NcoreID string
	// SavePath is the download folder relative to qBit's default save path
	// (e.g. "Show.Name.S01"). Empty = qBit default.
	SavePath string
	// Rename renames the torrent / content root (e.g. "Show.Name.S01E01").
	Rename string
}

// AddTorrent uploads a .torrent file with optional tags, folder layout, and rename.
func AddTorrent(torrentData []byte, filename string, opts AddTorrentOpts) error {
	if err := qbitLogin(); err != nil {
		return fmt.Errorf("failed to login to qbit: %w", err)
	}

	addURL := fmt.Sprintf("%s/api/v2/torrents/add", qbitHost)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("torrents", filename)
	if err != nil {
		return err
	}
	if _, err := io.Copy(part, bytes.NewReader(torrentData)); err != nil {
		return err
	}

	if opts.NcoreID != "" {
		_ = writer.WriteField("tags", NcoreTag(opts.NcoreID))
	}
	// Force manual path so season subfolders are respected (disable Auto TMM).
	if opts.SavePath != "" {
		_ = writer.WriteField("autoTMM", "false")
		_ = writer.WriteField("savepath", opts.SavePath)
		// Prefer a subfolder named after the torrent (after rename).
		_ = writer.WriteField("contentLayout", "Subfolder")
	}
	if opts.Rename != "" {
		_ = writer.WriteField("rename", opts.Rename)
	}

	if err := writer.Close(); err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, addURL, body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Referer", qbitHost)

	resp, err := qbitClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if !qbitOK(resp.StatusCode) {
		return fmt.Errorf("failed to add torrent: %s - %s", resp.Status, string(respBody))
	}

	return nil
}

// FollowSeasonSavePath returns the season folder under the downloads root, e.g. "Show.Name.S01".
func FollowSeasonSavePath(seriesName string, season int) string {
	return fmt.Sprintf("%s.S%02d", sanitizeFolderName(seriesName), season)
}

// FollowEpisodeFolderName returns the episode (or pack) folder name inside the season dir.
// Episode 0 → season pack: "Show.Name.S01"
// Episode N → "Show.Name.S01E0N"
func FollowEpisodeFolderName(seriesName string, season, episode int) string {
	base := sanitizeFolderName(seriesName)
	if episode <= 0 {
		return fmt.Sprintf("%s.S%02d", base, season)
	}
	return fmt.Sprintf("%s.S%02dE%02d", base, season, episode)
}

func sanitizeFolderName(name string) string {
	name = strings.TrimSpace(name)
	// Match common release-style folders: spaces → dots, strip path-hostile chars
	repl := strings.NewReplacer(
		" ", ".", "/", ".", "\\", ".", ":", ".",
		"?", "", "*", "", "\"", "", "<", "", ">", "", "|", "",
		"'", "",
	)
	name = repl.Replace(name)
	// Collapse repeated dots
	for strings.Contains(name, "..") {
		name = strings.ReplaceAll(name, "..", ".")
	}
	name = strings.Trim(name, ".")
	if name == "" {
		return "Unknown"
	}
	return name
}

// QbitTorrent is a subset of qBittorrent's torrents/info payload + ncore link.
type QbitTorrent struct {
	Hash         string  `json:"hash"`
	Name         string  `json:"name"`
	Progress     float64 `json:"progress"` // 0..1
	State        string  `json:"state"`
	Size         int64   `json:"size"`
	Downloaded   int64   `json:"downloaded"`
	Uploaded     int64   `json:"uploaded"`
	Dlspeed      int64   `json:"dlspeed"`
	Upspeed      int64   `json:"upspeed"`
	Eta          int64   `json:"eta"`
	Ratio        float64 `json:"ratio"`
	Tags         string  `json:"tags"`
	Category     string  `json:"category"`
	SavePath     string  `json:"save_path"`
	AddedOn      int64   `json:"added_on"`
	CompletionOn int64   `json:"completion_on"`
	NumSeeds     int     `json:"num_seeds"`
	NumLeechs    int     `json:"num_leechs"`

	// Enriched by our API (not from qbit raw JSON alone)
	NcoreID string `json:"ncoreId,omitempty"`
}

func GetTorrentsStatus() ([]QbitTorrent, error) {
	if err := qbitLogin(); err != nil {
		return nil, err
	}

	infoURL := fmt.Sprintf("%s/api/v2/torrents/info", qbitHost)
	req, err := http.NewRequest(http.MethodGet, infoURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Referer", qbitHost)

	resp, err := qbitClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if !qbitOK(resp.StatusCode) {
		return nil, fmt.Errorf("failed to get torrents info: %s", resp.Status)
	}

	var torrents []QbitTorrent
	if err := json.NewDecoder(resp.Body).Decode(&torrents); err != nil {
		return nil, err
	}

	for i := range torrents {
		torrents[i].NcoreID = ParseNcoreIDFromTags(torrents[i].Tags)
	}

	return torrents, nil
}

// GetTorrentByNcoreID finds a qBittorrent entry tagged with ncore:{id}.
func GetTorrentByNcoreID(ncoreID string) (*QbitTorrent, error) {
	list, err := GetTorrentsStatus()
	if err != nil {
		return nil, err
	}
	tag := NcoreTag(ncoreID)
	for i := range list {
		if list[i].NcoreID == ncoreID || strings.Contains(list[i].Tags, tag) {
			t := list[i]
			return &t, nil
		}
	}
	// Fallback: match name containing the id (legacy adds without tags)
	for i := range list {
		if strings.Contains(list[i].Name, ncoreID) {
			t := list[i]
			t.NcoreID = ncoreID
			return &t, nil
		}
	}
	return nil, nil
}

// EnrichNcoreIDFromName is used by API layer when tags are missing.
func EnrichNcoreID(t *QbitTorrent, ncoreID string) {
	if t != nil && t.NcoreID == "" && ncoreID != "" {
		t.NcoreID = ncoreID
	}
}

// DeleteTorrent removes a torrent from qBittorrent. When deleteFiles is true,
// downloaded content on disk is removed as well.
func DeleteTorrent(hash string, deleteFiles bool) error {
	if hash == "" {
		return fmt.Errorf("missing torrent hash")
	}
	if err := qbitLogin(); err != nil {
		return fmt.Errorf("failed to login to qbit: %w", err)
	}

	delURL := fmt.Sprintf("%s/api/v2/torrents/delete", qbitHost)
	data := url.Values{}
	data.Set("hashes", hash)
	if deleteFiles {
		data.Set("deleteFiles", "true")
	} else {
		data.Set("deleteFiles", "false")
	}

	req, err := http.NewRequest(http.MethodPost, delURL, strings.NewReader(data.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Referer", qbitHost)

	resp, err := qbitClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if !qbitOK(resp.StatusCode) {
		return fmt.Errorf("failed to delete torrent: %s - %s", resp.Status, string(respBody))
	}

	return nil
}
