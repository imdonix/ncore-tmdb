package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"media-manager/internal/database"
)

var (
	ncoreHost string
	ncoreUser string
	ncorePass string
)

type Torrent struct {
	ID          string      `json:"ID"`
	Title       string      `json:"Title"`
	Key         string      `json:"Key"`         // added - present in API response 
	Type        string      `json:"Type"`
	Date        string      `json:"Date"`
	Seeders     int         `json:"Seeders"`
	Leechers    int         `json:"Leechers"`
	Completed   int         `json:"Completed"`   // kept but with correct tag (may be absent)
	DownloadURL string      `json:"Download"`    // changed - API uses "Download", not "download_url"
}

type SearchRequest struct {
	Pattern   string `json:"pattern"`
	Type      string `json:"type"`
	Where     string `json:"where"`
	SortBy    string `json:"sort_by"`
	SortOrder string `json:"sort_order"`
	Page      int    `json:"page"`
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
	url := fmt.Sprintf("%s/torrent/2128123", ncoreHost)
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

	// Read response body for debugging
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Decode response - API returns {"Torrents": [...]}
	var response struct {
		Torrents []Torrent `json:"Torrents"`
	}
	if err := json.Unmarshal(bodyBytes, &response); err != nil {
		preview := string(bodyBytes)
		if len(preview) > 300 {
			preview = preview[:300]
		}
		return nil, fmt.Errorf("failed to decode response: %s", preview)
	}

	return response.Torrents, nil
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
