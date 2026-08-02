package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"ncore-tmdb/internal/database"
)

var (
	ncoreHost string
	ncoreUser string
	ncorePass string
)

type Torrent struct {
	ID          string `json:"ID"`
	Title       string `json:"Title"`
	Key         string `json:"Key"`
	Size        any    `json:"Size,omitempty"`
	Type        string `json:"Type"`
	Date        string `json:"Date"`
	Seeders     int    `json:"Seeders"`
	Leechers    int    `json:"Leechers"`
	DownloadURL string `json:"Download"`
	URL         string `json:"URL,omitempty"`
	Extra       any    `json:"Extra,omitempty"`
}

type SearchRequest struct {
	Pattern   string `json:"pattern"`
	Type      string `json:"type"`
	Where     string `json:"where"`
	SortBy    string `json:"sort_by"`
	SortOrder string `json:"sort_order"`
	Page      int    `json:"page"`
}

type SearchResult struct {
	Torrents   []Torrent `json:"Torrents"`
	NumOfPages int       `json:"NumOfPages"`
}

// SearchTypeOption describes a filter category for the UI.
type SearchTypeOption struct {
	Value  string `json:"value"`
	Label  string `json:"label"`
	Group  string `json:"group"`
}

func InitNCore() {
	ncoreHost = os.Getenv("NCORE_HOST")
	ncoreUser = os.Getenv("NCORE_USER")
	ncorePass = os.Getenv("NCORE_PASS")

	if ncoreHost == "" || ncoreUser == "" || ncorePass == "" {
		fmt.Println("Warning: NCORE_HOST, NCORE_USER, or NCORE_PASS not set")
		return
	}

	token := getToken()
	if token == "" || !verifyToken() {
		err := login()
		if err != nil {
			fmt.Printf("Failed to login to NCore: %v\n", err)
		}
	}
}

func getToken() string {
	token, _ := database.GetContentKV(0, "system", "ncore_token")
	return token
}

func login() error {
	url := fmt.Sprintf("%s/login", ncoreHost)
	payload := map[string]string{
		"username": ncoreUser,
		"password": ncorePass,
	}
	body, _ := json.Marshal(payload)

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("login failed with status: %s", resp.Status)
	}

	var result struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}

	database.SetContentKV(0, "system", "ncore_token", result.Token)
	return nil
}

func verifyToken() bool {
	token := getToken()
	if token == "" {
		return false
	}
	url := fmt.Sprintf("%s/verify", ncoreHost)
	httpReq, _ := http.NewRequest("GET", url, nil)
	httpReq.Header.Set("X-Ncore-Auth", token)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}

func ensureValidToken() error {
	if !verifyToken() {
		return login()
	}
	return nil
}

func SearchNCore(req SearchRequest) ([]Torrent, error) {
	result, err := SearchNCoreFull(req)
	if err != nil {
		return nil, err
	}
	return result.Torrents, nil
}

func SearchNCoreFull(req SearchRequest) (*SearchResult, error) {
	if err := ensureValidToken(); err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/search", ncoreHost)
	body, _ := json.Marshal(req)

	httpReq, _ := http.NewRequest("POST", url, bytes.NewBuffer(body))
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-Ncore-Auth", getToken())

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("search failed with status: %s", resp.Status)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var response SearchResult
	if err := json.Unmarshal(bodyBytes, &response); err != nil {
		preview := string(bodyBytes)
		if len(preview) > 300 {
			preview = preview[:300]
		}
		return nil, fmt.Errorf("failed to decode response: %s", preview)
	}

	if response.Torrents == nil {
		response.Torrents = []Torrent{}
	}

	return &response, nil
}

func GetTorrentDetails(id string) (json.RawMessage, error) {
	url := fmt.Sprintf("%s/torrent/%s", ncoreHost, id)
	return doGet(url)
}

func DownloadTorrent(id string) ([]byte, error) {
	if err := ensureValidToken(); err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/torrent/%s/download", ncoreHost, id)

	httpReq, _ := http.NewRequest("GET", url, nil)
	httpReq.Header.Set("X-Ncore-Auth", getToken())

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download failed with status: %s", resp.Status)
	}

	return io.ReadAll(resp.Body)
}

func GetActivity() (json.RawMessage, error) {
	url := fmt.Sprintf("%s/activity", ncoreHost)
	return doGet(url)
}

func GetRecommended() (json.RawMessage, error) {
	url := fmt.Sprintf("%s/recommended", ncoreHost)
	return doGet(url)
}

func doGet(url string) (json.RawMessage, error) {
	if err := ensureValidToken(); err != nil {
		return nil, err
	}

	httpReq, _ := http.NewRequest("GET", url, nil)
	httpReq.Header.Set("X-Ncore-Auth", getToken())

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("request failed with status: %s", resp.Status)
	}

	return io.ReadAll(resp.Body)
}

func GetTorrentsNCore() string {
	return fmt.Sprintf("NCore service initialized at %s", ncoreHost)
}

// SearchTypes returns UI-friendly category options matching ncore-go SearchParamType.
func SearchTypes() []SearchTypeOption {
	return []SearchTypeOption{
		{Value: "all_own", Label: "All", Group: "General"},

		{Value: "hd_hun", Label: "HD (Hun)", Group: "Movies"},
		{Value: "hd", Label: "HD", Group: "Movies"},
		{Value: "xvid_hun", Label: "SD (Hun)", Group: "Movies"},
		{Value: "xvid", Label: "SD", Group: "Movies"},
		{Value: "dvd_hun", Label: "DVD (Hun)", Group: "Movies"},
		{Value: "dvd", Label: "DVD", Group: "Movies"},
		{Value: "dvd9_hun", Label: "DVD9 (Hun)", Group: "Movies"},
		{Value: "dvd9", Label: "DVD9", Group: "Movies"},

		{Value: "hdser_hun", Label: "HD Series (Hun)", Group: "Series"},
		{Value: "hdser", Label: "HD Series", Group: "Series"},
		{Value: "xvidser_hun", Label: "SD Series (Hun)", Group: "Series"},
		{Value: "xvidser", Label: "SD Series", Group: "Series"},
		{Value: "dvdser_hun", Label: "DVD Series (Hun)", Group: "Series"},
		{Value: "dvdser", Label: "DVD Series", Group: "Series"},

		{Value: "mp3_hun", Label: "MP3 (Hun)", Group: "Music"},
		{Value: "mp3", Label: "MP3", Group: "Music"},
		{Value: "lossless_hun", Label: "Lossless (Hun)", Group: "Music"},
		{Value: "lossless", Label: "Lossless", Group: "Music"},
		{Value: "clip", Label: "Music Video", Group: "Music"},

		{Value: "game_iso", Label: "Game ISO", Group: "Games"},
		{Value: "game_rip", Label: "Game Rip", Group: "Games"},
		{Value: "console", Label: "Console", Group: "Games"},

		{Value: "ebook_hun", Label: "eBook (Hun)", Group: "Books"},
		{Value: "ebook", Label: "eBook", Group: "Books"},

		{Value: "iso", Label: "Program ISO", Group: "Apps"},
		{Value: "misc", Label: "Program Misc", Group: "Apps"},
		{Value: "mobil", Label: "Mobile", Group: "Apps"},
	}
}
